package generator

import (
	"log"
	"strings"

	"katenary/utils"

	"github.com/compose-spec/compose-go/types"
	goyaml "gopkg.in/yaml.v3"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var _ Yaml = (*Ingress)(nil)

type Ingress struct {
	*networkv1.Ingress
	service *types.ServiceConfig `yaml:"-"`
}

// NewIngress creates a new Ingress from a compose service.
func NewIngress(service types.ServiceConfig, Chart *HelmChart) *Ingress {
	appName := Chart.Name

	if service.Labels == nil {
		service.Labels = make(map[string]string)
	}
	var label string
	var ok bool
	if label, ok = service.Labels[LabelIngress]; !ok {
		return nil
	}

	mapping := map[string]interface{}{
		"enabled": false,
		"host":    service.Name + ".tld",
		"path":    "/",
		"class":   "-",
	}
	if err := goyaml.Unmarshal([]byte(label), &mapping); err != nil {
		log.Fatalf("Failed to parse ingress label: %s\n", err)
	}

	// create the ingress
	pathType := networkv1.PathTypeImplementationSpecific
	serviceName := `{{ include "` + appName + `.fullname" . }}-` + service.Name
	if v, ok := mapping["port"]; ok {
		if port, ok := v.(int); ok {
			mapping["port"] = int32(port)
		}
	} else {
		log.Fatalf("No port provided for ingress target in service %s\n", service.Name)
	}

	// Add the ingress host to the values.yaml
	if Chart.Values[service.Name] == nil {
		Chart.Values[service.Name] = &Value{}
	}

	// fix the ingress host => hostname
	if hostname, ok := mapping["host"]; ok && hostname != "" {
		mapping["hostname"] = hostname
	}

	Chart.Values[service.Name].(*Value).Ingress = &IngressValue{
		Enabled:     mapping["enabled"].(bool),
		Path:        mapping["path"].(string),
		Host:        mapping["hostname"].(string),
		Class:       mapping["class"].(string),
		Annotations: map[string]string{},
	}

	// ingressClassName := `{{ .Values.` + service.Name + `.ingress.class }}`
	ingressClassName := utils.TplValue(service.Name, "ingress.class")

	servicePortName := utils.GetServiceNameByPort(int(mapping["port"].(int32)))
	ingressService := &networkv1.IngressServiceBackend{
		Name: serviceName,
		Port: networkv1.ServiceBackendPort{},
	}
	if servicePortName != "" {
		ingressService.Port.Name = servicePortName
	} else {
		ingressService.Port.Number = mapping["port"].(int32)
	}

	ing := &Ingress{
		service: &service,
		Ingress: &networkv1.Ingress{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Ingress",
				APIVersion: "networking.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        utils.TplName(service.Name, appName),
				Labels:      GetLabels(service.Name, appName),
				Annotations: Annotations,
			},
			Spec: networkv1.IngressSpec{
				IngressClassName: &ingressClassName,
				Rules: []networkv1.IngressRule{
					{
						Host: utils.TplValue(service.Name, "ingress.host"),
						IngressRuleValue: networkv1.IngressRuleValue{
							HTTP: &networkv1.HTTPIngressRuleValue{
								Paths: []networkv1.HTTPIngressPath{
									{
										Path:     utils.TplValue(service.Name, "ingress.path"),
										PathType: &pathType,
										Backend: networkv1.IngressBackend{
											Service: ingressService,
										},
									},
								},
							},
						},
					},
				},
				TLS: []networkv1.IngressTLS{
					{
						Hosts: []string{
							`{{ tpl .Values.` + service.Name + `.ingress.host . }}`,
						},
						SecretName: `{{ include "` + appName + `.fullname" . }}-` + service.Name + `-tls`,
					},
				},
			},
		},
	}

	return ing
}

func (ingress *Ingress) Yaml() ([]byte, error) {
	serviceName := ingress.service.Name
	ret, err := yaml.Marshal(ingress)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(ret), "\n")
	out := []string{
		`{{- if .Values.` + serviceName + `.ingress.enabled -}}`,
	}
	for _, line := range lines {
		if strings.Contains(line, "loadBalancer: ") {
			continue
		}

		if strings.Contains(line, "labels:") {
			// add annotations above labels from values.yaml
			content := `` +
				`    {{- if .Values.` + serviceName + `.ingress.annotations -}}` + "\n" +
				`        {{- toYaml .Values.` + serviceName + `.ingress.annotations | nindent 4 }}` + "\n" +
				`    {{- end }}` + "\n" +
				line

			out = append(out, content)
		} else if strings.Contains(line, "ingressClassName: ") {
			content := utils.Wrap(
				line,
				`{{- if ne .Values.`+serviceName+`.ingress.class "-" }}`,
				`{{- end }}`,
			)
			out = append(out, content)
		} else {
			out = append(out, line)
		}
	}
	out = append(out, `{{- end -}}`)
	ret = []byte(strings.Join(out, "\n"))
	return ret, nil
}

func (ingress *Ingress) Filename() string {
	return ingress.service.Name + ".ingress.yaml"
}
