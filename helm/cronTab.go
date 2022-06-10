package helm

type CronTab struct {
	*K8sBase `yaml:",inline"`
	Spec     CronSpec `yaml:"spec"`
}
type CronSpec struct {
	Schedule                   string      `yaml:"schedule"`
	JobTemplate                JobTemplate `yaml:"jobTemplate"`
	SuccessfulJobsHistoryLimit int         `yaml:"successfulJobsHistoryLimit"`
	FailedJobsHistoryLimit     int         `yaml:"failedJobsHistoryLimit"`
	ConcurrencyPolicy          string      `yaml:"concurrencyPolicy"`
}
type JobTemplate struct {
	Spec JobSpecDescription `yaml:"spec"`
}

type JobSpecDescription struct {
	Template JobSpecTemplate `yaml:"template"`
}

type JobSpecTemplate struct {
	Metadata Metadata `yaml:"metadata"`
	Spec     Job      `yaml:"spec"`
}

type Job struct {
	ServiceAccount     string      `yaml:"serviceAccount,omitempty"`
	ServiceAccountName string      `yaml:"serviceAccountName,omitempty"`
	Containers         []Container `yaml:"containers"`
	RestartPolicy      string      `yaml:"restartPolicy,omitempty"`
}

func NewCrontab(name, image, command, schedule string, serviceAccount *ServiceAccount) *CronTab {
	cron := &CronTab{
		K8sBase: NewBase(),
	}
	cron.K8sBase.ApiVersion = "batch/v1"
	cron.K8sBase.Kind = "CronJob"

	cron.K8sBase.Metadata.Name = ReleaseNameTpl + "-" + name
	cron.K8sBase.Metadata.Labels[K+"/component"] = name
	cron.Spec.Schedule = schedule
	cron.Spec.SuccessfulJobsHistoryLimit = 3
	cron.Spec.FailedJobsHistoryLimit = 3
	cron.Spec.ConcurrencyPolicy = "Forbid"
	cron.Spec.JobTemplate.Spec.Template.Metadata = Metadata{
		Labels: cron.K8sBase.Metadata.Labels,
	}
	cron.Spec.JobTemplate.Spec.Template.Spec = Job{
		ServiceAccount:     serviceAccount.Name(),
		ServiceAccountName: serviceAccount.Name(),
		RestartPolicy:      "OnFailure",
	}
	if command != "" {
		cron.AddCommand(command, image, name)
	}

	return cron
}

// AddCommand adds a command to the cron job
func (c *CronTab) AddCommand(command, image, name string) {
	container := Container{
		Name:    name,
		Image:   image,
		Command: []string{"sh", "-c", command},
	}
	c.Spec.JobTemplate.Spec.Template.Spec.Containers = append(c.Spec.JobTemplate.Spec.Template.Spec.Containers, container)
}
