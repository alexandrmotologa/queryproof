package metrics

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/ast"
)

// MetricValidationResult represents the outcome of checking a candidate query projection against a certified KPI.
type MetricValidationResult struct {
	MetricName        string   `json:"metric_name"`
	CertifiedFormula  string   `json:"certified_formula"`
	CandidateFormula  string   `json:"candidate_formula"`
	IsValid           bool     `json:"is_valid"`
	DivergenceScore   float64  `json:"divergence_score"` // 0.0 = identical, 1.0 = completely hallucinated
	DiscrepancyNotes  []string `json:"discrepancy_notes"`
	MissingFilters    []string `json:"missing_filters,omitempty"`
	InvalidDimensions []string `json:"invalid_dimensions,omitempty"`
}

// Validator validates query AST projections against the enterprise metric registry.
type Validator struct {
	registry *Registry
}

// NewValidator creates a validator instance.
func NewValidator(reg *Registry) *Validator {
	return &Validator{registry: reg}
}

// ValidateQueryMetrics checks all projections and clauses in stmt against certified metrics.
func (v *Validator) ValidateQueryMetrics(stmt *ast.SelectStmt) []MetricValidationResult {
	var results []MetricValidationResult
	if v.registry == nil || stmt == nil {
		return results
	}

	whereConds := ast.ExtractWhereConjunctions(stmt.Where)
	var whereStrs []string
	for _, cond := range whereConds {
		whereStrs = append(whereStrs, strings.ToLower(cond.ToSQL()))
	}

	// Inspect each projection
	for _, proj := range stmt.Projections {
		alias := proj.Alias
		if alias == "" {
			// Check if expression is a function call like SUM(amount)
			if fn, ok := proj.Expr.(*ast.FunctionCall); ok {
				alias = fn.Name
			}
		}

		matchedMetric := v.registry.FindMatchingMetric(alias)
		if matchedMetric == nil {
			continue
		}

		candFormula := proj.Expr.ToSQL()
		res := v.verifyMetric(matchedMetric, candFormula, whereStrs, stmt.GroupBy)
		results = append(results, res)
	}

	return results
}

func (v *Validator) verifyMetric(metric *Metric, candFormula string, whereStrs []string, groupBy []ast.Expr) MetricValidationResult {
	res := MetricValidationResult{
		MetricName:       metric.Name,
		CertifiedFormula: metric.CertifiedFormula,
		CandidateFormula: candFormula,
		IsValid:          true,
	}

	// 1. Formula AST comparison
	normCert := normalizeExprString(metric.CertifiedFormula)
	normCand := normalizeExprString(candFormula)

	if normCert != normCand {
		// Calculate structural divergence
		div := computeStringDivergence(normCert, normCand)
		res.DivergenceScore = div
		if div > 0.05 {
			res.IsValid = false
			res.DiscrepancyNotes = append(res.DiscrepancyNotes,
				fmt.Sprintf("Formula diverges from boardroom-certified definition. Expected %q, got %q (divergence: %.1f%%)",
					metric.CertifiedFormula, candFormula, div*100))
		}
	}

	// 2. Check required filters in WHERE clause
	for _, reqFilter := range metric.RequiredFilters {
		normReq := normalizeExprString(reqFilter)
		found := false
		for _, w := range whereStrs {
			if strings.Contains(w, normReq) || strings.Contains(normReq, w) {
				found = true
				break
			}
		}
		if !found {
			res.IsValid = false
			res.MissingFilters = append(res.MissingFilters, reqFilter)
			res.DiscrepancyNotes = append(res.DiscrepancyNotes,
				fmt.Sprintf("Mandatory metric governance filter missing: %q", reqFilter))
		}
	}

	// 3. Check allowed dimensions in GROUP BY
	if len(metric.AllowedDimensions) > 0 {
		allowedMap := make(map[string]bool)
		for _, dim := range metric.AllowedDimensions {
			allowedMap[strings.ToLower(dim)] = true
		}

		for _, grp := range groupBy {
			grpCols := ast.ExtractColumns(grp)
			for _, col := range grpCols {
				colName := strings.ToLower(col.Column)
				if !allowedMap[colName] {
					res.InvalidDimensions = append(res.InvalidDimensions, colName)
					res.DiscrepancyNotes = append(res.DiscrepancyNotes,
						fmt.Sprintf("Dimension %q is not certified for metric %q (allowed: %s)",
							col.ToSQL(), metric.Name, strings.Join(metric.AllowedDimensions, ", ")))
				}
			}
		}
	}

	return res
}

func normalizeExprString(s string) string {
	clean := strings.ToLower(s)
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.ReplaceAll(clean, "(", "")
	clean = strings.ReplaceAll(clean, ")", "")
	clean = strings.ReplaceAll(clean, `"`, "")
	clean = strings.ReplaceAll(clean, "`", "")
	clean = strings.ReplaceAll(clean, ";", "")
	return clean
}

func computeStringDivergence(a, b string) float64 {
	if a == b {
		return 0.0
	}
	lenA := len(a)
	lenB := len(b)
	if lenA == 0 || lenB == 0 {
		return 1.0
	}

	// Levenshtein distance
	matrix := make([][]int, lenA+1)
	for i := range matrix {
		matrix[i] = make([]int, lenB+1)
		matrix[i][0] = i
	}
	for j := 0; j <= lenB; j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}

	dist := matrix[lenA][lenB]
	maxLen := max(lenA, lenB)
	return float64(dist) / float64(maxLen)
}
