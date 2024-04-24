package labelStructs

import (
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

type Probe struct {
	LivenessProbe  *corev1.Probe `yaml:"livenessProbe,omitempty"`
	ReadinessProbe *corev1.Probe `yaml:"readinessProbe,omitempty"`
}

func ProbeFrom(data string) (*Probe, error) {
	var mapping Probe
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}
