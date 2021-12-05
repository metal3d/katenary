package helm

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const K = "katenary.io"
const RELEASE_NAME = "{{ .Release.Name }}"
const (
	LABEL_ENV_SECRET  = K + "/secret-envfiles"
	LABEL_PORT        = K + "/ports"
	LABEL_INGRESS     = K + "/ingress"
	LABEL_ENV_SERVICE = K + "/env-to-service"
	LABEL_VOL_CM      = K + "/configmap-volumes"
)

var (
	Appname = ""
	Version = "1.0" // should be set from main.Version
)

type Kinded interface {
	Get() string
}

type Signable interface {
	BuildSHA(filename string)
}

type Named interface {
	Name() string
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
	// add some information of the build
	b.Metadata.Labels[K+"/project"] = GetProjectName()
	b.Metadata.Labels[K+"/release"] = RELEASE_NAME
	b.Metadata.Annotations[K+"/version"] = Version
	return b
}

func (k *K8sBase) BuildSHA(filename string) {
	c, _ := ioutil.ReadFile(filename)
	//sum := sha256.Sum256(c)
	sum := sha1.Sum(c)
	k.Metadata.Annotations[K+"/docker-compose-sha1"] = fmt.Sprintf("%x", string(sum[:]))
}

func (k *K8sBase) Get() string {
	return k.Kind
}

func (k *K8sBase) Name() string {
	return k.Metadata.Name
}

func GetProjectName() string {
	if len(Appname) > 0 {
		return Appname
	}
	p, _ := os.Getwd()
	path := strings.Split(p, string(os.PathSeparator))
	return path[len(path)-1]
}
