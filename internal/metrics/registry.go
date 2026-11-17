package metrics

import (
	"strings"
)

// Registry manages enterprise certified metrics in memory.
type Registry struct {
	dict      *SemanticDictionary
	metricMap map[string]*Metric
}

// NewRegistry initializes an in-memory metric registry.
func NewRegistry(dict *SemanticDictionary) *Registry {
	r := &Registry{
		dict:      dict,
		metricMap: make(map[string]*Metric),
	}
	for i := range dict.Metrics {
		m := &dict.Metrics[i]
		cleanName := strings.ToLower(strings.TrimSpace(m.Name))
		r.metricMap[cleanName] = m

		// Also register normalized key without spaces or underscores
		compactKey := strings.ReplaceAll(strings.ReplaceAll(cleanName, " ", ""), "_", "")
		r.metricMap[compactKey] = m
	}
	return r
}

// ListMetrics returns all certified metrics in the registry.
func (r *Registry) ListMetrics() []Metric {
	if r.dict == nil {
		return nil
	}
	return r.dict.Metrics
}

// GetMetric finds a metric by exact or compact name.
func (r *Registry) GetMetric(name string) *Metric {
	clean := strings.ToLower(strings.TrimSpace(name))
	if m, ok := r.metricMap[clean]; ok {
		return m
	}
	compact := strings.ReplaceAll(strings.ReplaceAll(clean, " ", ""), "_", "")
	if m, ok := r.metricMap[compact]; ok {
		return m
	}
	return nil
}

// FindMatchingMetric finds a metric either by projection alias (e.g. "arr", "net_churn")
// or by matching formula semantics.
func (r *Registry) FindMatchingMetric(alias string) *Metric {
	if m := r.GetMetric(alias); m != nil {
		return m
	}

	cleanAlias := strings.ToLower(strings.TrimSpace(alias))
	// Check if alias contains metric name (e.g. "total_arr" -> "arr")
	for name, m := range r.metricMap {
		if strings.Contains(cleanAlias, name) && len(name) >= 3 {
			return m
		}
	}

	return nil
}
