package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProbeConfig controls sensitivity and severity for each falsification probe.
type ProbeConfig struct {
	EnableFanoutGuard     bool    `json:"enable_fanout_guard" yaml:"enable_fanout_guard"`
	FanoutMaxTolerance    float64 `json:"fanout_max_tolerance" yaml:"fanout_max_tolerance"` // default 1.05
	EnableNullabilityTest bool    `json:"enable_nullability_test" yaml:"enable_nullability_test"`
	EnableMetricCheck     bool    `json:"enable_metric_check" yaml:"enable_metric_check"`
	EnableGovernanceGuard bool    `json:"enable_governance_guard" yaml:"enable_governance_guard"`
	EnablePartitionGuard  bool    `json:"enable_partition_guard" yaml:"enable_partition_guard"`
	EnforceTenantFilter   bool    `json:"enforce_tenant_filter" yaml:"enforce_tenant_filter"`
	EnforceSoftDelete     bool    `json:"enforce_soft_delete" yaml:"enforce_soft_delete"`
}

// ServerConfig configures the embedded React 19 BI Auditor HTTP server.
type ServerConfig struct {
	Port int    `json:"port" yaml:"port"`
	Host string `json:"host" yaml:"host"`
	CORS bool   `json:"cors" yaml:"cors"`
}

// Config represents root configuration for queryproof execution.
type Config struct {
	Dialect     Dialect     `json:"dialect" yaml:"dialect"`
	CatalogPath string      `json:"catalog_path" yaml:"catalog_path"`
	MetricsPath string      `json:"metrics_path" yaml:"metrics_path"`
	Probes      ProbeConfig `json:"probes" yaml:"probes"`
	Server      ServerConfig`json:"server" yaml:"server"`
	SecretKey   string      `json:"secret_key,omitempty" yaml:"secret_key,omitempty"` // For HMAC proof signing
}

// DefaultConfig provides recommended production defaults.
func DefaultConfig() *Config {
	return &Config{
		Dialect:     DialectPostgres,
		CatalogPath: "catalog.json",
		MetricsPath: "metrics.yaml",
		Probes: ProbeConfig{
			EnableFanoutGuard:     true,
			FanoutMaxTolerance:    1.0,
			EnableNullabilityTest: true,
			EnableMetricCheck:     true,
			EnableGovernanceGuard: true,
			EnablePartitionGuard:  true,
			EnforceTenantFilter:   true,
			EnforceSoftDelete:     true,
		},
		Server: ServerConfig{
			Port: 7373,
			Host: "127.0.0.1",
			CORS: true,
		},
		SecretKey: "queryproof-enterprise-master-key",
	}
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}
	return cfg, nil
}
