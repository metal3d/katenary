package helm

type RoleRef struct {
	Kind     string `yaml:"kind"`
	Name     string `yaml:"name"`
	APIGroup string `yaml:"apiGroup"`
}

type Subject struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type RoleBinding struct {
	*K8sBase `yaml:",inline"`
	RoleRef  RoleRef   `yaml:"roleRef,omitempty"`
	Subjects []Subject `yaml:"subjects,omitempty"`
}

func NewRoleBinding(name string, user *ServiceAccount, role *Role) *RoleBinding {
	rb := &RoleBinding{
		K8sBase: NewBase(),
	}

	rb.K8sBase.Kind = "RoleBinding"
	rb.K8sBase.Metadata.Name = ReleaseNameTpl + "-" + name + "-cron-allow"
	rb.K8sBase.ApiVersion = "rbac.authorization.k8s.io/v1"
	rb.K8sBase.Metadata.Labels[K+"/component"] = name

	rb.RoleRef.Kind = "Role"
	rb.RoleRef.Name = role.Metadata.Name
	rb.RoleRef.APIGroup = "rbac.authorization.k8s.io"

	rb.Subjects = []Subject{
		{
			Kind:      "ServiceAccount",
			Name:      user.Metadata.Name,
			Namespace: "{{ .Release.Namespace }}",
		},
	}

	return rb
}
