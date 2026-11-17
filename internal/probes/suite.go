package probes

import (
	"github.com/alexandrmotologa/queryproof/internal/config"
)

// Suite coordinates and executes all registered falsification probes.
type Suite struct {
	probes []Probe
	config *config.ProbeConfig
}

// SuiteResult contains aggregated findings from all probes.
type SuiteResult struct {
	Passed          bool           `json:"passed"`
	OverallScore    float64        `json:"overall_score"`
	CriticalCount   int            `json:"critical_count"`
	HighCount       int            `json:"high_count"`
	MediumCount     int            `json:"medium_count"`
	LowCount        int            `json:"low_count"`
	Results         []*ProbeResult `json:"results"`
	AllViolations   []Violation    `json:"all_violations"`
	AutoFixPossible bool           `json:"auto_fix_possible"`
}

// NewDefaultSuite registers all active falsification probes based on configuration.
func NewDefaultSuite(cfg *config.ProbeConfig) *Suite {
	s := &Suite{
		probes: make([]Probe, 0),
		config: cfg,
	}

	if cfg == nil || cfg.EnableFanoutGuard {
		s.probes = append(s.probes, &FanoutProbe{})
	}
	if cfg == nil || cfg.EnableNullabilityTest {
		s.probes = append(s.probes, &NullsProbe{})
	}
	if cfg == nil || cfg.EnableMetricCheck {
		s.probes = append(s.probes, &MetricCheckProbe{})
	}
	if cfg == nil || cfg.EnableGovernanceGuard {
		s.probes = append(s.probes, &GovernanceProbe{})
	}
	if cfg == nil || cfg.EnablePartitionGuard {
		s.probes = append(s.probes, &PartitionProbe{})
	}

	return s
}

// Run executes all probes against the evaluation context.
func (s *Suite) Run(ctx *EvaluationContext) (*SuiteResult, error) {
	sr := &SuiteResult{
		Passed:          true,
		OverallScore:    1.0,
		Results:         make([]*ProbeResult, 0, len(s.probes)),
		AllViolations:   make([]Violation, 0),
		AutoFixPossible: true,
	}

	totalScore := 0.0

	for _, probe := range s.probes {
		res, err := probe.Evaluate(ctx)
		if err != nil {
			return nil, err
		}

		sr.Results = append(sr.Results, res)
		totalScore += res.Score

		if !res.Passed {
			sr.Passed = false
			for _, v := range res.Violations {
				sr.AllViolations = append(sr.AllViolations, v)
				switch v.Severity {
				case SeverityCritical:
					sr.CriticalCount++
				case SeverityHigh:
					sr.HighCount++
				case SeverityMedium:
					sr.MediumCount++
				case SeverityLow:
					sr.LowCount++
				}
				if !v.AutoFixAvailable {
					sr.AutoFixPossible = false
				}
			}
		}
	}

	if len(s.probes) > 0 {
		sr.OverallScore = totalScore / float64(len(s.probes))
	}
	if sr.CriticalCount > 0 {
		sr.Passed = false
	}

	return sr, nil
}
