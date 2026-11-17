package probes

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/ast"
)

// FanoutProbe detects 1:N join fan-out multipliers inflating aggregated parent measures.
type FanoutProbe struct{}

func (f *FanoutProbe) Name() string {
	return "JoinFanoutGuard"
}

func (f *FanoutProbe) Description() string {
	return "Detects 1:N join fan-out that causes Cartesian multiplication and inflates parent table aggregations"
}

func (f *FanoutProbe) Evaluate(ctx *EvaluationContext) (*ProbeResult, error) {
	res := &ProbeResult{
		ProbeName:   f.Name(),
		Description: f.Description(),
		Passed:      true,
		Score:       1.0,
		Violations:  make([]Violation, 0),
		Metrics:     make(map[string]float64),
	}

	if ctx.AST == nil || ctx.AST.From == nil || len(ctx.AST.From.Joins) == 0 {
		return res, nil
	}

	aliasMap := ast.ExtractAliases(ctx.AST)
	primaryTbl := ctx.AST.From.Primary.Name
	primaryAlias := ctx.AST.From.Primary.EffectiveName()
	aggregates := ast.ExtractAggregates(ctx.AST)

	maxMultiplier := 1.0

	// Check each join
	for _, join := range ctx.AST.From.Joins {
		childTbl := join.Table.Name
		childAlias := join.Table.EffectiveName()

		// Resolve actual table names from catalog
		realParent := aliasMap[strings.ToLower(primaryAlias)]
		if realParent == "" {
			realParent = primaryTbl
		}
		realChild := aliasMap[strings.ToLower(childAlias)]
		if realChild == "" {
			realChild = childTbl
		}

		// Inspect join condition
		lCol, rCol, ok := ast.InspectJoinCondition(join.Condition)
		pColName := "id"
		cColName := "id"
		if ok {
			if strings.EqualFold(lCol.Table, primaryAlias) {
				pColName = lCol.Column
				cColName = rCol.Column
			} else {
				pColName = rCol.Column
				cColName = lCol.Column
			}
		}

		isFanout, multiplier := false, 1.0
		if ctx.Catalog != nil {
			isFanout, multiplier = ctx.Catalog.CheckJoinFanout(realParent, realChild, pColName, cColName)
		} else {
			// In absence of catalog, assume non-PK join or multiple joins have fanout risk
			if strings.Contains(strings.ToLower(cColName), "id") && !strings.EqualFold(cColName, "id") {
				isFanout = true
				multiplier = 3.0
			}
		}

		if multiplier > maxMultiplier {
			maxMultiplier = multiplier
		}

		if isFanout && multiplier > 1.0 {
			// Check if any aggregate targets parent table columns
			for _, agg := range aggregates {
				cols := ast.ExtractColumns(agg)
				for _, col := range cols {
					colTbl := col.Table
					if colTbl == "" || strings.EqualFold(colTbl, primaryAlias) || strings.EqualFold(colTbl, primaryTbl) {
						// Parent column is aggregated in a 1:N join query!
						res.Passed = false
						res.Score = 0.0

						viol := Violation{
							ProbeName:        f.Name(),
							Severity:         SeverityCritical,
							Title:            fmt.Sprintf("Cartesian Metric Inflation: %s -> %s (Fanout: %.1fx)", realParent, realChild, multiplier),
							Message:          fmt.Sprintf("Parent table %q measure %q is aggregated (%s) over 1:N join with %q, inflating totals by approx %.1fx", realParent, col.ToSQL(), agg.Name, realChild, multiplier),
							OffendingNode:    agg.ToSQL(),
							Remediation:      fmt.Sprintf("Isolate child relation %q into a pre-aggregated Common Table Expression (CTE) before joining with %q", realChild, realParent),
							AutoFixAvailable: true,
							MathematicalProof: fmt.Sprintf("Phi_join(%s, %s) = %.2f > 1.0. Aggregate %s(%s) evaluates across %s rows instead of 1 row per %s record.",
								realParent, realChild, multiplier, agg.Name, col.ToSQL(), realChild, realParent),
						}
						res.Violations = append(res.Violations, viol)
					}
				}
			}
		}
	}

	res.Metrics["phi_join_max"] = maxMultiplier
	if !res.Passed {
		res.Metrics["inflation_multiplier"] = maxMultiplier
	}

	return res, nil
}
