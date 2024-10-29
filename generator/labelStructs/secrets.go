package labelStructs

import "gopkg.in/yaml.v3"

type Secrets []string

func SecretsFrom(data string) (Secrets, error) {
	var mapping Secrets
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}
