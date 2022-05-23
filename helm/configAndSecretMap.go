package helm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
)

// InlineConfig is made to represent a configMap or a secret
type InlineConfig interface {
	AddEnvFile(filename string, filter []string) error
	AddEnv(key, val string) error
	Metadata() *Metadata
}

var _ InlineConfig = (*ConfigMap)(nil)
var _ InlineConfig = (*Secret)(nil)

// ConfigMap is made to represent a configMap with data.
type ConfigMap struct {
	*K8sBase `yaml:",inline"`
	Data     map[string]string `yaml:"data"`
}

// NewConfigMap returns a new initialzed ConfigMap.
func NewConfigMap(name, path string) *ConfigMap {
	base := NewBase()
	base.ApiVersion = "v1"
	base.Kind = "ConfigMap"
	base.Metadata.Name = ReleaseNameTpl + "-" + name
	base.Metadata.Labels[K+"/component"] = name
	if path != "" {
		base.Metadata.Labels[K+"/path"] = path
	}
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
func (c *ConfigMap) AddEnvFile(file string, filter []string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		//Check if the line is a comment
		l = strings.TrimSpace(l)
		isComment := strings.HasPrefix(l, "#")
		if len(l) == 0 || isComment {
			continue
		}
		parts := strings.SplitN(l, "=", 2)
		if len(parts) < 2 {
			return errors.New("The environment file " + file + " is not valid")
		}

		var skip bool
		for _, filterEnv := range filter {
			if parts[0] == filterEnv {
				skip = true
			}
		}
		if !skip {
			c.Data[parts[0]] = parts[1]
		}
	}
	return nil
}

func (c *ConfigMap) AddEnv(key, val string) error {
	c.Data[key] = val
	return nil
}

// Secret is made to represent a secret with data.
type Secret struct {
	*K8sBase `yaml:",inline"`
	Data     map[string]string `yaml:"data"`
}

// NewSecret returns a new initialzed Secret.
func NewSecret(name, path string) *Secret {
	base := NewBase()
	base.ApiVersion = "v1"
	base.Kind = "Secret"
	base.Metadata.Name = ReleaseNameTpl + "-" + name
	base.Metadata.Labels[K+"/component"] = name
	if path != "" {
		base.Metadata.Labels[K+"/path"] = path
	}
	return &Secret{
		K8sBase: base,
		Data:    make(map[string]string),
	}
}

// AddEnvFile adds an environment file to the secret.
func (s *Secret) AddEnvFile(file string, filter []string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		isComment := strings.HasPrefix(l, "#")
		if len(l) == 0 || isComment {
			continue
		}
		parts := strings.SplitN(l, "=", 2)
		if len(parts) < 2 {
			return errors.New("The environment file " + file + " is not valid")
		}

		var skip bool
		for _, filterEnv := range filter {
			if parts[0] == filterEnv {
				skip = true
			}
		}
		if !skip {
			s.Data[parts[0]] = fmt.Sprintf(`{{ "%s" | b64enc }}`, parts[1])
		}
	}

	return nil

}

// Metadata returns the metadata of the secret.
func (s *Secret) Metadata() *Metadata {
	return s.K8sBase.Metadata
}

// AddEnv adds an environment variable to the secret.
func (s *Secret) AddEnv(key, val string) error {
	s.Data[key] = fmt.Sprintf(`{{ %s | b64enc }}`, val)
	return nil
}
