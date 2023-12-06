package generator

import (
	"katenary/utils"
	"regexp"
	"strings"

	"github.com/compose-spec/compose-go/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

var _ Yaml = (*Service)(nil)

// Service is a kubernetes Service.
type Service struct {
	*v1.Service `yaml:",inline"`
	service     *types.ServiceConfig `yaml:"-"`
}

// NewService creates a new Service from a compose service.
func NewService(service types.ServiceConfig, appName string) *Service {

	ports := []v1.ServicePort{}

	s := &Service{
		service: &service,
		Service: &v1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        utils.TplName(service.Name, appName),
				Labels:      GetLabels(service.Name, appName),
				Annotations: Annotations,
			},
			Spec: v1.ServiceSpec{
				Selector: GetMatchLabels(service.Name, appName),
				Ports:    ports,
			},
		},
	}
	for _, port := range service.Ports {
		s.AddPort(port)
	}

	return s
}

// AddPort adds a port to the service.
func (s *Service) AddPort(port types.ServicePortConfig, serviceName ...string) {
	name := s.service.Name
	if len(serviceName) > 0 {
		name = serviceName[0]
	}

	var finalport intstr.IntOrString

	if targetPort := utils.GetServiceNameByPort(int(port.Target)); targetPort == "" {
		finalport = intstr.FromInt(int(port.Target))
	} else {
		finalport = intstr.FromString(targetPort)
		name = targetPort
	}

	s.Spec.Ports = append(s.Spec.Ports, v1.ServicePort{
		Protocol:   v1.ProtocolTCP,
		Port:       int32(port.Target),
		TargetPort: finalport,
		Name:       name,
	})
}

// Yaml returns the yaml representation of the service.
func (s *Service) Yaml() ([]byte, error) {
	y, err := yaml.Marshal(s)
	lines := []string{}
	for _, line := range strings.Split(string(y), "\n") {
		if regexp.MustCompile(`^\s*loadBalancer:\s*`).MatchString(line) {
			continue
		}
		lines = append(lines, line)
	}
	y = []byte(strings.Join(lines, "\n"))

	return y, err
}

// Filename returns the filename of the service.
func (s *Service) Filename() string {
	return s.service.Name + ".service.yaml"
}
