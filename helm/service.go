package helm

type Service struct {
	*K8sBase `yaml:",inline"`
	Spec     *ServiceSpec `yaml:"spec"`
}

func NewService(name string) *Service {
	s := &Service{
		K8sBase: NewBase(),
		Spec:    NewServiceSpec(),
	}
	s.K8sBase.Metadata.Name = RELEASE_NAME + "-" + name
	s.K8sBase.Kind = "Service"
	s.K8sBase.ApiVersion = "v1"
	s.K8sBase.Metadata.Labels[K+"/component"] = name
	return s
}

type ServicePort struct {
	Protocol   string `yaml:"protocol"`
	Port       int    `yaml:"port"`
	TargetPort int    `yaml:"targetPort"`
}

func NewServicePort(port, target int) *ServicePort {
	return &ServicePort{
		Protocol:   "TCP",
		Port:       port,
		TargetPort: port,
	}
}

type ServiceSpec struct {
	Selector map[string]string
	Ports    []*ServicePort
	Type     string `yaml:"type,omitempty"`
}

func NewServiceSpec() *ServiceSpec {
	return &ServiceSpec{
		Selector: make(map[string]string),
		Ports:    make([]*ServicePort, 0),
	}
}
