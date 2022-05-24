package helm

type Job struct {
	ServiceAccount     string      `yaml:"serviceAccount,omitempty"`
	ServiceAccountName string      `yaml:"serviceAccountName,omitempty"`
	Containers         []Container `yaml:"containers"`
	RestartPolicy      string      `yaml:"restartPolicy,omitempty"`
}
type JobSpec struct {
	Template Job `yaml:"template"`
}

type JobTemplate struct {
	Metadata Metadata `yaml:"metadata"`
	Spec     JobSpec  `yaml:"spec"`
	Schedule string   `yaml:"schedule"`
}

type CronTab struct {
	*K8sBase    `yaml:",inline"`
	JobTemplate JobTemplate `yaml:"jobTemplate"`
}

func NewCrontab(name, image, command, schedule string, serviceAccount *ServiceAccount) *CronTab {
	cron := &CronTab{
		K8sBase: NewBase(),
	}
	cron.K8sBase.ApiVersion = "batch/v1"
	cron.K8sBase.Kind = "CronJob"

	//cmd, err := shlex.Split(command)
	//if err != nil {
	//	panic(err)
	//}

	cron.K8sBase.Metadata.Name = ReleaseNameTpl + "-" + name
	cron.K8sBase.Metadata.Labels[K+"/component"] = name
	cron.JobTemplate = JobTemplate{
		Schedule: schedule,
		Metadata: Metadata{
			Labels: cron.K8sBase.Metadata.Labels,
		},
		Spec: JobSpec{
			Template: Job{
				ServiceAccount:     serviceAccount.Name(),
				ServiceAccountName: serviceAccount.Name(),
				Containers: []Container{
					{
						Name:    name,
						Image:   image,
						Command: []string{command},
					},
				},
				RestartPolicy: "OnFailure",
			},
		},
	}

	return cron
}
