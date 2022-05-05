package helm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
)

// InlineConfig is made to represent a configMap or a secret
type InlineConfig interface {
	AddEnvFile(filename string) error
	Metadata() *Metadata
}

// ConfigMap is made to represent a configMap with data.
type ConfigMap struct {
	*K8sBase `yaml:",inline"`
	Data     map[string]string `yaml:"data"`
}

// NewConfigMap returns a new initialzed ConfigMap.
func NewConfigMap(name string) *ConfigMap {
	base := NewBase()
	base.ApiVersion = "v1"
	base.Kind = "ConfigMap"
	base.Metadata.Name = ReleaseNameTpl + "-" + name
	base.Metadata.Labels[K+"/component"] = name
	return &ConfigMap{
		K8sBase: base,
		Data:    make(map[string]string),
	}
}

// Metadata returns the metadata of the configMap.
func (c *ConfigMap) Metadata() *Metadata {
	return c.K8sBase.Metadata
}

// AddEnvFile adds an environment file to the configMap.
func (c *ConfigMap) AddEnvFile(file string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if len(l) == 0 {
			continue
		}
		parts := strings.SplitN(l, "=", 2)
		if len(parts) < 2 {
			return errors.New("The environment file " + file + " is not valid")
		}
		c.Data[parts[0]] = parts[1]
	}

	return nil

}

// Secret is made to represent a secret with data.
type Secret struct {
	*K8sBase `yaml:",inline"`
	Data     map[string]string `yaml:"data"`
}

// NewSecret returns a new initialzed Secret.
func NewSecret(name string) *Secret {
	base := NewBase()
	base.ApiVersion = "v1"
	base.Kind = "Secret"
	base.Metadata.Name = ReleaseNameTpl + "-" + name
	base.Metadata.Labels[K+"/component"] = name
	return &Secret{
		K8sBase: base,
		Data:    make(map[string]string),
	}
}

// AddEnvFile adds an environment file to the secret.
func (s *Secret) AddEnvFile(file string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if len(l) == 0 {
			continue
		}
		parts := strings.SplitN(l, "=", 2)
		if len(parts) < 2 {
			return errors.New("The environment file " + file + " is not valid")
		}
		s.Data[parts[0]] = fmt.Sprintf(`{{ "%s" | b64enc }}`, parts[1])
	}

	return nil

}

// Metadata returns the metadata of the secret.
func (s *Secret) Metadata() *Metadata {
	return s.K8sBase.Metadata
}
