package dbt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportDbtProject(t *testing.T) {
	tmpDir := t.TempDir()

	schemaYml := `
version: 2
models:
  - name: customers
    description: "Enterprise customers"
    columns:
      - name: id
        tests:
          - unique
          - not_null
      - name: tenant_id
        tests:
          - not_null

  - name: orders
    columns:
      - name: id
        tests:
          - unique
      - name: customer_id
        tests:
          - relationships:
              to: ref('customers')
              field: id
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.yml"), []byte(schemaYml), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	semanticYml := `
version: 2
semantic_models:
  - name: orders_semantic
    model: ref('orders')
    measures:
      - name: revenue
        agg: sum
        expr: amount
    dimensions:
      - name: tenant_id
        type: categorical

metrics:
  - name: arr
    description: "Annual recurring revenue"
    type: simple
    type_params:
      measure: revenue
      expr: "revenue * 12"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "semantic_models.yml"), []byte(semanticYml), 0644); err != nil {
		t.Fatalf("write semantic: %v", err)
	}

	cat, dict, err := ImportDbtProject(tmpDir)
	if err != nil {
		t.Fatalf("import error: %v", err)
	}

	if len(cat.Tables) != 2 {
		t.Errorf("expected 2 tables, got %d", len(cat.Tables))
	}
	if len(cat.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(cat.Relationships))
	}

	if len(dict.Metrics) < 2 {
		t.Errorf("expected at least 2 metrics from measures and metrics, got %d", len(dict.Metrics))
	}
}
