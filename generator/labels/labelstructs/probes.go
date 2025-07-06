package labelstructs

import (
	"encoding/json"
	"log"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

type HealthCheck struct {
	LivenessProbe  *corev1.Probe `yaml:"livenessProbe,omitempty" json:"livenessProbe,omitempty"`
	ReadinessProbe *corev1.Probe `yaml:"readinessProbe,omitempty" json:"readinessProbe,omitempty"`
}

func ProbeFrom(data string) (*HealthCheck, error) {
	mapping := HealthCheck{}
	tmp := map[string]any{}
	err := yaml.Unmarshal([]byte(data), &tmp)
	if err != nil {
		return nil, err
	}

	if livenessProbe, ok := tmp["livenessProbe"]; ok {
		livenessProbeBytes, err := json.Marshal(livenessProbe)
		if err != nil {
			log.Printf("Error marshalling livenessProbe: %v", err)
			return nil, err
		}
		livenessProbe := &corev1.Probe{}
		err = json.Unmarshal(livenessProbeBytes, livenessProbe)
		if err != nil {
			log.Printf("Error unmarshalling livenessProbe: %v", err)
			return nil, err
		}
		mapping.LivenessProbe = livenessProbe
	}

	if readinessProbe, ok := tmp["readinessProbe"]; ok {
		readinessProbeBytes, err := json.Marshal(readinessProbe)
		if err != nil {
			log.Printf("Error marshalling readinessProbe: %v", err)
			return nil, err
		}
		readinessProbe := &corev1.Probe{}
		err = json.Unmarshal(readinessProbeBytes, readinessProbe)
		if err != nil {
			log.Printf("Error unmarshalling readinessProbe: %v", err)
			return nil, err
		}
		mapping.ReadinessProbe = readinessProbe

	}

	return &mapping, err
}
