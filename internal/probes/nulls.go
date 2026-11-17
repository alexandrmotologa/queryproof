package probes

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/ast"
)

// NullsProbe detects unhandled NULLs in aggregations, divisions by zero, and 3-valued logic traps.
type NullsProbe struct{}

func (n *NullsProbe) Name() string {
	return "NullabilityGuard"
}

func (n *NullsProbe) Description() string {
	return "Detects unhandled NULLs in AVG aggregations, division by zero vulnerabilities, and 3-valued logic traps"
}

func (n *NullsProbe) Evaluate(ctx *EvaluationContext) (*ProbeResult, error) {
	res := &ProbeResult{
		ProbeName:   n.Name(),
		Description: n.Description(),
		Passed:      true,
		Score:       1.0,
		Violations:  make([]Violation, 0),
		Metrics:     make(map[string]float64),
	}

	if ctx.AST == nil {
		return res, nil
	}

	aliasMap := ast.ExtractAliases(ctx.AST)

	// 1. Check division by zero and nullable denominators in projections
	for _, proj := range ctx.AST.Projections {
		n.inspectDivision(proj.Expr, res)
	}

	// 2. Check AVG on nullable columns
	aggregates := ast.ExtractAggregates(ctx.AST)
	for _, agg := range aggregates {
		if strings.EqualFold(agg.Name, "AVG") && len(agg.Args) == 1 {
			cols := ast.ExtractColumns(agg.Args[0])
			for _, col := range cols {
				// Check if column is wrapped in COALESCE
				if n.isWrappedInCoalesce(agg.Args[0]) {
					continue
				}

				tblName := aliasMap[strings.ToLower(col.Table)]
				if tblName == "" {
					tblName = col.Table
				}

				isNullable := true
				if ctx.Catalog != nil && tblName != "" {
					cMeta := ctx.Catalog.FindColumn(tblName, col.Column)
					if cMeta != nil {
						isNullable = cMeta.Nullable
					}
				}

				if isNullable {
					res.Passed = false
					res.Score = 0.5
					res.Violations = append(res.Violations, Violation{
						ProbeName:        n.Name(),
						Severity:         SeverityHigh,
						Title:            fmt.Sprintf("Silent Mean Distortion: AVG(%s)", col.ToSQL()),
						Message:          fmt.Sprintf("Column %q is nullable. SQL standard AVG() silently drops NULL rows rather than treating them as 0, artificially inflating averages", col.ToSQL()),
						OffendingNode:    agg.ToSQL(),
						Remediation:      fmt.Sprintf("Wrap column with COALESCE(%s, 0) inside AVG() to include non-participating rows in denominator", col.ToSQL()),
						AutoFixAvailable: true,
						MathematicalProof: "AVG(S) = sum(s_i) / |{s_i != NULL}|. If NULL represents zero activity, true mean is sum(s_i) / |S|.",
					})
				}
			}
		}
	}

	// 3. Check NOT IN with nullable subqueries
	if ctx.AST.Where != nil {
		n.inspectNotIn(ctx.AST.Where, res)
	}

	res.Metrics["null_violations_count"] = float64(len(res.Violations))
	return res, nil
}

func (n *NullsProbe) inspectDivision(expr ast.Expr, res *ProbeResult) {
	if expr == nil {
		return
	}

	if bin, ok := expr.(*ast.BinaryExpr); ok {
		if bin.Op == "/" {
			// Check if denominator has NULLIF wrap
			if !n.hasNullifWrap(bin.Right) {
				res.Passed = false
				res.Score = min(res.Score, 0.4)
				res.Violations = append(res.Violations, Violation{
					ProbeName:        n.Name(),
					Severity:         SeverityHigh,
					Title:            "Unprotected Arithmetic Division",
					Message:          fmt.Sprintf("Division denominator %q is not protected against zero or NULL values, leading to query crashes or NULL propagation", bin.Right.ToSQL()),
					OffendingNode:    bin.ToSQL(),
					Remediation:      fmt.Sprintf("Wrap denominator in NULLIF(%s, 0) and encapsulate division in COALESCE", bin.Right.ToSQL()),
					AutoFixAvailable: true,
					MathematicalProof: "lim (x -> 0) A / x is undefined; SQL engines abort with division by zero unless protected by NULLIF.",
				})
			}
		}
		n.inspectDivision(bin.Left, res)
		n.inspectDivision(bin.Right, res)
	}

	if paren, ok := expr.(*ast.ParenExpr); ok {
		n.inspectDivision(paren.Inner, res)
	}
}

func (n *NullsProbe) hasNullifWrap(expr ast.Expr) bool {
	if fn, ok := expr.(*ast.FunctionCall); ok {
		return strings.EqualFold(fn.Name, "NULLIF")
	}
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return n.hasNullifWrap(paren.Inner)
	}
	return false
}

func (n *NullsProbe) isWrappedInCoalesce(expr ast.Expr) bool {
	if fn, ok := expr.(*ast.FunctionCall); ok {
		return strings.EqualFold(fn.Name, "COALESCE")
	}
	return false
}

func (n *NullsProbe) inspectNotIn(expr ast.Expr, res *ProbeResult) {
	if inExpr, ok := expr.(*ast.InExpr); ok && inExpr.Not {
		if inExpr.Subquery != nil {
			res.Passed = false
			res.Score = min(res.Score, 0.5)
			res.Violations = append(res.Violations, Violation{
				ProbeName:        n.Name(),
				Severity:         SeverityHigh,
				Title:            "3-Valued Logic NULL Trap in NOT IN Subquery",
				Message:          "Using NOT IN with a subquery returns zero rows if any subquery result contains a NULL value",
				OffendingNode:    inExpr.ToSQL(),
				Remediation:      "Replace NOT IN with NOT EXISTS or explicitly add 'WHERE <col> IS NOT NULL' to subquery",
				AutoFixAvailable: false,
				MathematicalProof: "In ANSI SQL, 'x NOT IN (y, NULL)' evaluates to UNKNOWN for all x, causing the WHERE filter to drop all candidate rows.",
			})
		}
	}

	if bin, ok := expr.(*ast.BinaryExpr); ok {
		n.inspectNotIn(bin.Left, res)
		n.inspectNotIn(bin.Right, res)
	}
	if paren, ok := expr.(*ast.ParenExpr); ok {
		n.inspectNotIn(paren.Inner, res)
	}
}
