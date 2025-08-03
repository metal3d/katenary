package labelstructs

import "gopkg.in/yaml.v3"

type ConfigMapFiles []string

func ConfigMapFileFrom(data string) (ConfigMapFiles, error) {
	var mapping ConfigMapFiles
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}
