package compose

type Service struct {
	Image       string            `yaml:"image"`
	Ports       []string          `yaml:"ports"`
	Environment map[string]string `yaml:"environment"`
	Labels      map[string]string `yaml:"labels"`
	DependsOn   []string          `yaml:"depends_on"`
	Volumes     []string          `yaml:"volumes"`
	Expose      []int             `yaml:"expose"`
}

type Compose struct {
	Version  string                 `yaml:"version"`
	Services map[string]Service     `yaml:"services"`
	Volumes  map[string]interface{} `yaml:"volumes"`
}

func NewCompose() *Compose {
	c := &Compose{}
	c.Services = make(map[string]Service)
	c.Volumes = make(map[string]interface{})
	return c
}
