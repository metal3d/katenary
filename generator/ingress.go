package generator

import (
	"katenary/generator/labelStructs"
	"katenary/utils"
	"log"
	"strings"

	"github.com/compose-spec/compose-go/types"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	mapping, err := labelStructs.IngressFrom(label)
	if err != nil {
		log.Fatalf("Failed to parse ingress label: %s\n", err)
	}
	if mapping.Hostname == "" {
		mapping.Hostname = service.Name + ".tld"
	}

	// create the ingress
	pathType := networkv1.PathTypeImplementationSpecific
	serviceName := `{{ include "` + appName + `.fullname" . }}-` + service.Name

	// Add the ingress host to the values.yaml
	if Chart.Values[service.Name] == nil {
		Chart.Values[service.Name] = &Value{}
	}

	Chart.Values[service.Name].(*Value).Ingress = &IngressValue{
		Enabled:     mapping.Enabled,
		Path:        mapping.Path,
		Host:        mapping.Hostname,
		Class:       mapping.Class,
		Annotations: mapping.Annotations,
		TLS:         TLS{Enabled: mapping.TLS.Enabled},
	}

	// ingressClassName := `{{ .Values.` + service.Name + `.ingress.class }}`
	ingressClassName := utils.TplValue(service.Name, "ingress.class")

	servicePortName := utils.GetServiceNameByPort(int(*mapping.Port))
	ingressService := &networkv1.IngressServiceBackend{
		Name: serviceName,
		Port: networkv1.ServiceBackendPort{},
	}
	if servicePortName != "" {
		ingressService.Port.Name = servicePortName
	} else {
		ingressService.Port.Number = *mapping.Port
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

func (ingress *Ingress) Filename() string {
	return ingress.service.Name + ".ingress.yaml"
}

func (ingress *Ingress) Yaml() ([]byte, error) {
	var ret []byte
	var err error
	if ret, err = ToK8SYaml(ingress); err != nil {
		return nil, err
	}

	serviceName := ingress.service.Name
	if err != nil {
		return nil, err
	}
	ret = UnWrapTPL(ret)

	lines := strings.Split(string(ret), "\n")

	// first pass, wrap the tls part with `{{- if .Values.serviceName.ingress.tlsEnabled -}}`
	// and `{{- end -}}`

	from := -1
	to := -1
	spaces := -1
	for i, line := range lines {
		if strings.Contains(line, "tls:") {
			from = i
			spaces = utils.CountStartingSpaces(line)
			continue
		}
		if from > -1 {
			if utils.CountStartingSpaces(line) >= spaces {
				to = i
				continue
			}
		}
	}
	if from > -1 && to > -1 {
		lines[from] = strings.Repeat(" ", spaces) +
			`{{- if .Values.` + serviceName + `.ingress.tls.enabled }}` +
			"\n" +
			lines[from]
		lines[to] = strings.Repeat(" ", spaces) + `{{ end -}}`
	}

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
