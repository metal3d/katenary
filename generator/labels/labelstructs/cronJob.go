package labelstructs

import "gopkg.in/yaml.v3"

type CronJob struct {
	Image    string `yaml:"image,omitempty" json:"image,omitempty"`
	Command  string `yaml:"command" json:"command,omitempty"`
	Schedule string `yaml:"schedule" json:"schedule,omitempty"`
	Rbac     bool   `yaml:"rbac" json:"rbac,omitempty"`
}

func CronJobFrom(data string) (*CronJob, error) {
	var mapping CronJob
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}
