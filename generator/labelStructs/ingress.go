package labelStructs

import "gopkg.in/yaml.v3"

type Ingress struct {
	Port        *int32            `yaml:"port,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
	Hostname    string            `yaml:"hostname"`
	Path        string            `yaml:"path"`
	Class       string            `yaml:"class"`
	Enabled     bool              `yaml:"enabled"`
}

// IngressFrom creates a new Ingress from a compose service.
func IngressFrom(data string) (*Ingress, error) {
	mapping := Ingress{
		Hostname: "",
		Path:     "/",
		Enabled:  false,
		Class:    "-",
		Port:     nil,
	}
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}
