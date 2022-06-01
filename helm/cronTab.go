package helm

type CronTab struct {
	*K8sBase `yaml:",inline"`
	Spec     CronSpec `yaml:"spec"`
}
type CronSpec struct {
	Schedule    string      `yaml:"schedule"`
	JobTemplate JobTemplate `yaml:"jobTemplate"`
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

	//cmd, err := shlex.Split(command)
	//if err != nil {
	//	panic(err)
	//}

	cron.K8sBase.Metadata.Name = ReleaseNameTpl + "-" + name
	cron.K8sBase.Metadata.Labels[K+"/component"] = name
	cron.Spec.Schedule = schedule
	cron.Spec.JobTemplate.Spec.Template.Metadata = Metadata{
		Labels: cron.K8sBase.Metadata.Labels,
	}
	cron.Spec.JobTemplate.Spec.Template.Spec = Job{
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
	}

	return cron
}
