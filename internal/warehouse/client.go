package warehouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/alexandrmotologa/queryproof/internal/config"
	_ "modernc.org/sqlite"
)

// ExecutionStats records timing and cardinality from running a query in the sandbox.
type ExecutionStats struct {
	RowCount     int64         `json:"row_count"`
	Duration     time.Duration `json:"duration"`
	DurationMs   float64       `json:"duration_ms"`
	PlanSummary  string        `json:"plan_summary,omitempty"`
	SampleResult []string      `json:"sample_result,omitempty"`
}

// Sandbox represents a local or remote warehouse execution engine.
type Sandbox struct {
	db      *sql.DB
	dialect config.Dialect
}

// NewInMemorySandbox initializes an in-memory database populated with schema tables and sample enterprise data.
func NewInMemorySandbox(cat *config.Catalog) (*Sandbox, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open sqlite memory: %w", err)
	}

	sb := &Sandbox{db: db, dialect: config.DialectPostgres}
	if cat != nil {
		if err := sb.bootstrapCatalog(cat); err != nil {
			db.Close()
			return nil, err
		}
	}

	return sb, nil
}

func (s *Sandbox) bootstrapCatalog(cat *config.Catalog) error {
	ctx := context.Background()

	// Seed basic enterprise tables if matching names exist
	for _, tbl := range cat.Tables {
		var colDefs []string
		for _, col := range tbl.Columns {
			colType := "TEXT"
			dt := strings.ToUpper(col.DataType)
			if strings.Contains(dt, "INT") {
				colType = "INTEGER"
			} else if strings.Contains(dt, "NUMERIC") || strings.Contains(dt, "FLOAT") || strings.Contains(dt, "DOUBLE") || strings.Contains(dt, "DECIMAL") {
				colType = "REAL"
			}

			def := fmt.Sprintf("%s %s", col.Name, colType)
			if col.IsPrimaryKey {
				def += " PRIMARY KEY"
			}
			colDefs = append(colDefs, def)
		}

		createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", tbl.Name, strings.Join(colDefs, ", "))
		if _, err := s.db.ExecContext(ctx, createSQL); err != nil {
			return fmt.Errorf("create table %s: %w", tbl.Name, err)
		}
	}

	// Seed realistic mock records for customers and orders if present
	if cat.FindTable("customers") != nil {
		_, _ = s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO customers (id, name, mrr, tenant_id, deleted_at) VALUES
			(1, 'Acme Corp', 5000.0, 'org_1', NULL),
			(2, 'Globex Inc', 12000.0, 'org_1', NULL),
			(3, 'Soylent Corp', 3500.0, 'org_1', '2026-01-15'),
			(4, 'Initech', 8200.0, 'org_2', NULL);
		`)
	}

	if cat.FindTable("orders") != nil {
		_, _ = s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO orders (id, customer_id, amount, discount_pct, tenant_id) VALUES
			(101, 1, 1200.0, 0.10, 'org_1'),
			(102, 1, 1800.0, NULL, 'org_1'),
			(103, 1, 2500.0, 0.05, 'org_1'),
			(104, 2, 4000.0, 0.15, 'org_1'),
			(105, 2, 6000.0, NULL, 'org_1'),
			(106, 2, 2200.0, 0.20, 'org_1'),
			(107, 3, 1500.0, NULL, 'org_1');
		`)
	}

	return nil
}

// ExecuteQuery runs a query and measures exact latency and row count.
func (s *Sandbox) ExecuteQuery(query string) (*ExecutionStats, error) {
	start := time.Now()
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("execute query error: %w", err)
	}
	defer rows.Close()

	var count int64
	var sample []string
	cols, _ := rows.Columns()

	for rows.Next() {
		count++
		if count <= 5 {
			// Scan sample
			vals := make([]interface{}, len(cols))
			valPtrs := make([]interface{}, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}
			if err := rows.Scan(valPtrs...); err == nil {
				var rowStr []string
				for i, v := range vals {
					if v == nil {
						rowStr = append(rowStr, fmt.Sprintf("%s: NULL", cols[i]))
					} else {
						rowStr = append(rowStr, fmt.Sprintf("%s: %v", cols[i], v))
					}
				}
				sample = append(sample, strings.Join(rowStr, ", "))
			}
		}
	}

	dur := time.Since(start)
	return &ExecutionStats{
		RowCount:     count,
		Duration:     dur,
		DurationMs:   float64(dur.Microseconds()) / 1000.0,
		SampleResult: sample,
	}, nil
}

// Close closes the sandbox database connection.
func (s *Sandbox) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
