package helm

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"strings"
)

// Metadata is the metadata for a kubernetes object.
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

// K8sBase is the base for all kubernetes objects.
type K8sBase struct {
	ApiVersion string    `yaml:"apiVersion"`
	Kind       string    `yaml:"kind"`
	Metadata   *Metadata `yaml:"metadata"`
}

// NewBase is a factory for creating a new base object with metadata, labels and annotations set to the default.
func NewBase() *K8sBase {
	b := &K8sBase{
		Metadata: NewMetadata(),
	}
	// add some information of the build
	b.Metadata.Labels[K+"/project"] = "{{ .Chart.Name }}"
	b.Metadata.Labels[K+"/release"] = ReleaseNameTpl
	b.Metadata.Annotations[K+"/version"] = Version
	return b
}

func (k *K8sBase) BuildSHA(filename string) {
	c, _ := ioutil.ReadFile(filename)
	//sum := sha256.Sum256(c)
	sum := sha1.Sum(c)
	k.Metadata.Annotations[K+"/docker-compose-sha1"] = fmt.Sprintf("%x", string(sum[:]))
}

// Get returns the Kind.
func (k *K8sBase) Get() string {
	return k.Kind
}

// Name returns the name of the object from Metadata.
func (k *K8sBase) Name() string {
	return k.Metadata.Name
}

func (k *K8sBase) GetType() string {
	if n, ok := k.Metadata.Labels[K+"/type"]; ok {
		return n
	}
	return strings.ToLower(k.Kind)
}

func (k *K8sBase) GetPathRessource() string {
	if p, ok := k.Metadata.Labels[K+"/path"]; ok {
		return p
	}
	return ""
}
