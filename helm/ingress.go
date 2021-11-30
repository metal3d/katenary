package helm

type Ingress struct {
	*K8sBase `yaml:",inline"`
	Spec     IngressSpec
}

func NewIngress(name string) *Ingress {
	i := &Ingress{}
	i.K8sBase = NewBase()
	i.K8sBase.Metadata.Name = "{{ .Release.Name }}-" + name
	i.K8sBase.Kind = "Ingress"
	i.ApiVersion = "networking.k8s.io/v1"

	return i
}

func (i *Ingress) SetIngressClass(name string) {
	class := "{{ .Values." + name + ".ingress.class }}"
	i.Metadata.Annotations["kuberntes.io/ingress.class"] = class
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
	Backend  IngressBackend
}

type IngressBackend struct {
	Service IngressService
}

type IngressService struct {
	Name string
	Port map[string]interface{}
}
