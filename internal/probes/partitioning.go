package probes

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/ast"
)

// PartitionProbe verifies partition pruning and prevents expensive full table scans.
type PartitionProbe struct{}

func (p *PartitionProbe) Name() string {
	return "PartitionGuard"
}

func (p *PartitionProbe) Description() string {
	return "Detects missing partition pruning filters on large analytical tables and unbounded table scans"
}

func (p *PartitionProbe) Evaluate(ctx *EvaluationContext) (*ProbeResult, error) {
	res := &ProbeResult{
		ProbeName:   p.Name(),
		Description: p.Description(),
		Passed:      true,
		Score:       1.0,
		Violations:  make([]Violation, 0),
		Metrics:     make(map[string]float64),
	}

	if ctx.AST == nil {
		return res, nil
	}

	tables := ast.ExtractTables(ctx.AST)
	whereConds := ast.ExtractWhereConjunctions(ctx.AST.Where)

	var whereStrs []string
	for _, cond := range whereConds {
		whereStrs = append(whereStrs, strings.ToLower(cond.ToSQL()))
	}

	// 1. Partition column pruning check
	if ctx.Catalog != nil {
		for _, tblRef := range tables {
			tbl := ctx.Catalog.FindTable(tblRef.Name)
			if tbl == nil {
				continue
			}

			partCol := tbl.PartitionColumn
			if partCol == "" {
				for _, c := range tbl.Columns {
					if c.IsPartitionKey {
						partCol = c.Name
						break
					}
				}
			}

			if partCol != "" {
				cleanPart := strings.ToLower(partCol)
				hasPartFilter := false
				for _, w := range whereStrs {
					if strings.Contains(w, cleanPart) {
						hasPartFilter = true
						break
					}
				}

				if !hasPartFilter {
					res.Passed = false
					res.Score = min(res.Score, 0.5)
					res.Violations = append(res.Violations, Violation{
						ProbeName:        p.Name(),
						Severity:         SeverityHigh,
						Title:            fmt.Sprintf("Missing Partition Pruning Filter on %s", tbl.Name),
						Message:          fmt.Sprintf("Table %q is partitioned by %q. Omitting partition filters causes full table scans costing up to hundreds of dollars per execution", tbl.Name, partCol),
						OffendingNode:    tblRef.ToSQL(),
						Remediation:      fmt.Sprintf("Add date/partition filter (e.g. %s.%s >= CURRENT_DATE - INTERVAL '30 days') to WHERE clause", tblRef.EffectiveName(), partCol),
						AutoFixAvailable: false,
						MathematicalProof: fmt.Sprintf("Partitioned table %s contains %d rows across multiple partitions. Full scan reads 100%% of storage blocks.", tbl.Name, tbl.EstimatedRowCount),
					})
				}
			}
		}
	}

	// 2. Unbounded query check (no LIMIT and no GROUP BY)
	if ctx.AST.Limit == nil && len(ctx.AST.GroupBy) == 0 {
		res.Violations = append(res.Violations, Violation{
			ProbeName:        p.Name(),
			Severity:         SeverityLow,
			Title:            "Unbounded Query Without LIMIT Clause",
			Message:          "Query selects raw rows without a LIMIT boundary, risking client memory exhaustion on large datasets",
			OffendingNode:    "SELECT",
			Remediation:      "Add an explicit LIMIT (e.g. LIMIT 1000) for analytical previews",
			AutoFixAvailable: true,
		})
	}

	res.Metrics["partition_violations"] = float64(len(res.Violations))
	return res, nil
}
