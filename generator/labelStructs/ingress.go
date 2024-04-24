package labelStructs

import "gopkg.in/yaml.v3"

type Ingress struct {
	// Hostname is the hostname to match against the request. It can contain wildcards.
	Hostname string `yaml:"hostname"`
	// Path is the path to match against the request. It can contain wildcards.
	Path string `yaml:"path"`
	// Enabled is a flag to enable or disable the ingress.
	Enabled bool `yaml:"enabled"`
	// Class is the ingress class to use.
	Class string `yaml:"class"`
	// Port is the port to use.
	Port *int32 `yaml:"port,omitempty"`
	// Annotations is a list of key-value pairs to add to the ingress.
	Annotations map[string]string `yaml:"annotations,omitempty"`
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
