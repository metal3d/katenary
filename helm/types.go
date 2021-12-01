package helm

import (
	"os"
	"strings"
)

const K = "katenary.io"

var Version = "1.0" // should be set from main.Version

type Kinded interface {
	Get() string
}

type Metadata struct {
	Name        string            `yaml:"name,omitempty"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

func NewMetadata() *Metadata {
	return &Metadata{
		Name:        "",
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
	}
}

type K8sBase struct {
	ApiVersion string    `yaml:"apiVersion"`
	Kind       string    `yaml:"kind"`
	Metadata   *Metadata `yaml:"metadata"`
}

func NewBase() *K8sBase {

	b := &K8sBase{
		Metadata: NewMetadata(),
	}
	b.Metadata.Labels[K+"/project"] = getProjectName()
	b.Metadata.Labels[K+"/release"] = "{{ .Release.Name }}"
	b.Metadata.Annotations[K+"/version"] = Version
	return b
}

func (k K8sBase) Get() string {
	return k.Kind
}

func getProjectName() string {
	p, _ := os.Getwd()
	path := strings.Split(p, string(os.PathSeparator))
	return path[len(path)-1]
}
