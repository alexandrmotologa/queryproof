package ast

import (
	"strings"
	"testing"
)

func TestParseSimpleSelect(t *testing.T) {
	sql := "SELECT id, name, email FROM customers WHERE active = true"
	stmt, err := ParseSQL(sql)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stmt.Projections) != 3 {
		t.Errorf("expected 3 projections, got %d", len(stmt.Projections))
	}
	if stmt.From == nil || stmt.From.Primary.Name != "customers" {
		t.Errorf("expected FROM customers")
	}
	if stmt.Where == nil {
		t.Errorf("expected WHERE clause")
	}
}

func TestParseJoinWithAggregates(t *testing.T) {
	sql := `SELECT c.id, c.name, SUM(o.amount) AS total_revenue
FROM customers AS c
INNER JOIN orders AS o ON c.id = o.customer_id
WHERE o.status = 'completed'
GROUP BY c.id, c.name
ORDER BY total_revenue DESC
LIMIT 10`

	stmt, err := ParseSQL(sql)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(stmt.From.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(stmt.From.Joins))
	}
	join := stmt.From.Joins[0]
	if join.Type != JoinInner {
		t.Errorf("expected INNER JOIN, got %v", join.Type)
	}
	if join.Table.Name != "orders" || join.Table.Alias != "o" {
		t.Errorf("unexpected join table: %+v", join.Table)
	}

	tables := ExtractTables(stmt)
	if len(tables) < 2 {
		t.Errorf("expected at least 2 tables, got %d", len(tables))
	}

	aggs := ExtractAggregates(stmt)
	if len(aggs) != 1 || strings.ToUpper(aggs[0].Name) != "SUM" {
		t.Errorf("expected 1 SUM aggregate, got %+v", aggs)
	}
}

func TestParseCTEAndWindow(t *testing.T) {
	sql := `WITH regional_sales AS (
SELECT region, SUM(amount) AS regional_rev
FROM sales
GROUP BY region
)
SELECT region, regional_rev, RANK() OVER (ORDER BY regional_rev DESC) AS rnk
FROM regional_sales`

	stmt, err := ParseSQL(sql)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if stmt.With == nil || len(stmt.With.CTEs) != 1 {
		t.Fatalf("expected 1 CTE in WITH clause")
	}
	if stmt.With.CTEs[0].Name != "regional_sales" {
		t.Errorf("expected CTE name regional_sales, got %s", stmt.With.CTEs[0].Name)
	}
}

func TestRewriterNullDivision(t *testing.T) {
	sql := "SELECT revenue / users_count AS arpu FROM kpis"
	stmt, err := ParseSQL(sql)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	rewriter := NewRewriter(nil)
	rewriter.RewriteNullSafeDivision(stmt)

	mutations := rewriter.GetMutations()
	if len(mutations) == 0 {
		t.Fatalf("expected division mutation")
	}

	outSQL := stmt.ToSQL()
	if !strings.Contains(outSQL, "NULLIF") || !strings.Contains(outSQL, "COALESCE") {
		t.Errorf("expected NULLIF and COALESCE in rewritten SQL: %s", outSQL)
	}
}
