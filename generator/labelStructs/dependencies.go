package labelStructs

import "gopkg.in/yaml.v3"

// Dependency is a dependency of a chart to other charts.
type Dependency struct {
	Name       string         `yaml:"name"`
	Version    string         `yaml:"version"`
	Repository string         `yaml:"repository"`
	Alias      string         `yaml:"alias,omitempty"`
	Values     map[string]any `yaml:"-"` // do not export to Chart.yaml
}

// DependenciesFrom returns a slice of dependencies from the given string.
func DependenciesFrom(data string) ([]Dependency, error) {
	var mapping []Dependency
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}
