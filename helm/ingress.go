package helm

// Ingress is the kubernetes ingress object.
type Ingress struct {
	*K8sBase `yaml:",inline"`
	Spec     IngressSpec
}

func NewIngress(name string) *Ingress {
	i := &Ingress{}
	i.K8sBase = NewBase()
	i.K8sBase.Metadata.Name = RELEASE_NAME + "-" + name
	i.K8sBase.Kind = "Ingress"
	i.ApiVersion = "networking.k8s.io/v1"
	i.K8sBase.Metadata.Labels[K+"/component"] = name

	return i
}

func (i *Ingress) SetIngressClass(name string) {
	class := "{{ .Values." + name + ".ingress.class }}"
	i.Spec.IngressClassName = class
}

type IngressSpec struct {
	IngressClassName string `yaml:"ingressClassName,omitempty"`
	Rules            []IngressRule
}

type IngressRule struct {
	Host string
	Http IngressHttp
}

type IngressHttp struct {
	Paths []IngressPath
}

type IngressPath struct {
	Path     string
	PathType string `yaml:"pathType"`
	Backend  *IngressBackend
}

type IngressBackend struct {
	Service     IngressService
	ServiceName string      `yaml:"serviceName"` // for kubernetes version < 1.18
	ServicePort interface{} `yaml:"servicePort"` // for kubernetes version < 1.18
}

type IngressService struct {
	Name string                 `yaml:"name"`
	Port map[string]interface{} `yaml:"port"`
}
