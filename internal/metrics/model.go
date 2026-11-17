package metrics

// Metric represents an enterprise certified business metric definition.
type Metric struct {
	Name               string   `json:"name" yaml:"name"`
	Description        string   `json:"description" yaml:"description"`
	CertifiedFormula   string   `json:"certified_formula" yaml:"certified_formula"`
	SourceTables       []string `json:"source_tables" yaml:"source_tables"`
	RequiredFilters    []string `json:"required_filters,omitempty" yaml:"required_filters,omitempty"`
	AllowedDimensions  []string `json:"allowed_dimensions,omitempty" yaml:"allowed_dimensions,omitempty"`
	Owner              string   `json:"owner,omitempty" yaml:"owner,omitempty"`
	Certified          bool     `json:"certified" yaml:"certified"`
	DialectSpecificSQL map[string]string `json:"dialect_specific_sql,omitempty" yaml:"dialect_specific_sql,omitempty"`
}

// SemanticDictionary is the top-level container for metrics.yaml.
type SemanticDictionary struct {
	Version     string   `json:"version" yaml:"version"`
	UpdatedAt   string   `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Metrics     []Metric `json:"metrics" yaml:"metrics"`
}
