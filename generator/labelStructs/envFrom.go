package labelStructs

import "gopkg.in/yaml.v3"

type EnvFrom []string

// EnvFromFrom returns a EnvFrom from the given string.
func EnvFromFrom(data string) (EnvFrom, error) {
	var mapping EnvFrom
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}
