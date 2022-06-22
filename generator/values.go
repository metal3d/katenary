package generator

import (
	"katenary/helm"
	"strings"

	"github.com/compose-spec/compose-go/types"
)

// AddValues adds values to the values.yaml map.
func AddValues(servicename string, values map[string]EnvVal) {
	locker.Lock()
	defer locker.Unlock()

	if _, ok := Values[servicename]; !ok {
		Values[servicename] = make(map[string]interface{})
	}

	for k, v := range values {
		Values[servicename][k] = v
	}
}

// AddVolumeValues add a volume to the values.yaml map for the given deployment name.
func AddVolumeValues(deployment string, volname string, values map[string]EnvVal) {
	locker.Lock()
	defer locker.Unlock()

	if _, ok := VolumeValues[deployment]; !ok {
		VolumeValues[deployment] = make(map[string]map[string]EnvVal)
	}
	VolumeValues[deployment][volname] = values
}

// setEnvToValues will set the environment variables to the values.yaml map.
func setEnvToValues(name string, s *types.ServiceConfig, c *helm.Container) {
	// crete the "environment" key

	env := make(map[string]EnvVal)
	for k, v := range s.Environment {
		env[k] = v
	}
	if len(env) == 0 {
		return
	}

	valuesEnv := make(map[string]interface{})
	for k, v := range env {
		k = strings.ReplaceAll(k, ".", "_")
		valuesEnv[k] = v
	}

	AddValues(name, map[string]EnvVal{"environment": valuesEnv})
	for k := range env {
		fixedK := strings.ReplaceAll(k, ".", "_")
		v := "{{ tpl .Values." + name + ".environment." + fixedK + " . }}"
		s.Environment[k] = &v
		touched := false
		for _, c := range c.Env {
			if c.Name == k {
				c.Value = v
				touched = true
			}
		}
		if !touched {
			c.Env = append(c.Env, &helm.Value{Name: k, Value: v})
		}
	}
}
