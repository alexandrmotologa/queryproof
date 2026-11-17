package probes

import (
	"fmt"

	"github.com/alexandrmotologa/queryproof/internal/metrics"
)

// MetricCheckProbe verifies query formulas against corporate semantic metrics.yaml.
type MetricCheckProbe struct{}

func (m *MetricCheckProbe) Name() string {
	return "SemanticMetricGuard"
}

func (m *MetricCheckProbe) Description() string {
	return "Validates calculated financial measures against authoritative enterprise KPI formulas and required filters"
}

func (m *MetricCheckProbe) Evaluate(ctx *EvaluationContext) (*ProbeResult, error) {
	res := &ProbeResult{
		ProbeName:   m.Name(),
		Description: m.Description(),
		Passed:      true,
		Score:       1.0,
		Violations:  make([]Violation, 0),
		Metrics:     make(map[string]float64),
	}

	if ctx.Registry == nil || ctx.AST == nil {
		return res, nil
	}

	validator := metrics.NewValidator(ctx.Registry)
	valResults := validator.ValidateQueryMetrics(ctx.AST)

	totalDivergence := 0.0
	for _, vr := range valResults {
		if !vr.IsValid {
			res.Passed = false
			totalDivergence += vr.DivergenceScore

			for _, note := range vr.DiscrepancyNotes {
				res.Violations = append(res.Violations, Violation{
					ProbeName:        m.Name(),
					Severity:         SeverityHigh,
					Title:            fmt.Sprintf("Semantic KPI Invariant Divergence: %s", vr.MetricName),
					Message:          note,
					OffendingNode:    vr.CandidateFormula,
					Remediation:      fmt.Sprintf("Replace candidate expression with certified definition: %s", vr.CertifiedFormula),
					AutoFixAvailable: true,
					MathematicalProof: fmt.Sprintf("Metric %s requires strictly formula %q. Divergence delta = %.2f",
						vr.MetricName, vr.CertifiedFormula, vr.DivergenceScore),
				})
			}

			for _, missingFilter := range vr.MissingFilters {
				res.Violations = append(res.Violations, Violation{
					ProbeName:        m.Name(),
					Severity:         SeverityCritical,
					Title:            fmt.Sprintf("Missing Governance Filter for %s", vr.MetricName),
					Message:          fmt.Sprintf("Certified metric %q requires filter %q in WHERE clause", vr.MetricName, missingFilter),
					OffendingNode:    "WHERE clause",
					Remediation:      fmt.Sprintf("Inject filter %q into WHERE clause", missingFilter),
					AutoFixAvailable: true,
				})
			}

			for _, invalidDim := range vr.InvalidDimensions {
				res.Violations = append(res.Violations, Violation{
					ProbeName:        m.Name(),
					Severity:         SeverityMedium,
					Title:            fmt.Sprintf("Uncertified Dimension Slicing: %s", invalidDim),
					Message:          fmt.Sprintf("Metric %q cannot be reliably aggregated across dimension %q", vr.MetricName, invalidDim),
					OffendingNode:    invalidDim,
					Remediation:      "Remove invalid grouping dimension or request data engineering certification",
					AutoFixAvailable: false,
				})
			}
		}
	}

	if len(valResults) > 0 {
		avgDiv := totalDivergence / float64(len(valResults))
		res.Metrics["formula_divergence"] = avgDiv
		if !res.Passed {
			res.Score = (1.0 - avgDiv) * 0.5
			if res.Score < 0 {
				res.Score = 0
			}
		}
	}

	return res, nil
}
