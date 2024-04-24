package labelstructs

type CronJob struct {
	Image    string `yaml:"image,omitempty"`
	Command  string `yaml:"command"`
	Schedule string `yaml:"schedule"`
	Rbac     bool   `yaml:"rbac"`
}
