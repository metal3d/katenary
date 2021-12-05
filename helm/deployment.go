package helm

import "strings"

// Deployment is a k8s deployment.
type Deployment struct {
	*K8sBase `yaml:",inline"`
	Spec     *DepSpec `yaml:"spec"`
}

func NewDeployment(name string) *Deployment {
	d := &Deployment{K8sBase: NewBase(), Spec: NewDepSpec()}
	d.K8sBase.Metadata.Name = RELEASE_NAME + "-" + name
	d.K8sBase.ApiVersion = "apps/v1"
	d.K8sBase.Kind = "Deployment"
	d.K8sBase.Metadata.Labels[K+"/component"] = name
	return d
}

type DepSpec struct {
	Replicas int                    `yaml:"replicas"`
	Selector map[string]interface{} `yaml:"selector"`
	Template PodTemplate            `yaml:"template"`
}

func NewDepSpec() *DepSpec {
	return &DepSpec{
		Replicas: 1,
	}
}

type Value struct {
	Name  string      `yaml:"name"`
	Value interface{} `yaml:"value"`
}

type ContainerPort struct {
	Name          string
	ContainerPort int `yaml:"containerPort"`
}

type Container struct {
	Name         string                         `yaml:"name,omitempty"`
	Image        string                         `yaml:"image"`
	Ports        []*ContainerPort               `yaml:"ports,omitempty"`
	Env          []Value                        `yaml:"env,omitempty"`
	EnvFrom      []map[string]map[string]string `yaml:"envFrom,omitempty"`
	Command      []string                       `yaml:"command,omitempty"`
	VolumeMounts []interface{}                  `yaml:"volumeMounts,omitempty"`
}

func NewContainer(name, image string, environment, labels map[string]string) *Container {
	container := &Container{
		Image:   image,
		Name:    name,
		Env:     make([]Value, len(environment)),
		EnvFrom: make([]map[string]map[string]string, 0),
	}

	// find bound environment variable to a service
	toServices := make([]string, 0)
	if bound, ok := labels[LABEL_ENV_SERVICE]; ok {
		toServices = strings.Split(bound, ",")
	}

	idx := 0
	for n, v := range environment {
		for _, name := range toServices {
			if name == n {
				v = RELEASE_NAME + "-" + v
			}
		}
		container.Env[idx] = Value{Name: n, Value: v}
		idx++
	}
	return container
}

type PodSpec struct {
	InitContainers []*Container             `yaml:"initContainers,omitempty"`
	Containers     []*Container             `yaml:"containers"`
	Volumes        []map[string]interface{} `yaml:"volumes,omitempty"`
}

type PodTemplate struct {
	Metadata Metadata `yaml:"metadata"`
	Spec     PodSpec  `yaml:"spec"`
}
