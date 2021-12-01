package helm

import (
	"errors"
	"io/ioutil"
	"log"
	"strings"
)

type ConfigMap struct {
	*K8sBase `yaml:",inline"`
	Data     map[string]string `yaml:"data"`
}

func NewConfigMap(name string) *ConfigMap {
	base := NewBase()
	base.Kind = "ConfigMap"
	base.Metadata.Name = "{{ .Release.Name }}-" + name
	return &ConfigMap{
		K8sBase: base,
		Data:    make(map[string]string),
	}
}

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
		log.Printf("%d %v\n", len(parts), parts)
		if len(parts) < 2 {
			return errors.New("The environment file " + file + " is not valid")
		}
		c.Data[parts[0]] = parts[1]
	}

	return nil

}
