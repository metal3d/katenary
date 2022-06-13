package helm

// Dependency represents a Helm Dependency to append to the chart, it replace
// the docker service when Katenary converts the compose file.
type Dependency struct {
	Name       string                   `yaml:"name"`
	Version    string                   `yaml:"version"`
	Repository string                   `yaml:"repository"`
	Alias      string                   `yaml:"alias,omitempty"`
	Config     *DependencyConfiguration `yaml:"config,omitempty"`
}

// DependencyConfiguration is the configuration of the dependency. It provides
// environment and service name.
type DependencyConfiguration struct {
	Environment *map[string]EnvValue `yaml:"environment"`
	ServiceName string               `yaml:"serviceName"`
}
