package warehouse

import (
	"testing"

	"github.com/alexandrmotologa/queryproof/internal/config"
)

func TestInMemorySandboxExecution(t *testing.T) {
	cat := &config.Catalog{
		Dialect: config.DialectPostgres,
		Tables: []config.Table{
			{
				Name: "customers",
				Columns: []config.Column{
					{Name: "id", DataType: "BIGINT", IsPrimaryKey: true},
					{Name: "name", DataType: "TEXT"},
					{Name: "mrr", DataType: "REAL"},
					{Name: "tenant_id", DataType: "TEXT"},
					{Name: "deleted_at", DataType: "TEXT"},
				},
			},
			{
				Name: "orders",
				Columns: []config.Column{
					{Name: "id", DataType: "BIGINT", IsPrimaryKey: true},
					{Name: "customer_id", DataType: "BIGINT"},
					{Name: "amount", DataType: "REAL"},
					{Name: "discount_pct", DataType: "REAL"},
					{Name: "tenant_id", DataType: "TEXT"},
				},
			},
		},
	}
	cat.Init()

	sb, err := NewInMemorySandbox(cat)
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	stats, err := sb.ExecuteQuery("SELECT count(*) FROM customers")
	if err != nil {
		t.Fatalf("query error: %v", err)
	}

	if stats.RowCount == 0 {
		t.Errorf("expected rows to be returned")
	}
	if stats.DurationMs < 0 {
		t.Errorf("invalid duration: %f", stats.DurationMs)
	}
}
