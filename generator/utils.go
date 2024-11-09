package generator

import (
	"katenary/generator/labelStructs"
	"katenary/utils"
	"regexp"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/types"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var regexpLineWrap = regexp.MustCompile(`\n\s+}}`)

// findDeployment finds the corresponding target deployment for a service.
func findDeployment(serviceName string, deployments map[string]*Deployment) *Deployment {
	for _, d := range deployments {
		if d.service.Name == serviceName {
			return d
		}
	}
	return nil
}

// addConfigMapToService adds the configmap to the service.
func addConfigMapToService(serviceName, fromservice, chartName string, target *Deployment) {
	for i, c := range target.Spec.Template.Spec.Containers {
		if c.Name != serviceName {
			continue
		}
		c.EnvFrom = append(c.EnvFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: utils.TplName(fromservice, chartName),
				},
			},
		})
		target.Spec.Template.Spec.Containers[i] = c
	}
}

// fixPorts checks the "ports" label from container and add it to the service.
func fixPorts(service *types.ServiceConfig) error {
	// check the "ports" label from container and add it to the service
	portsLabel := ""
	ok := false
	if portsLabel, ok = service.Labels[LabelPorts]; !ok {
		return nil
	}
	ports, err := labelStructs.PortsFrom(portsLabel)
	if err != nil {
		// maybe it's a string, comma separated
		parts := strings.Split(portsLabel, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			port, err := strconv.ParseUint(part, 10, 32)
			if err != nil {
				return err
			}
			ports = append(ports, uint32(port))
		}
	}
	for _, port := range ports {
		service.Ports = append(service.Ports, types.ServicePortConfig{
			Target: port,
		})
	}
	return nil
}

// isIgnored returns true if the service is ignored.
func isIgnored(service types.ServiceConfig) bool {
	if v, ok := service.Labels[LabelIgnore]; ok {
		return v == "true" || v == "yes" || v == "1"
	}
	return false
}

// UnWrapTPL removes the line wrapping from a template.
func UnWrapTPL(in []byte) []byte {
	return regexpLineWrap.ReplaceAll(in, []byte(" }}"))
}

func ToK8SYaml(obj interface{}) ([]byte, error) {
	if o, err := yaml.Marshal(obj); err != nil {
		return nil, nil
	} else {
		return UnWrapTPL(o), nil
	}
}
