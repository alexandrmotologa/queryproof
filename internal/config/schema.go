package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Dialect represents the supported SQL target dialect.
type Dialect string

const (
	DialectPostgres  Dialect = "postgres"
	DialectSnowflake Dialect = "snowflake"
	DialectBigQuery  Dialect = "bigquery"
)

// CardinalityType represents table relationship cardinalities.
type CardinalityType string

const (
	CardOneToOne   CardinalityType = "1:1"
	CardOneToMany  CardinalityType = "1:N"
	CardManyToOne  CardinalityType = "N:1"
	CardManyToMany CardinalityType = "N:M"
)

// Column represents metadata for a database column.
type Column struct {
	Name            string `json:"name" yaml:"name"`
	DataType        string `json:"data_type" yaml:"data_type"`
	Nullable        bool   `json:"nullable" yaml:"nullable"`
	IsPrimaryKey    bool   `json:"is_primary_key" yaml:"is_primary_key"`
	IsForeignKey    bool   `json:"is_foreign_key" yaml:"is_foreign_key"`
	RefTable        string `json:"ref_table,omitempty" yaml:"ref_table,omitempty"`
	RefColumn       string `json:"ref_column,omitempty" yaml:"ref_column,omitempty"`
	IsPartitionKey  bool   `json:"is_partition_key" yaml:"is_partition_key"`
	IsTenancyKey    bool   `json:"is_tenancy_key" yaml:"is_tenancy_key"`
	IsSoftDeleteKey bool   `json:"is_soft_delete_key" yaml:"is_soft_delete_key"`
	Description     string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Relationship defines relationship cardinality between two tables.
type Relationship struct {
	FromTable   string          `json:"from_table" yaml:"from_table"`
	FromColumn  string          `json:"from_column" yaml:"from_column"`
	ToTable     string          `json:"to_table" yaml:"to_table"`
	ToColumn    string          `json:"to_column" yaml:"to_column"`
	Cardinality CardinalityType `json:"cardinality" yaml:"cardinality"`
	FanoutRate  float64         `json:"fanout_rate,omitempty" yaml:"fanout_rate,omitempty"` // Average child rows per parent row
}

// Table represents a database table in the catalog.
type Table struct {
	Name              string            `json:"name" yaml:"name"`
	Schema            string            `json:"schema,omitempty" yaml:"schema,omitempty"`
	Columns           []Column          `json:"columns" yaml:"columns"`
	EstimatedRowCount int64             `json:"estimated_row_count,omitempty" yaml:"estimated_row_count,omitempty"`
	TenancyFilter     string            `json:"tenancy_filter,omitempty" yaml:"tenancy_filter,omitempty"` // e.g. "tenant_id"
	SoftDeleteFilter  string            `json:"soft_delete_filter,omitempty" yaml:"soft_delete_filter,omitempty"` // e.g. "deleted_at IS NULL"
	PartitionColumn   string            `json:"partition_column,omitempty" yaml:"partition_column,omitempty"` // e.g. "_PARTITIONDATE"
	colMap            map[string]Column `json:"-" yaml:"-"`
}

// Catalog represents the complete database catalog for verification.
type Catalog struct {
	Dialect       Dialect        `json:"dialect" yaml:"dialect"`
	DefaultSchema string         `json:"default_schema,omitempty" yaml:"default_schema,omitempty"`
	Tables        []Table        `json:"tables" yaml:"tables"`
	Relationships []Relationship `json:"relationships,omitempty" yaml:"relationships,omitempty"`
	tableMap      map[string]*Table
}

// LoadCatalog loads a schema catalog from a JSON or YAML file.
func LoadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog file: %w", err)
	}

	var cat Catalog
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &cat); err != nil {
			return nil, fmt.Errorf("parse catalog yaml: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &cat); err != nil {
			return nil, fmt.Errorf("parse catalog json: %w", err)
		}
	}

	if cat.Dialect == "" {
		cat.Dialect = DialectPostgres
	}

	cat.Init()
	return &cat, nil
}

