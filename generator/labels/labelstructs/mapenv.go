package labelstructs

import "gopkg.in/yaml.v3"

type MapEnv map[string]string

// MapEnvFrom returns a MapEnv from the given string.
func MapEnvFrom(data string) (MapEnv, error) {
	var mapping MapEnv
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}
