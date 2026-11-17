package introspect

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/queryproof/internal/config"
	_ "modernc.org/sqlite"
)

func TestExtractSQLiteSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_introspect.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create sqlite db: %v", err)
	}

	createSchema := `
		CREATE TABLE customers (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			deleted_at TEXT
		);

		CREATE TABLE orders (
			id INTEGER PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			amount REAL NOT NULL,
			tenant_id TEXT NOT NULL,
			FOREIGN KEY (customer_id) REFERENCES customers(id)
		);
	`
	if _, err := db.Exec(createSchema); err != nil {
		t.Fatalf("exec schema error: %v", err)
	}
	db.Close()

	cat, err := ExtractFromDatabase(context.Background(), "sqlite", dbPath)
	if err != nil {
		t.Fatalf("extract error: %v", err)
	}

	if len(cat.Tables) < 2 {
		t.Fatalf("expected at least 2 tables, got %d", len(cat.Tables))
	}

	cust := cat.FindTable("customers")
	if cust == nil {
		t.Fatalf("customers table not found")
	}

	if cust.TenancyFilter != "tenant_id" {
		t.Errorf("expected TenancyFilter tenant_id, got %q", cust.TenancyFilter)
	}
	if cust.SoftDeleteFilter != "deleted_at IS NULL" {
		t.Errorf("expected SoftDeleteFilter 'deleted_at IS NULL', got %q", cust.SoftDeleteFilter)
	}

	// Verify relationship
	if len(cat.Relationships) == 0 {
		t.Fatalf("expected at least 1 foreign key relationship")
	}
	rel := cat.Relationships[0]
	if rel.FromTable != "customers" || rel.ToTable != "orders" || rel.Cardinality != config.CardOneToMany {
		t.Errorf("unexpected relationship extracted: %+v", rel)
	}

	_ = os.Remove(dbPath)
}
