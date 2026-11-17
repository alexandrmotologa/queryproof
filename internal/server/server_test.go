package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/engine"
	"github.com/alexandrmotologa/queryproof/internal/metrics"
)

func TestServerAPIEndpoints(t *testing.T) {
	cat := &config.Catalog{
		Dialect: config.DialectPostgres,
		Tables: []config.Table{
			{Name: "customers", Columns: []config.Column{{Name: "id", IsPrimaryKey: true}}},
		},
	}
	cat.Init()

	dict := &metrics.SemanticDictionary{
		Version: "1.0",
		Metrics: []metrics.Metric{{Name: "ARR", CertifiedFormula: "SUM(mrr * 12)"}},
	}
	reg := metrics.NewRegistry(dict)

	cfg := config.DefaultConfig()
	eng, err := engine.NewEngine(cfg, cat, reg)
	if err != nil {
		t.Fatalf("engine error: %v", err)
	}

	handler := NewAPIHandler(eng, cat, reg, "test-secret")

	// Test 1: Health endpoint
	reqHealth := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	wHealth := httptest.NewRecorder()
	handler.HandleHealth(wHealth, reqHealth)
	if wHealth.Code != http.StatusOK {
		t.Errorf("health returned status %d", wHealth.Code)
	}

	// Test 2: Scenarios endpoint
	reqSc := httptest.NewRequest(http.MethodGet, "/api/scenarios", nil)
	wSc := httptest.NewRecorder()
	handler.HandleScenarios(wSc, reqSc)
	if wSc.Code != http.StatusOK {
		t.Errorf("scenarios returned status %d", wSc.Code)
	}

	// Test 3: Catalog endpoint
	reqCat := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	wCat := httptest.NewRecorder()
	handler.HandleCatalog(wCat, reqCat)
	if wCat.Code != http.StatusOK {
		t.Errorf("catalog returned status %d", wCat.Code)
	}

	// Test 4: Metrics endpoint
	reqMet := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	wMet := httptest.NewRecorder()
	handler.HandleMetrics(wMet, reqMet)
	if wMet.Code != http.StatusOK {
		t.Errorf("metrics returned status %d", wMet.Code)
	}

	// Test 5: Verify endpoint
	verifyBody := map[string]string{
		"query":   "SELECT id FROM customers",
		"dialect": "postgres",
	}
	bodyBytes, _ := json.Marshal(verifyBody)
	reqVerify := httptest.NewRequest(http.MethodPost, "/api/verify", bytes.NewReader(bodyBytes))
	wVerify := httptest.NewRecorder()
	handler.HandleVerify(wVerify, reqVerify)
	if wVerify.Code != http.StatusOK {
		t.Errorf("verify returned status %d", wVerify.Code)
	}
}
