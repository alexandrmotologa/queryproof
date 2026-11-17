package introspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/config"
	_ "modernc.org/sqlite"
)

// Extractor inspects a live database and extracts a queryproof Catalog.
type Extractor interface {
	Extract(ctx context.Context, dsn string) (*config.Catalog, error)
}

// ExtractFromDatabase inspects a database using the specified driver.
func ExtractFromDatabase(ctx context.Context, driver, dsn string) (*config.Catalog, error) {
	switch strings.ToLower(driver) {
	case "sqlite", "sqlite3":
		return extractSQLite(ctx, dsn)
	case "postgres", "postgresql":
		return extractPostgres(ctx, dsn)
	default:
		return nil, fmt.Errorf("unsupported driver %q (supported: sqlite, postgres)", driver)
	}
}

func extractSQLite(ctx context.Context, dsn string) (*config.Catalog, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	defer db.Close()

	// 1. Get all user tables
	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';")
	if err != nil {
		return nil, fmt.Errorf("list sqlite tables: %w", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tableNames = append(tableNames, name)
		}
	}

	cat := &config.Catalog{
		Dialect:       config.DialectPostgres,
		DefaultSchema: "main",
		Tables:        make([]config.Table, 0, len(tableNames)),
		Relationships: make([]config.Relationship, 0),
	}

	for _, tblName := range tableNames {
		tbl := config.Table{
			Name:    tblName,
			Schema:  "main",
			Columns: make([]config.Column, 0),
		}

		// Count approximate rows
		var rowCount int64
		_ = db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", tblName)).Scan(&rowCount)
		tbl.EstimatedRowCount = rowCount

		// Get column info
		colRows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s);", tblName))
		if err == nil {
			for colRows.Next() {
				var cid int
				var colName, colType string
				var notNull, pk int
				var dfltValue sql.NullString

				if err := colRows.Scan(&cid, &colName, &colType, &notNull, &dfltValue, &pk); err == nil {
					col := config.Column{
						Name:         colName,
						DataType:     colType,
						Nullable:     (notNull == 0),
						IsPrimaryKey: (pk > 0),
					}
					enrichColumnGovernance(&col)
					tbl.Columns = append(tbl.Columns, col)
				}
			}
			colRows.Close()
		}

		// Get foreign keys
		fkRows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%s);", tblName))
		if err == nil {
			for fkRows.Next() {
				var id, seq int
				var refTable, fromCol, toCol, onUpdate, onDelete, match string
				if err := fkRows.Scan(&id, &seq, &refTable, &fromCol, &toCol, &onUpdate, &onDelete, &match); err == nil {
					// Mark column as FK
					for i := range tbl.Columns {
						if strings.EqualFold(tbl.Columns[i].Name, fromCol) {
							tbl.Columns[i].IsForeignKey = true
							tbl.Columns[i].RefTable = refTable
							tbl.Columns[i].RefColumn = toCol
						}
					}
					// Add 1:N relationship
					cat.Relationships = append(cat.Relationships, config.Relationship{
						FromTable:   refTable,
						FromColumn:  toCol,
						ToTable:     tblName,
						ToColumn:    fromCol,
						Cardinality: config.CardOneToMany,
						FanoutRate:  3.5,
					})
				}
			}
			fkRows.Close()
		}

		// Set table-level governance filters if detected
		enrichTableGovernance(&tbl)
		cat.Tables = append(cat.Tables, tbl)
	}

	cat.Init()
	return cat, nil
}

func extractPostgres(ctx context.Context, dsn string) (*config.Catalog, error) {
	// For environments with a postgres connection pool
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()

	query := `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_schema NOT IN ('information_schema', 'pg_catalog')
		  AND table_type = 'BASE TABLE';
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query postgres tables: %w", err)
	}
	defer rows.Close()

	cat := &config.Catalog{
		Dialect:       config.DialectPostgres,
		DefaultSchema: "public",
		Tables:        make([]config.Table, 0),
		Relationships: make([]config.Relationship, 0),
	}

	for rows.Next() {
		var schema, name string
		if err := rows.Scan(&schema, &name); err == nil {
			tbl := config.Table{
				Name:    name,
				Schema:  schema,
				Columns: make([]config.Column, 0),
			}
			enrichTableGovernance(&tbl)
			cat.Tables = append(cat.Tables, tbl)
		}
	}

	cat.Init()
	return cat, nil
}

func enrichColumnGovernance(col *config.Column) {
	lower := strings.ToLower(col.Name)
	if lower == "tenant_id" || lower == "organization_id" || lower == "account_id" {
		col.IsTenancyKey = true
	}
	if lower == "deleted_at" || lower == "is_deleted" || lower == "archived_at" {
		col.IsSoftDeleteKey = true
	}
	if lower == "event_date" || lower == "created_date" || lower == "_partitiondate" || lower == "dt" {
		col.IsPartitionKey = true
	}
}

func enrichTableGovernance(tbl *config.Table) {
	for _, col := range tbl.Columns {
		if col.IsTenancyKey && tbl.TenancyFilter == "" {
			tbl.TenancyFilter = col.Name
		}
		if col.IsSoftDeleteKey && tbl.SoftDeleteFilter == "" {
			if strings.Contains(strings.ToLower(col.DataType), "bool") {
				tbl.SoftDeleteFilter = fmt.Sprintf("%s = false", col.Name)
			} else {
				tbl.SoftDeleteFilter = fmt.Sprintf("%s IS NULL", col.Name)
			}
		}
		if col.IsPartitionKey && tbl.PartitionColumn == "" {
			tbl.PartitionColumn = col.Name
		}
	}
}
