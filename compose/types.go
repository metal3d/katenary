package compose

// Compose is a complete docker-compse representation.
type Compose struct {
	Version  string                 `yaml:"version"`
	Services map[string]*Service    `yaml:"services"`
	Volumes  map[string]interface{} `yaml:"volumes"`
}

// NewCompose resturs a Compose object.
func NewCompose() *Compose {
	c := &Compose{}
	c.Services = make(map[string]*Service)
	c.Volumes = make(map[string]interface{})
	return c
}

// HealthCheck manage generic type to handle TCP, HTTP and TCP health check.
type HealthCheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	Retries     int      `yaml:"retries"`
	StartPeriod string   `yaml:"start_period"`
}

// Service represent a "service" in a docker-compose file.
type Service struct {
	Image       string            `yaml:"image"`
	Ports       []string          `yaml:"ports"`
	Environment map[string]string `yaml:"environment"`
	Labels      map[string]string `yaml:"labels"`
	DependsOn   []string          `yaml:"depends_on"`
	Volumes     []string          `yaml:"volumes"`
	Expose      []int             `yaml:"expose"`
	EnvFiles    []string          `yaml:"env_file"`
	RawBuild    interface{}       `yaml:"build"`
	HealthCheck *HealthCheck      `yaml:"healthcheck"`
}
