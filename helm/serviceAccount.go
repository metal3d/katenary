package helm

// ServiceAccount defines a service account
type ServiceAccount struct {
	*K8sBase `yaml:",inline"`
}

// NewServiceAccount creates a new service account with a given name.
func NewServiceAccount(name string) *ServiceAccount {
	sa := &ServiceAccount{
		K8sBase: NewBase(),
	}
	sa.K8sBase.Kind = "ServiceAccount"
	sa.K8sBase.ApiVersion = "v1"
	sa.K8sBase.Metadata.Name = ReleaseNameTpl + "-" + name + "-cron-user"
	sa.K8sBase.Metadata.Labels[K+"/component"] = name
	return sa
}
