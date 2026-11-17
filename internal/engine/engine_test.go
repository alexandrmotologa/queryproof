package engine

import (
	"testing"

	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/metrics"
	"github.com/alexandrmotologa/queryproof/internal/proof"
)

func setupTestCatalog() *config.Catalog {
	cat := &config.Catalog{
		Dialect: config.DialectPostgres,
		Tables: []config.Table{
			{
				Name:              "customers",
				EstimatedRowCount: 100,
				TenancyFilter:     "tenant_id",
				SoftDeleteFilter:  "deleted_at IS NULL",
				Columns: []config.Column{
					{Name: "id", DataType: "BIGINT", IsPrimaryKey: true},
					{Name: "name", DataType: "VARCHAR", Nullable: false},
					{Name: "mrr", DataType: "NUMERIC", Nullable: false},
					{Name: "tenant_id", DataType: "VARCHAR", IsTenancyKey: true},
					{Name: "deleted_at", DataType: "TIMESTAMP", Nullable: true, IsSoftDeleteKey: true},
				},
			},
			{
				Name:              "orders",
				EstimatedRowCount: 1000,
				TenancyFilter:     "tenant_id",
				Columns: []config.Column{
					{Name: "id", DataType: "BIGINT", IsPrimaryKey: true},
					{Name: "customer_id", DataType: "BIGINT", IsForeignKey: true, RefTable: "customers", RefColumn: "id"},
					{Name: "amount", DataType: "NUMERIC", Nullable: false},
					{Name: "discount_pct", DataType: "NUMERIC", Nullable: true},
					{Name: "tenant_id", DataType: "VARCHAR", IsTenancyKey: true},
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
	return cat
}

func setupTestRegistry() *metrics.Registry {
	dict := &metrics.SemanticDictionary{
		Version: "1.0",
		Metrics: []metrics.Metric{
			{
				Name:             "ARR",
				CertifiedFormula: "SUM(mrr * 12)",
				RequiredFilters:  []string{"status = 'active'"},
				SourceTables:     []string{"customers"},
				Certified:        true,
			},
		},
	}
	return metrics.NewRegistry(dict)
}

func TestEngineFullPipeline(t *testing.T) {
	cat := setupTestCatalog()
	reg := setupTestRegistry()
	cfg := config.DefaultConfig()

	eng, err := NewEngine(cfg, cat, reg)
	if err != nil {
		t.Fatalf("create engine error: %v", err)
	}

	// Query with join fan-out and unhandled null division
	candidateSQL := `SELECT c.id, SUM(c.mrr) AS total_mrr, SUM(o.amount) / c.mrr AS ratio
FROM customers AS c
INNER JOIN orders AS o ON c.id = o.customer_id
WHERE c.tenant_id = 'org_1' AND c.deleted_at IS NULL
GROUP BY c.id`

	report, err := eng.Verify(candidateSQL)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}

	if report.IsClean {
		t.Fatalf("expected query to fail falsification probes")
	}

	if report.RewrittenSQL == "" {
		t.Fatalf("expected rewritten SQL to be generated")
	}

	if report.Voucher == nil {
		t.Fatalf("expected signed voucher")
	}

	if report.Voucher.Status != proof.StatusFalsifiedRewritten {
		t.Errorf("expected status %v, got %v", proof.StatusFalsifiedRewritten, report.Voucher.Status)
	}

	// Verify cryptographic signature
	valid, err := proof.VerifyVoucher(report.Voucher, cfg.SecretKey)
	if err != nil || !valid {
		t.Fatalf("signature verification failed: %v", err)
	}

	t.Logf("Verification completed in %.2f ms", report.VerificationMs)
	if report.VerificationMs > 50.0 {
		t.Errorf("verification latency took %.2f ms, expected < 50ms", report.VerificationMs)
	}
}
