package helm

// Service is a Kubernetes service.
type Service struct {
	*K8sBase `yaml:",inline"`
	Spec     *ServiceSpec `yaml:"spec"`
}

// NewService creates a new initialized service.
func NewService(name string) *Service {
	s := &Service{
		K8sBase: NewBase(),
		Spec:    NewServiceSpec(),
	}
	s.K8sBase.Metadata.Name = ReleaseNameTpl + "-" + name
	s.K8sBase.Kind = "Service"
	s.K8sBase.ApiVersion = "v1"
	s.K8sBase.Metadata.Labels[K+"/component"] = name
	return s
}

// ServicePort is a port on a service.
type ServicePort struct {
	Protocol   string `yaml:"protocol"`
	Port       int    `yaml:"port"`
	TargetPort int    `yaml:"targetPort"`
}

// NewServicePort creates a new initialized service port.
func NewServicePort(port, target int) *ServicePort {
	return &ServicePort{
		Protocol:   "TCP",
		Port:       port,
		TargetPort: port,
	}
}

// ServiceSpec is the spec for a service.
type ServiceSpec struct {
	Selector map[string]string
	Ports    []*ServicePort
	Type     string `yaml:"type,omitempty"`
}

// NewServiceSpec creates a new initialized service spec.
func NewServiceSpec() *ServiceSpec {
	return &ServiceSpec{
		Selector: make(map[string]string),
		Ports:    make([]*ServicePort, 0),
	}
}
