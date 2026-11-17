package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadMetrics loads and initializes the metric dictionary from a YAML or JSON file.
func LoadMetrics(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metrics file: %w", err)
	}

	var dict SemanticDictionary
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &dict); err != nil {
			return nil, fmt.Errorf("parse metrics yaml: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &dict); err != nil {
			return nil, fmt.Errorf("parse metrics json: %w", err)
		}
	}

	reg := NewRegistry(&dict)
	return reg, nil
}
