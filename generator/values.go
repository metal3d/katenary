package generator

import (
	"katenary/helm"
	"strings"

	"github.com/compose-spec/compose-go/types"
)

var (
	// Values is kept in memory to create a values.yaml file.
	Values = make(map[string]map[string]interface{})
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

func AddEnvironment(servicename string, key string, val EnvVal) {
	locker.Lock()
	defer locker.Unlock()

	if _, ok := Values[servicename]; !ok {
		Values[servicename] = make(map[string]interface{})
	}

	if _, ok := Values[servicename]["environment"]; !ok {
		Values[servicename]["environment"] = make(map[string]EnvVal)
	}
	Values[servicename]["environment"].(map[string]EnvVal)[key] = val

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

	for k, v := range env {
		k = strings.ReplaceAll(k, ".", "_")
		AddEnvironment(name, k, v)
	}

	//AddValues(name, map[string]EnvVal{"environment": valuesEnv})
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
