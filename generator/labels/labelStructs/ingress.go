package labelStructs

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type TLS struct {
	Enabled bool `yaml:"enabled" json:"enabled,omitempty"`
}

type Ingress struct {
	Port        *int32            `yaml:"port,omitempty" json:"port,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" jsonschema:"nullable" json:"annotations,omitempty"`
	Hostname    string            `yaml:"hostname" json:"hostname,omitempty"`
	Path        *string           `yaml:"path,omitempty" json:"path,omitempty"`
	Class       *string           `yaml:"class,omitempty" json:"class,omitempty" jsonschema:"default:-"`
	Enabled     bool              `yaml:"enabled" json:"enabled,omitempty"`
	TLS         *TLS              `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// IngressFrom creates a new Ingress from a compose service.
func IngressFrom(data string) (*Ingress, error) {
	strPtr := func(s string) *string {
		return &s
	}
	mapping := Ingress{
		Hostname: "",
		Path:     strPtr("/"),
		Enabled:  false,
		Class:    strPtr("-"),
		Port:     nil,
		TLS:      &TLS{Enabled: true},
	}
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	if mapping.Port == nil {
		return nil, fmt.Errorf("port is required in ingress definition")
	}
	return &mapping, nil
}
