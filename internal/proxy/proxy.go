package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/engine"
	"github.com/alexandrmotologa/queryproof/internal/ledger"
	"github.com/alexandrmotologa/queryproof/internal/proof"
)

// InterceptRequest is sent by Text-to-SQL agents prior to database execution.
type InterceptRequest struct {
	Query     string         `json:"query"`
	Dialect   config.Dialect `json:"dialect,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Execute   bool           `json:"execute,omitempty"`
}

// InterceptResponse returns the safe rewritten SQL, proof passport, and execution status.
type InterceptResponse struct {
	Status          proof.VerificationStatus `json:"status"`
	OriginalSQL     string                   `json:"original_sql"`
	SafeSQL         string                   `json:"safe_sql"`
	IntegrityScore  float64                  `json:"integrity_score"`
	Falsified       bool                     `json:"falsified"`
	MutationsCount  int                      `json:"mutations_count"`
	ViolationsCount int                      `json:"violations_count"`
	Voucher         *proof.ProofVoucher      `json:"voucher"`
	ExecutionSample []string                 `json:"execution_sample,omitempty"`
	LatencyMs       float64                  `json:"latency_ms"`
}

// AgentGateway provides a drop-in middleware interceptor for LLM Text-to-SQL agents.
type AgentGateway struct {
	engine *engine.VerificationEngine
	ledger *ledger.AuditLedger
}

// NewAgentGateway creates a new gateway instance.
func NewAgentGateway(eng *engine.VerificationEngine, led *ledger.AuditLedger) *AgentGateway {
	return &AgentGateway{
		engine: eng,
		ledger: led,
	}
}

// HandleIntercept handles agent query interception requests.
func (g *AgentGateway) HandleIntercept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InterceptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Query) == "" {
		http.Error(w, "query cannot be empty", http.StatusBadRequest)
		return
	}

	report, err := g.engine.Verify(req.Query)
	if err != nil {
		http.Error(w, fmt.Sprintf("verification error: %v", err), http.StatusUnprocessableEntity)
		return
	}

	// Persist to audit ledger if configured
	if g.ledger != nil {
		_ = g.ledger.RecordEvent(report)
	}

	safeSQL := report.OriginalSQL
	if report.RewrittenSQL != "" {
		safeSQL = report.RewrittenSQL
	}

	resp := InterceptResponse{
		Status:          report.Voucher.Status,
		OriginalSQL:     report.OriginalSQL,
		SafeSQL:         safeSQL,
		IntegrityScore:  report.Voucher.IntegrityScore,
		Falsified:       !report.IsClean,
		MutationsCount:  len(report.Mutations),
		ViolationsCount: len(report.SuiteResult.AllViolations),
		Voucher:         report.Voucher,
		LatencyMs:       report.VerificationMs,
	}

	if report.SandboxExecution != nil && len(report.SandboxExecution.SampleResult) > 0 {
		resp.ExecutionSample = report.SandboxExecution.SampleResult
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// ToolCallInterceptor intercepts OpenAI/Anthropic format tool execution arguments.
func (g *AgentGateway) InterceptToolArguments(argsJSON string) (safeArgsJSON string, modified bool, err error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &parsed); err != nil {
		return argsJSON, false, err
	}

	// Look for query field
	for _, key := range []string{"query", "sql", "statement", "command"} {
		if rawVal, exists := parsed[key]; exists {
			if sqlStr, ok := rawVal.(string); ok && len(sqlStr) > 0 {
				rep, err := g.engine.Verify(sqlStr)
				if err != nil {
					return argsJSON, false, err
				}
				if g.ledger != nil {
					_ = g.ledger.RecordEvent(rep)
				}
				if rep.RewrittenSQL != "" {
					parsed[key] = rep.RewrittenSQL
					parsed["_queryproof_voucher"] = rep.Voucher.VoucherID
					parsed["_queryproof_status"] = rep.Voucher.Status
					newBytes, _ := json.Marshal(parsed)
					return string(newBytes), true, nil
				}
			}
		}
	}

	return argsJSON, false, nil
}
