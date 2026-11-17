package probes

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/ast"
)

// GovernanceProbe ensures multi-tenant scoping and soft-delete filters are strictly applied.
type GovernanceProbe struct{}

func (g *GovernanceProbe) Name() string {
	return "GovernanceGuard"
}

func (g *GovernanceProbe) Description() string {
	return "Enforces mandatory multi-tenant isolation predicates and soft-delete compliance across participating tables"
}

func (g *GovernanceProbe) Evaluate(ctx *EvaluationContext) (*ProbeResult, error) {
	res := &ProbeResult{
		ProbeName:   g.Name(),
		Description: g.Description(),
		Passed:      true,
		Score:       1.0,
		Violations:  make([]Violation, 0),
		Metrics:     make(map[string]float64),
	}

	if ctx.AST == nil || ctx.Catalog == nil {
		return res, nil
	}

	tables := ast.ExtractTables(ctx.AST)
	whereConds := ast.ExtractWhereConjunctions(ctx.AST.Where)

	var whereStrs []string
	for _, cond := range whereConds {
		whereStrs = append(whereStrs, strings.ToLower(cond.ToSQL()))
	}

	for _, tblRef := range tables {
		tbl := ctx.Catalog.FindTable(tblRef.Name)
		if tbl == nil {
			continue
		}

		tblAlias := tblRef.EffectiveName()

		// 1. Check Tenancy filter
		tenancyCol := tbl.TenancyFilter
		if tenancyCol == "" {
			for _, col := range tbl.Columns {
				if col.IsTenancyKey {
					tenancyCol = col.Name
					break
				}
			}
		}

		if tenancyCol != "" {
			cleanTenancy := strings.ToLower(tenancyCol)
			hasTenantFilter := false
			for _, w := range whereStrs {
				if strings.Contains(w, cleanTenancy) {
					hasTenantFilter = true
					break
				}
			}

			if !hasTenantFilter {
				res.Passed = false
				res.Score = 0.0
				targetExpr := fmt.Sprintf("%s.%s = :tenant_id", tblAlias, tenancyCol)
				res.Violations = append(res.Violations, Violation{
					ProbeName:        g.Name(),
					Severity:         SeverityCritical,
					Title:            fmt.Sprintf("Tenant Isolation Breach on %s", tbl.Name),
					Message:          fmt.Sprintf("Table %q requires tenant isolation column %q in WHERE clause, but none was provided", tbl.Name, tenancyCol),
					OffendingNode:    tblRef.ToSQL(),
					Remediation:      fmt.Sprintf("Inject filter %s into WHERE clause", targetExpr),
					AutoFixAvailable: true,
					MathematicalProof: fmt.Sprintf("Table %s contains multi-tenant records; omitting %s leaks cross-organizational data.", tbl.Name, tenancyCol),
				})
			}
		}

		// 2. Check Soft Delete filter
		softDeleteFilter := tbl.SoftDeleteFilter
		softDeleteCol := ""
		if softDeleteFilter == "" {
			for _, col := range tbl.Columns {
				if col.IsSoftDeleteKey {
					softDeleteCol = col.Name
					softDeleteFilter = fmt.Sprintf("%s IS NULL", col.Name)
					break
				}
			}
		}

		if softDeleteFilter != "" {
			cleanSoftDel := strings.ToLower(softDeleteFilter)
			hasSoftDeleteFilter := false
			for _, w := range whereStrs {
				if strings.Contains(w, cleanSoftDel) || (softDeleteCol != "" && strings.Contains(w, strings.ToLower(softDeleteCol))) {
					hasSoftDeleteFilter = true
					break
				}
			}

			if !hasSoftDeleteFilter {
				res.Passed = false
				res.Score = min(res.Score, 0.4)
				injectPredicate := fmt.Sprintf("%s.%s", tblAlias, softDeleteFilter)
				res.Violations = append(res.Violations, Violation{
					ProbeName:        g.Name(),
					Severity:         SeverityHigh,
					Title:            fmt.Sprintf("Soft-Delete Filter Omitted on %s", tbl.Name),
					Message:          fmt.Sprintf("Table %q implements soft deletes; query omitted mandatory predicate %q", tbl.Name, softDeleteFilter),
					OffendingNode:    tblRef.ToSQL(),
					Remediation:      fmt.Sprintf("Inject %q into WHERE clause", injectPredicate),
					AutoFixAvailable: true,
					MathematicalProof: fmt.Sprintf("Records marked as deleted in %s remain visible in queries lacking soft-delete predicates.", tbl.Name),
				})
			}
		}
	}

	res.Metrics["governance_violations"] = float64(len(res.Violations))
	return res, nil
}
