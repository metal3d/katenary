package labelStructs

import "gopkg.in/yaml.v3"

type TLS struct {
	Enabled bool `yaml:"enabled" json:"enabled,omitempty"`
}

type Ingress struct {
	Port        *int32            `yaml:"port,omitempty" jsonschema:"nullable" json:"port,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" jsonschema:"nullable" json:"annotations,omitempty"`
	Hostname    string            `yaml:"hostname" json:"hostname,omitempty"`
	Path        string            `yaml:"path" json:"path,omitempty"`
	Class       string            `yaml:"class" json:"class,omitempty" jsonschema:"default:-"`
	Enabled     bool              `yaml:"enabled" json:"enabled,omitempty"`
	TLS         *TLS              `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// IngressFrom creates a new Ingress from a compose service.
func IngressFrom(data string) (*Ingress, error) {
	mapping := Ingress{
		Hostname: "",
		Path:     "/",
		Enabled:  false,
		Class:    "-",
		Port:     nil,
		TLS:      &TLS{Enabled: true},
	}
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}
