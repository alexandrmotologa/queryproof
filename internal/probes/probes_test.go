package probes

import (
	"testing"

	"github.com/alexandrmotologa/queryproof/internal/ast"
	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/metrics"
)

func setupTestCatalog() *config.Catalog {
	cat := &config.Catalog{
		Dialect: config.DialectPostgres,
		Tables: []config.Table{
			{
				Name:              "customers",
				EstimatedRowCount: 1000,
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
				EstimatedRowCount: 10000,
				TenancyFilter:     "tenant_id",
				Columns: []config.Column{
					{Name: "id", DataType: "BIGINT", IsPrimaryKey: true},
					{Name: "customer_id", DataType: "BIGINT", IsForeignKey: true, RefTable: "customers", RefColumn: "id"},
					{Name: "amount", DataType: "NUMERIC", Nullable: false},
					{Name: "discount_pct", DataType: "NUMERIC", Nullable: true},
					{Name: "tenant_id", DataType: "VARCHAR", IsTenancyKey: true},
				},
			},
			{
				Name:              "analytics_events",
				EstimatedRowCount: 50000000,
				PartitionColumn:   "event_date",
				Columns: []config.Column{
					{Name: "id", DataType: "VARCHAR", IsPrimaryKey: true},
					{Name: "event_date", DataType: "DATE", IsPartitionKey: true},
					{Name: "action", DataType: "VARCHAR"},
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

func TestFanoutProbeDetectsCartesianInflation(t *testing.T) {
	cat := setupTestCatalog()
	sql := `SELECT c.id, SUM(c.mrr) AS total_mrr, COUNT(o.id) AS order_cnt
FROM customers AS c
INNER JOIN orders AS o ON c.id = o.customer_id
WHERE c.tenant_id = 'org_1' AND c.deleted_at IS NULL
GROUP BY c.id`

	stmt, err := ast.ParseSQL(sql)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ctx := &EvaluationContext{
		AST:     stmt,
		RawSQL:  sql,
		Catalog: cat,
	}

	probe := &FanoutProbe{}
	res, err := probe.Evaluate(ctx)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}

	if res.Passed {
		t.Fatalf("expected fanout probe to fail due to SUM(c.mrr) over 1:N orders join")
	}
	if len(res.Violations) == 0 {
		t.Fatalf("expected violation record")
	}
	if res.Violations[0].Severity != SeverityCritical {
		t.Errorf("expected CRITICAL severity, got %v", res.Violations[0].Severity)
	}
}

func TestNullsProbeDetectsDistortions(t *testing.T) {
	cat := setupTestCatalog()
	sql := `SELECT AVG(o.discount_pct) AS avg_discount, SUM(o.amount) / o.amount AS ratio
FROM orders AS o
WHERE o.tenant_id = 'org_1'`

	stmt, err := ast.ParseSQL(sql)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ctx := &EvaluationContext{
		AST:     stmt,
		RawSQL:  sql,
		Catalog: cat,
	}

	probe := &NullsProbe{}
	res, err := probe.Evaluate(ctx)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}

	if res.Passed {
		t.Fatalf("expected nulls probe to fail on AVG(nullable) and unprotected division")
	}
	if len(res.Violations) < 2 {
		t.Errorf("expected at least 2 violations, got %d", len(res.Violations))
	}
}

func TestGovernanceProbeCatchesMissingTenantAndSoftDelete(t *testing.T) {
	cat := setupTestCatalog()
	sql := `SELECT id, name FROM customers`

	stmt, err := ast.ParseSQL(sql)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ctx := &EvaluationContext{
		AST:     stmt,
		RawSQL:  sql,
		Catalog: cat,
	}

	probe := &GovernanceProbe{}
	res, err := probe.Evaluate(ctx)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}

	if res.Passed {
		t.Fatalf("expected governance probe to fail on missing tenant_id and deleted_at filter")
	}

	hasTenant := false
	hasSoftDelete := false
	for _, v := range res.Violations {
		if v.Severity == SeverityCritical {
			hasTenant = true
		}
		if v.Severity == SeverityHigh {
			hasSoftDelete = true
		}
	}

	if !hasTenant || !hasSoftDelete {
		t.Errorf("expected both tenant and soft delete violations: tenant=%v, softDelete=%v", hasTenant, hasSoftDelete)
	}
}

func TestSuiteExecution(t *testing.T) {
	cat := setupTestCatalog()
	reg := setupTestRegistry()

	sql := `SELECT c.id, SUM(c.mrr) AS total_mrr, AVG(o.discount_pct) AS avg_discount
FROM customers AS c
INNER JOIN orders AS o ON c.id = o.customer_id`

	stmt, err := ast.ParseSQL(sql)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ctx := &EvaluationContext{
		AST:      stmt,
		RawSQL:   sql,
		Catalog:  cat,
		Registry: reg,
	}

	suite := NewDefaultSuite(nil)
	sr, err := suite.Run(ctx)
	if err != nil {
		t.Fatalf("suite run error: %v", err)
	}

	if sr.Passed {
		t.Fatalf("expected suite to fail on invalid query")
	}
	if sr.CriticalCount == 0 {
		t.Errorf("expected critical violations")
	}
}
