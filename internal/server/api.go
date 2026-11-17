package server

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/engine"
	"github.com/alexandrmotologa/queryproof/internal/ledger"
	"github.com/alexandrmotologa/queryproof/internal/metrics"
	"github.com/alexandrmotologa/queryproof/internal/proof"
	"github.com/alexandrmotologa/queryproof/internal/proxy"
)

type VerifyRequest struct {
	Query   string         `json:"query"`
	Dialect config.Dialect `json:"dialect,omitempty"`
}

type VerifyPassportRequest struct {
	Voucher *proof.ProofVoucher `json:"voucher"`
}

type Scenario struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	TargetProbe string         `json:"target_probe"`
	Dialect     config.Dialect `json:"dialect"`
	SQL         string         `json:"sql"`
}

type HealthResponse struct {
	Status        string  `json:"status"`
	Version       string  `json:"version"`
	AllocatedMB   float64 `json:"allocated_mb"`
	TotalAllocMB  float64 `json:"total_alloc_mb"`
	NumGoroutine  int     `json:"num_goroutine"`
	UptimeSeconds float64 `json:"uptime_seconds"`
}

// APIHandler handles REST requests for the verification studio.
type APIHandler struct {
	engine    *engine.VerificationEngine
	catalog   *config.Catalog
	registry  *metrics.Registry
	ledger    *ledger.AuditLedger
	gateway   *proxy.AgentGateway
	secretKey string
	startTime time.Time
}

func NewAPIHandler(
	eng *engine.VerificationEngine,
	cat *config.Catalog,
	reg *metrics.Registry,
	led *ledger.AuditLedger,
	gw *proxy.AgentGateway,
	secretKey string,
) *APIHandler {
	return &APIHandler{
		engine:    eng,
		catalog:   cat,
		registry:  reg,
		ledger:    led,
		gateway:   gw,
		secretKey: secretKey,
		startTime: time.Now(),
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *APIHandler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Query cannot be empty"})
		return
	}

	report, err := h.engine.Verify(req.Query)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	if h.ledger != nil {
		_ = h.ledger.RecordEvent(report)
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *APIHandler) HandleLedger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.ledger == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"entries": []interface{}{}, "total": 0})
		return
	}

	limit := 50
	offset := 0
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if val, err := strconv.Atoi(lStr); err == nil && val > 0 {
			limit = val
		}
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if val, err := strconv.Atoi(oStr); err == nil && val >= 0 {
			offset = val
		}
	}

	entries, total, err := h.ledger.ListEvents(limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"total":   total,
	})
}

func (h *APIHandler) HandleLedgerStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.ledger == nil {
		writeJSON(w, http.StatusOK, &ledger.LedgerStats{})
		return
	}

	stats, err := h.ledger.GetStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (h *APIHandler) HandleVerifyPassport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyPassportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		return
	}

	if req.Voucher == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Voucher missing"})
		return
	}

	valid, err := proof.VerifyVoucher(req.Voucher, h.secretKey)
	resp := map[string]interface{}{
		"valid": valid,
	}
	if err != nil {
		resp["error"] = err.Error()
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *APIHandler) HandleCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, h.catalog)
}

func (h *APIHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.registry == nil {
		writeJSON(w, http.StatusOK, []metrics.Metric{})
		return
	}
	writeJSON(w, http.StatusOK, h.registry.ListMetrics())
}

func (h *APIHandler) HandleScenarios(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scenarios := []Scenario{
		{
			ID:          "cartesian_fanout",
			Title:       "1:N Join Fan-Out on Enterprise Subscriptions",
			Description: "Candidate Text-to-SQL joins customers to orders and naively sums customer ARR, producing a 3.5x financial inflation.",
			TargetProbe: "JoinFanoutGuard",
			Dialect:     config.DialectPostgres,
			SQL: `SELECT c.id, c.name, SUM(c.mrr) AS total_mrr, COUNT(o.id) AS order_cnt
FROM customers AS c
INNER JOIN orders AS o ON c.id = o.customer_id
WHERE c.tenant_id = 'org_1' AND c.deleted_at IS NULL
GROUP BY c.id, c.name;`,
		},
		{
			ID:          "null_average_distortion",
			Title:       "Silent Mean Bias & Unprotected Division",
			Description: "Calculates average discount and margin ratio; ignores that SQL AVG() drops NULLs and denominator can be zero.",
			TargetProbe: "NullabilityGuard",
			Dialect:     config.DialectPostgres,
			SQL: `SELECT AVG(o.discount_pct) AS avg_discount, SUM(o.amount) / o.amount AS margin_ratio
FROM orders AS o
WHERE o.tenant_id = 'org_1'
GROUP BY o.amount;`,
		},
		{
			ID:          "tenant_softdelete_breach",
			Title:       "Tenant Scoping & Soft-Delete Filter Omission",
			Description: "Agent omitted mandatory multi-tenant and deleted_at predicates, leaking confidential enterprise data.",
			TargetProbe: "GovernanceGuard",
			Dialect:     config.DialectPostgres,
			SQL: `SELECT id, name, mrr
FROM customers
WHERE mrr > 1000;`,
		},
		{
			ID:          "partition_full_scan",
			Title:       "BigQuery Full Table Scan on 50M Events",
			Description: "Query targets petabyte analytics table without filtering by partition date column event_date.",
			TargetProbe: "PartitionGuard",
			Dialect:     config.DialectBigQuery,
			SQL: `SELECT id, action
FROM analytics_events
WHERE action = 'checkout';`,
		},
		{
			ID:          "hallucinated_metric",
			Title:       "Ad-hoc ARR Formula Hallucination",
			Description: "LLM attempts to calculate ARR using uncertified formula instead of the corporate definition in metrics.yaml.",
			TargetProbe: "SemanticMetricGuard",
			Dialect:     config.DialectPostgres,
			SQL: `SELECT c.id, SUM(c.mrr * 10) AS arr
FROM customers AS c
WHERE c.tenant_id = 'org_1' AND c.deleted_at IS NULL
GROUP BY c.id;`,
		},
	}

	writeJSON(w, http.StatusOK, scenarios)
}

func (h *APIHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	resp := HealthResponse{
		Status:        "healthy",
		Version:       "1.0.0",
		AllocatedMB:   float64(m.Alloc) / (1024 * 1024),
		TotalAllocMB:  float64(m.TotalAlloc) / (1024 * 1024),
		NumGoroutine:  runtime.NumGoroutine(),
		UptimeSeconds: time.Since(h.startTime).Seconds(),
	}

	writeJSON(w, http.StatusOK, resp)
}
