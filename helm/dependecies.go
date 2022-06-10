package helm

type Dependency struct {
	Name        string               `yaml:"name"`
	Version     string               `yaml:"version"`
	Repository  string               `yaml:"repository"`
	Environment *map[string]EnvValue `yaml:"environment,omitempty"`
}
