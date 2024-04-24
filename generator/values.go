package generator

import (
	"strings"

	"github.com/compose-spec/compose-go/types"
)

// Values is a map of all values for all services. Written to values.yaml.
// var Values = map[string]any{}

// RepositoryValue is a docker repository image and tag that will be saved in values.yaml.
type RepositoryValue struct {
	Image string `yaml:"image"`
	Tag   string `yaml:"tag"`
}

// PersistenceValue is a persistence configuration that will be saved in values.yaml.
type PersistenceValue struct {
	Enabled      bool     `yaml:"enabled"`
	StorageClass string   `yaml:"storageClass"`
	Size         string   `yaml:"size"`
	AccessMode   []string `yaml:"accessMode"`
}

// IngressValue is a ingress configuration that will be saved in values.yaml.
type IngressValue struct {
	Enabled     bool              `yaml:"enabled"`
	Host        string            `yaml:"host"`
	Path        string            `yaml:"path"`
	Class       string            `yaml:"class"`
	Annotations map[string]string `yaml:"annotations"`
}

// Value will be saved in values.yaml. It contains configuraiton for all deployment and services.
type Value struct {
	Repository      *RepositoryValue             `yaml:"repository,omitempty"`
	Persistence     map[string]*PersistenceValue `yaml:"persistence,omitempty"`
	Ingress         *IngressValue                `yaml:"ingress,omitempty"`
	ImagePullPolicy string                       `yaml:"imagePullPolicy,omitempty"`
	Environment     map[string]any               `yaml:"environment,omitempty"`
	Replicas        *uint32                      `yaml:"replicas,omitempty"`
	CronJob         *CronJobValue                `yaml:"cronjob,omitempty"`
	NodeSelector    map[string]string            `yaml:"nodeSelector"`
	ServiceAccount  string                       `yaml:"serviceAccount"`
	Resources       map[string]any               `yaml:"resources"`
}

// CronJobValue is a cronjob configuration that will be saved in values.yaml.
type CronJobValue struct {
	Repository      *RepositoryValue `yaml:"repository,omitempty"`
	Environment     map[string]any   `yaml:"environment,omitempty"`
	ImagePullPolicy string           `yaml:"imagePullPolicy,omitempty"`
	Schedule        string           `yaml:"schedule"`
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
	v.Repository = &RepositoryValue{
		Image: split[0],
	}

	// for main service, the tag should the appVersion. So here we set it to empty.
	if len(main) > 0 && !main[0] {
		if len(split) > 1 {
			tag = split[1]
		}
		v.Repository.Tag = tag
	} else {
		v.Repository.Tag = ""
	}

	return v
}

// AddPersistence adds persistence configuration to the Value.
func (v *Value) AddPersistence(volumeName string) {
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

func (v *Value) AddIngress(host, path string) {
	v.Ingress = &IngressValue{
		Enabled: true,
		Host:    host,
		Path:    path,
		Class:   "-",
	}
}
