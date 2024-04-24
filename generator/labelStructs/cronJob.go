package labelStructs

import "gopkg.in/yaml.v3"

type CronJob struct {
	Image    string `yaml:"image,omitempty"`
	Command  string `yaml:"command"`
	Schedule string `yaml:"schedule"`
	Rbac     bool   `yaml:"rbac"`
}

func CronJobFrom(data string) (*CronJob, error) {
	var mapping CronJob
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}
