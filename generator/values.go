package generator

import (
	"strings"

	"github.com/compose-spec/compose-go/types"
)

// RepositoryValue is a docker repository image and tag that will be saved in values.yaml.
type RepositoryValue struct {
	Image string `yaml:"image"`
	Tag   string `yaml:"tag"`
}

// PersistenceValue is a persistence configuration that will be saved in values.yaml.
type PersistenceValue struct {
	StorageClass string   `yaml:"storageClass"`
	Size         string   `yaml:"size"`
	AccessMode   []string `yaml:"accessMode"`
	Enabled      bool     `yaml:"enabled"`
}

type TLS struct {
	Enabled    bool   `yaml:"enabled"`
	SecretName string `yaml:"secretName"`
}

// IngressValue is a ingress configuration that will be saved in values.yaml.
type IngressValue struct {
	Annotations map[string]string `yaml:"annotations"`
	Host        string            `yaml:"host"`
	Path        string            `yaml:"path"`
	Class       string            `yaml:"class"`
	Enabled     bool              `yaml:"enabled"`
	TLS         TLS               `yaml:"tls"`
}

// Value will be saved in values.yaml. It contains configuration for all deployment and services.
type Value struct {
	Repository      *RepositoryValue             `yaml:"repository,omitempty"`
	Persistence     map[string]*PersistenceValue `yaml:"persistence,omitempty"`
	Ingress         *IngressValue                `yaml:"ingress,omitempty"`
	Environment     map[string]any               `yaml:"environment,omitempty"`
	Replicas        *uint32                      `yaml:"replicas,omitempty"`
	CronJob         *CronJobValue                `yaml:"cronjob,omitempty"`
	NodeSelector    map[string]string            `yaml:"nodeSelector"`
	Resources       map[string]any               `yaml:"resources"`
	ImagePullPolicy string                       `yaml:"imagePullPolicy,omitempty"`
	ServiceAccount  string                       `yaml:"serviceAccount"`
}

// NewValue creates a new Value from a compose service.
// The value contains the necessary information to deploy the service (image, tag, replicas, etc.).
//
// If `main` is true, the tag will be empty because
// it will be set in the helm chart appVersion.
func NewValue(service types.ServiceConfig, main ...bool) *Value {
	replicas := uint32(1)
	v := &Value{
		Replicas: &replicas,
	}

	// find the image tag
	tag := ""

	split := strings.Split(service.Image, ":")
	if len(split) == 1 {
		v.Repository = &RepositoryValue{
			Image: service.Image,
		}
	} else {
		v.Repository = &RepositoryValue{
			Image: strings.Join(split[:len(split)-1], ":"),
		}
	}

	// for main service, the tag should the appVersion. So here we set it to empty.
	if len(main) > 0 && !main[0] {
		if len(split) > 1 {
			tag = split[len(split)-1]
		}
		v.Repository.Tag = tag
	} else {
		v.Repository.Tag = ""
	}

	return v
}

func (v *Value) AddIngress(host, path string) {
	v.Ingress = &IngressValue{
		Enabled: true,
		Host:    host,
		Path:    path,
		Class:   "-",
		TLS: TLS{
			Enabled:    true,
			SecretName: "",
		},
	}
}

// AddPersistence adds persistence configuration to the Value.
func (v *Value) AddPersistence(volumeName string) {
	volumeName = strings.ReplaceAll(volumeName, "-", "_")
	if v.Persistence == nil {
		v.Persistence = make(map[string]*PersistenceValue, 0)
	}
	v.Persistence[volumeName] = &PersistenceValue{
		Enabled:      true,
		StorageClass: "-",
		Size:         "1Gi",
		AccessMode:   []string{"ReadWriteOnce"},
	}
}

// CronJobValue is a cronjob configuration that will be saved in values.yaml.
type CronJobValue struct {
	Repository      *RepositoryValue `yaml:"repository,omitempty"`
	Environment     map[string]any   `yaml:"environment,omitempty"`
	ImagePullPolicy string           `yaml:"imagePullPolicy,omitempty"`
	Schedule        string           `yaml:"schedule"`
}
