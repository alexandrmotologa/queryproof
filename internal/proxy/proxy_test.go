package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/engine"
	"github.com/alexandrmotologa/queryproof/internal/ledger"
	"github.com/alexandrmotologa/queryproof/internal/metrics"
	"github.com/alexandrmotologa/queryproof/internal/proof"
)

func setupTestGateway(t *testing.T) *AgentGateway {
	cat := &config.Catalog{
		Dialect: config.DialectPostgres,
		Tables: []config.Table{
			{
				Name: "customers",
				Columns: []config.Column{
					{Name: "id", IsPrimaryKey: true},
					{Name: "mrr", DataType: "NUMERIC"},
					{Name: "tenant_id", IsTenancyKey: true},
					{Name: "deleted_at", IsSoftDeleteKey: true},
				},
				TenancyFilter:    "tenant_id",
				SoftDeleteFilter: "deleted_at IS NULL",
			},
			{
				Name: "orders",
				Columns: []config.Column{
					{Name: "id", IsPrimaryKey: true},
					{Name: "customer_id", IsForeignKey: true, RefTable: "customers", RefColumn: "id"},
					{Name: "amount", DataType: "NUMERIC"},
					{Name: "discount_pct", Nullable: true},
				},
			},
		},
		Relationships: []config.Relationship{
			{
				FromTable:   "customers",
				FromColumn:  "id",
				ToTable:     "orders",
				ToColumn:    "customer_id",
				Cardinality: config.CardOneToMany,
				FanoutRate:  10.0,
			},
		},
	}
	cat.Init()

	dict := &metrics.SemanticDictionary{Version: "1.0"}
	reg := metrics.NewRegistry(dict)

	cfg := config.DefaultConfig()
	eng, err := engine.NewEngine(cfg, cat, reg)
	if err != nil {
		t.Fatalf("engine error: %v", err)
	}

	led, err := ledger.OpenLedger(":memory:")
	if err != nil {
		t.Fatalf("ledger error: %v", err)
	}

	return NewAgentGateway(eng, led)
}

func TestAgentGatewayIntercept(t *testing.T) {
	gw := setupTestGateway(t)

	reqPayload := InterceptRequest{
		Query:   "SELECT c.id, SUM(c.mrr) FROM customers c JOIN orders o ON c.id=o.customer_id GROUP BY c.id",
		AgentID: "langchain-agent-42",
		Execute: false,
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/v1/query/intercept", bytes.NewReader(body))
	w := httptest.NewRecorder()

	gw.HandleIntercept(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp InterceptResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response error: %v", err)
	}

	if !resp.Falsified {
		t.Errorf("expected query to be flagged as falsified")
	}
	if resp.Status != proof.StatusFalsifiedRewritten {
		t.Errorf("expected status FALSIFIED_AND_REWRITTEN, got %v", resp.Status)
	}
	if !strings.Contains(resp.SafeSQL, "orders_agg") {
		t.Errorf("expected safe rewritten SQL with orders_agg CTE, got %s", resp.SafeSQL)
	}
}

func TestInterceptToolArguments(t *testing.T) {
	gw := setupTestGateway(t)

	rawArgs := `{"query": "SELECT id, name FROM customers", "limit": 10}`
	safeArgs, modified, err := gw.InterceptToolArguments(rawArgs)
	if err != nil {
		t.Fatalf("intercept tool arguments error: %v", err)
	}

	if !modified {
		t.Errorf("expected arguments to be modified (injected governance filters)")
	}
	if !strings.Contains(safeArgs, "tenant_id") {
		t.Errorf("expected injected tenant_id filter in safe arguments: %s", safeArgs)
	}
}
