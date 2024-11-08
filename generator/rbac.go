package generator

import (
	"katenary/utils"

	"github.com/compose-spec/compose-go/types"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var (
	_ Yaml = (*RoleBinding)(nil)
	_ Yaml = (*Role)(nil)
	_ Yaml = (*ServiceAccount)(nil)
)

// RBAC is a kubernetes RBAC containing a role, a rolebinding and an associated serviceaccount.
type RBAC struct {
	RoleBinding    *RoleBinding
	Role           *Role
	ServiceAccount *ServiceAccount
}

// NewRBAC creates a new RBAC from a compose service. The appName is the name of the application taken from the project name.
func NewRBAC(service types.ServiceConfig, appName string) *RBAC {
	role := &Role{
		Role: &rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Role",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        utils.TplName(service.Name, appName),
				Labels:      GetLabels(service.Name, appName),
				Annotations: Annotations,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"", "extensions", "apps"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
			},
		},
		service: &service,
	}

	rolebinding := &RoleBinding{
		RoleBinding: &rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        utils.TplName(service.Name, appName),
				Labels:      GetLabels(service.Name, appName),
				Annotations: Annotations,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      utils.TplName(service.Name, appName),
					Namespace: "{{ .Release.Namespace }}",
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "Role",
				Name:     utils.TplName(service.Name, appName),
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		service: &service,
	}

	serviceaccount := &ServiceAccount{
		ServiceAccount: &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        utils.TplName(service.Name, appName),
				Labels:      GetLabels(service.Name, appName),
				Annotations: Annotations,
			},
		},
		service: &service,
	}

	return &RBAC{
		RoleBinding:    rolebinding,
		Role:           role,
		ServiceAccount: serviceaccount,
	}
}

// RoleBinding is a kubernetes RoleBinding.
type RoleBinding struct {
	*rbacv1.RoleBinding
	service *types.ServiceConfig
}

func (r *RoleBinding) Filename() string {
	return r.service.Name + ".rolebinding.yaml"
}

func (r *RoleBinding) Yaml() ([]byte, error) {
	return yaml.Marshal(r)
}

// Role is a kubernetes Role.
type Role struct {
	*rbacv1.Role
	service *types.ServiceConfig
}

func (r *Role) Filename() string {
	return r.service.Name + ".role.yaml"
}

func (r *Role) Yaml() ([]byte, error) {
	if o, err := yaml.Marshal(r); err != nil {
		return nil, err
	} else {
		return UnWrapTPL(o), nil
	}
}

// ServiceAccount is a kubernetes ServiceAccount.
type ServiceAccount struct {
	*corev1.ServiceAccount
	service *types.ServiceConfig
}

func (r *ServiceAccount) Filename() string {
	return r.service.Name + ".serviceaccount.yaml"
}

func (r *ServiceAccount) Yaml() ([]byte, error) {
	return yaml.Marshal(r)
}
