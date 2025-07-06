package labelstructs

import "gopkg.in/yaml.v3"

type ConfigMapFile []string

func ConfigMapFileFrom(data string) (ConfigMapFile, error) {
	var mapping ConfigMapFile
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}
