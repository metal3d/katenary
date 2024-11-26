package labelStructs

import "gopkg.in/yaml.v3"

type ValueFrom map[string]string

func GetValueFrom(data string) (*ValueFrom, error) {
	vf := ValueFrom{}
	if err := yaml.Unmarshal([]byte(data), &vf); err != nil {
		return nil, err
	}
	return &vf, nil
}
