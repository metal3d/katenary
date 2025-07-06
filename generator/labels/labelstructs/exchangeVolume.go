package labelstructs

import "gopkg.in/yaml.v3"

type ExchangeVolume struct {
	Name      string `yaml:"name" json:"name"`
	MountPath string `yaml:"mountPath" json:"mountPath"`
	Type      string `yaml:"type,omitempty" json:"type,omitempty"`
	Init      string `yaml:"init,omitempty" json:"init,omitempty"`
}

func NewExchangeVolumes(data string) ([]*ExchangeVolume, error) {
	mapping := []*ExchangeVolume{}

	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}

	return mapping, nil
}
