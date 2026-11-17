package dbt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/metrics"
	"gopkg.in/yaml.v3"
)

type DbtModelColumn struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description,omitempty"`
	Tests       []interface{} `yaml:"tests,omitempty"`
}

type DbtModel struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description,omitempty"`
	Columns     []DbtModelColumn `yaml:"columns,omitempty"`
}

type DbtMeasure struct {
	Name string `yaml:"name"`
	Agg  string `yaml:"agg"`
	Expr string `yaml:"expr,omitempty"`
}

type DbtDimension struct {
	Name string `yaml:"name"`
	Type string `yaml:"type,omitempty"`
}

type DbtSemanticModel struct {
	Name       string         `yaml:"name"`
	Model      string         `yaml:"model"`
	Measures   []DbtMeasure   `yaml:"measures,omitempty"`
	Dimensions []DbtDimension `yaml:"dimensions,omitempty"`
}

type DbtMetricParams struct {
	Measure string `yaml:"measure,omitempty"`
	Expr    string `yaml:"expr,omitempty"`
}

type DbtMetric struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description,omitempty"`
	Type        string          `yaml:"type"`
	TypeParams  DbtMetricParams `yaml:"type_params,omitempty"`
}

type DbtFile struct {
	Version        int                `yaml:"version,omitempty"`
	Models         []DbtModel         `yaml:"models,omitempty"`
	SemanticModels []DbtSemanticModel `yaml:"semantic_models,omitempty"`
	Metrics        []DbtMetric        `yaml:"metrics,omitempty"`
}

// ImportDbtProject traverses a dbt models directory and compiles catalog and metrics.
func ImportDbtProject(dirPath string) (*config.Catalog, *metrics.SemanticDictionary, error) {
	cat := &config.Catalog{
		Dialect:       config.DialectPostgres,
		Tables:        make([]config.Table, 0),
		Relationships: make([]config.Relationship, 0),
	}
	dict := &metrics.SemanticDictionary{
		Version:     "1.0.0",
		Description: "Imported from dbt project models",
		Metrics:     make([]metrics.Metric, 0),
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var dbt DbtFile
		if err := yaml.Unmarshal(data, &dbt); err != nil {
			return nil
		}

		// Process models
		for _, m := range dbt.Models {
			tbl := config.Table{
				Name:    m.Name,
				Columns: make([]config.Column, 0, len(m.Columns)),
			}

			for _, col := range m.Columns {
				c := config.Column{
					Name:     col.Name,
					DataType: "VARCHAR",
					Nullable: true,
				}

				// Check tests
				for _, t := range col.Tests {
					if testName, ok := t.(string); ok {
						if testName == "unique" || testName == "primary_key" {
							c.IsPrimaryKey = true
							c.Nullable = false
						} else if testName == "not_null" {
							c.Nullable = false
						}
					} else if testMap, ok := t.(map[string]interface{}); ok {
						if rel, hasRel := testMap["relationships"]; hasRel {
							if relMap, isMap := rel.(map[string]interface{}); isMap {
								targetModel := parseRef(fmt.Sprintf("%v", relMap["to"]))
								targetField := fmt.Sprintf("%v", relMap["field"])
								c.IsForeignKey = true
								c.RefTable = targetModel
								c.RefColumn = targetField

								cat.Relationships = append(cat.Relationships, config.Relationship{
									FromTable:   targetModel,
									FromColumn:  targetField,
									ToTable:     m.Name,
									ToColumn:    col.Name,
									Cardinality: config.CardOneToMany,
									FanoutRate:  5.0,
								})
							}
						}
					}
				}

				// Governance heuristics
				colLower := strings.ToLower(col.Name)
				if colLower == "tenant_id" || colLower == "organization_id" {
					c.IsTenancyKey = true
					tbl.TenancyFilter = col.Name
				}
				if colLower == "deleted_at" || colLower == "is_deleted" {
					c.IsSoftDeleteKey = true
					tbl.SoftDeleteFilter = fmt.Sprintf("%s IS NULL", col.Name)
				}
				if colLower == "event_date" || colLower == "_partitiondate" || colLower == "dt" {
					c.IsPartitionKey = true
					tbl.PartitionColumn = col.Name
				}

				tbl.Columns = append(tbl.Columns, c)
			}
			cat.Tables = append(cat.Tables, tbl)
		}

		// Process dbt semantic models and metrics
		for _, sm := range dbt.SemanticModels {
			sourceTable := parseRef(sm.Model)
			var allowedDims []string
			for _, d := range sm.Dimensions {
				allowedDims = append(allowedDims, d.Name)
			}

			for _, measure := range sm.Measures {
				expr := measure.Expr
				if expr == "" {
					expr = measure.Name
				}
				formula := fmt.Sprintf("%s(%s)", strings.ToUpper(measure.Agg), expr)
				dict.Metrics = append(dict.Metrics, metrics.Metric{
					Name:              measure.Name,
					CertifiedFormula:  formula,
					SourceTables:      []string{sourceTable},
					AllowedDimensions: allowedDims,
					Certified:         true,
					Owner:             "dbt Semantic Layer",
				})
			}
		}

		for _, met := range dbt.Metrics {
			formula := met.TypeParams.Expr
			if formula == "" {
				formula = fmt.Sprintf("SUM(%s)", met.TypeParams.Measure)
			}
			dict.Metrics = append(dict.Metrics, metrics.Metric{
				Name:             met.Name,
				Description:      met.Description,
				CertifiedFormula: formula,
				Certified:        true,
				Owner:            "dbt Semantic Layer",
			})
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	cat.Init()
	return cat, dict, nil
}

func parseRef(s string) string {
	// ref('customers') -> customers
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "ref(") && strings.HasSuffix(s, ")") {
		inner := strings.TrimPrefix(s, "ref(")
		inner = strings.TrimSuffix(inner, ")")
		inner = strings.Trim(inner, `"'`)
		return inner
	}
	return s
}