// Init initializes fast lookup maps inside the catalog.
func (c *Catalog) Init() {
	c.tableMap = make(map[string]*Table, len(c.Tables))
	for i := range c.Tables {
		tbl := &c.Tables[i]
		tbl.colMap = make(map[string]Column, len(tbl.Columns))
		for _, col := range tbl.Columns {
			tbl.colMap[strings.ToLower(col.Name)] = col
		}
		c.tableMap[strings.ToLower(tbl.Name)] = tbl
	}
}

// FindTable looks up a table by name (case-insensitive).
func (c *Catalog) FindTable(name string) *Table {
	if c.tableMap == nil {
		c.Init()
	}
	// Strip schema prefix if given, or match full/short
	clean := strings.ToLower(strings.Trim(name, `"`+"`"))
	parts := strings.Split(clean, ".")
	shortName := parts[len(parts)-1]

	if tbl, ok := c.tableMap[clean]; ok {
		return tbl
	}
	if tbl, ok := c.tableMap[shortName]; ok {
		return tbl
	}
	return nil
}

// FindColumn finds column metadata for a given table and column name.
func (c *Catalog) FindColumn(tableName, colName string) *Column {
	tbl := c.FindTable(tableName)
	if tbl == nil {
		return nil
	}
	cleanCol := strings.ToLower(strings.Trim(colName, `"`+"`"))
	if col, ok := tbl.colMap[cleanCol]; ok {
		return &col
	}
	return nil
}

// CheckJoinFanout determines if joining tableA with tableB causes fanout,
// and returns the expected multiplier Φ_join.
func (c *Catalog) CheckJoinFanout(parentTable, childTable, parentCol, childCol string) (isFanout bool, multiplier float64) {
	parentClean := strings.ToLower(strings.Trim(parentTable, `"`+"`"))
	childClean := strings.ToLower(strings.Trim(childTable, `"`+"`"))
	pColClean := strings.ToLower(strings.Trim(parentCol, `"`+"`"))
	cColClean := strings.ToLower(strings.Trim(childCol, `"`+"`"))

	// 1. Check explicit relationships first
	for _, rel := range c.Relationships {
		fromT := strings.ToLower(rel.FromTable)
		toT := strings.ToLower(rel.ToTable)

		if fromT == parentClean && toT == childClean {
			if rel.Cardinality == CardOneToMany || rel.Cardinality == CardManyToMany {
				fanout := rel.FanoutRate
				if fanout <= 1.0 {
					fanout = 3.5 // Default typical enterprise 1:N fanout multiplier
				}
				return true, fanout
			}
		} else if fromT == childClean && toT == parentClean {
			if rel.Cardinality == CardManyToOne || rel.Cardinality == CardManyToMany {
				fanout := rel.FanoutRate
				if fanout <= 1.0 {
					fanout = 3.5
				}
				return true, fanout
			}
		}
	}

	// 2. Inferred from foreign keys and primary keys
	parentTbl := c.FindTable(parentClean)
	childTbl := c.FindTable(childClean)
	if parentTbl != nil && childTbl != nil {
		pCol := c.FindColumn(parentClean, pColClean)
		cCol := c.FindColumn(childClean, cColClean)

		// If parentCol is primary key in parent, but childCol is NOT primary key in child,
		// this is classic 1:N relationship (e.g. customers.id = orders.customer_id)
		if pCol != nil && pCol.IsPrimaryKey && (cCol == nil || !cCol.IsPrimaryKey) {
			mult := 3.0
			if parentTbl.EstimatedRowCount > 0 && childTbl.EstimatedRowCount > 0 {
				ratio := float64(childTbl.EstimatedRowCount) / float64(parentTbl.EstimatedRowCount)
				if ratio > 1.0 {
					mult = ratio
				}
			}
			return true, mult
		}

		// Also check if childCol references parentTable
		if cCol != nil && cCol.IsForeignKey && strings.EqualFold(cCol.RefTable, parentClean) {
			return true, 3.0
		}
	}

	return false, 1.0
}
