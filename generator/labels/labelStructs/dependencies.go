package labelStructs

import "gopkg.in/yaml.v3"

// Dependency is a dependency of a chart to other charts.
type Dependency struct {
	Values     map[string]any `yaml:"-" json:"values,omitempty"`
	Name       string         `yaml:"name" json:"name"`
	Version    string         `yaml:"version" json:"version"`
	Repository string         `yaml:"repository" json:"repository"`
	Alias      string         `yaml:"alias,omitempty" json:"alias,omitempty"`
}

// DependenciesFrom returns a slice of dependencies from the given string.
func DependenciesFrom(data string) ([]Dependency, error) {
	var mapping []Dependency
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}
