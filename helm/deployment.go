package helm

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

type PodSpec struct {
	InitContainers []*Container             `yaml:"initContainers,omitempty"`
	Containers     []*Container             `yaml:"containers"`
	Volumes        []map[string]interface{} `yaml:"volumes,omitempty"`
}

type PodTemplate struct {
	Metadata Metadata `yaml:"metadata"`
	Spec     PodSpec  `yaml:"spec"`
}
