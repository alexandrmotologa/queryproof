package probes

import (
	"github.com/alexandrmotologa/queryproof/internal/ast"
	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/metrics"
)

// Severity indicates the risk level of an integrity violation.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL" // Mathematical inflation or data leak
	SeverityHigh     Severity = "HIGH"     // Runtime crash or silent null bias
	SeverityMedium   Severity = "MEDIUM"   // Uncertified metric or dimension
	SeverityLow      Severity = "LOW"      // Missing partition or performance risk
	SeverityInfo     Severity = "INFO"
)

// Violation describes a specific detected mathematical or compliance defect.
type Violation struct {
	ProbeName         string   `json:"probe_name"`
	Severity          Severity `json:"severity"`
	Title             string   `json:"title"`
	Message           string   `json:"message"`
	OffendingNode     string   `json:"offending_node"`
	Remediation       string   `json:"remediation"`
	AutoFixAvailable  bool     `json:"auto_fix_available"`
	MathematicalProof string   `json:"mathematical_proof,omitempty"`
}

// ProbeResult encapsulates the evaluation results of a single probe.
type ProbeResult struct {
	ProbeName   string             `json:"probe_name"`
	Description string             `json:"description"`
	Passed      bool               `json:"passed"`
	Score       float64            `json:"score"` // 0.0 (failed) to 1.0 (perfect)
	Violations  []Violation        `json:"violations"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
}

// EvaluationContext provides all required schema, metric, and query context to probes.
type EvaluationContext struct {
	AST      *ast.SelectStmt
	RawSQL   string
	Catalog  *config.Catalog
	Registry *metrics.Registry
	Config   *config.ProbeConfig
	Dialect  config.Dialect
}

// Probe is the standard interface for all falsification probes.
type Probe interface {
	Name() string
	Description() string
	Evaluate(ctx *EvaluationContext) (*ProbeResult, error)
}
