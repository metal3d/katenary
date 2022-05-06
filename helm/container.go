package helm

import (
	"katenary/logger"
	"strings"

	"github.com/compose-spec/compose-go/types"
)

type EnvValue interface{}

// ContainerPort represent a port mapping.
type ContainerPort struct {
	Name          string
	ContainerPort int `yaml:"containerPort"`
}

// Value represent a environment variable with name and value.
type Value struct {
	Name  string   `yaml:"name"`
	Value EnvValue `yaml:"value"`
}

// Container represent a container with name, image, and environment variables. It is used in Deployment.
type Container struct {
	Name          string                         `yaml:"name,omitempty"`
	Image         string                         `yaml:"image"`
	Ports         []*ContainerPort               `yaml:"ports,omitempty"`
	Env           []*Value                       `yaml:"env,omitempty"`
	EnvFrom       []map[string]map[string]string `yaml:"envFrom,omitempty"`
	Command       []string                       `yaml:"command,omitempty"`
	VolumeMounts  []interface{}                  `yaml:"volumeMounts,omitempty"`
	LivenessProbe *Probe                         `yaml:"livenessProbe,omitempty"`
}

// NewContainer creates a new container with name, image, labels and environment variables.
func NewContainer(name, image string, environment types.MappingWithEquals, labels map[string]string) *Container {
	container := &Container{
		Image:   image,
		Name:    name,
		Env:     make([]*Value, len(environment)),
		EnvFrom: make([]map[string]map[string]string, 0),
	}

	// find bound environment variable to a service
	toServices := make([]string, 0)
	if bound, ok := labels[LABEL_ENV_SERVICE]; ok {
		toServices = strings.Split(bound, ",")
	}
	if len(toServices) > 0 {
		// warn, it's deprecated now
		logger.ActivateColors = true
		logger.Yellowf(
			"[deprecated] in \"%s\" service: label %s is deprecated, please use %s instead\n",
			name,
			LABEL_ENV_SERVICE,
			LABEL_MAP_ENV,
		)
		logger.ActivateColors = false
	}

	idx := 0
	for n, v := range environment {
		for _, name := range toServices {
			if name == n {
				*v = ReleaseNameTpl + "-" + *v
			}
		}
		container.Env[idx] = &Value{Name: n, Value: v}
		idx++
	}
	return container
}
