package helm

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

const K = "katenary.io"
const RELEASE_NAME = "{{ .Release.Name }}"
const (
	LABEL_ENV_SECRET  = K + "/secret-envfiles"
	LABEL_PORT        = K + "/ports"
	LABEL_INGRESS     = K + "/ingress"
	LABEL_ENV_SERVICE = K + "/env-to-service"
	LABEL_VOL_CM      = K + "/configmap-volumes"
	LABEL_HEALTHCHECK = K + "/healthcheck"
	LABEL_SAMEPOD     = K + "/same-pod"
	LABEL_EMPTYDIRS   = K + "/empty-dirs"
)

func GetLabelsDocumentation() string {
	t, _ := template.New("labels").Parse(`
# Labels
{{.LABEL_ENV_SECRET  | printf "%-33s"}}: set the given file names as a secret instead of configmap
{{.LABEL_PORT        | printf "%-33s"}}: set the ports to expose as a service (coma separated)
{{.LABEL_INGRESS     | printf "%-33s"}}: set the port to expose in an ingress (coma separated)
{{.LABEL_ENV_SERVICE | printf "%-33s"}}: specifies that the environment variable points on a service name (coma separated)
{{.LABEL_VOL_CM      | printf "%-33s"}}: specifies that the volumes points on a configmap (coma separated)
{{.LABEL_SAMEPOD     | printf "%-33s"}}: specifies that the pod should be deployed in the same pod than the given service name
{{.LABEL_EMPTYDIRS   | printf "%-33s"}}: specifies that the given volume names should be "emptyDir" instead of persistentVolumeClaim (coma separated)
{{.LABEL_HEALTHCHECK | printf "%-33s"}}: specifies that the container should be monitored by a healthcheck, **it overrides the docker-compose healthcheck**. 
{{ printf "%-34s" ""}} You can use these form of label values:
{{ printf "%-35s" ""}}- "http://[not used address][:port][/path]" to specify an http healthcheck
{{ printf "%-35s" ""}}- "tcp://[not used address]:port" to specify a tcp healthcheck
{{ printf "%-35s" ""}}- other string is condidered as a "command" healthcheck
    `)
	buff := bytes.NewBuffer(nil)
	t.Execute(buff, map[string]string{
		"LABEL_ENV_SECRET":  LABEL_ENV_SECRET,
		"LABEL_ENV_SERVICE": LABEL_ENV_SERVICE,
		"LABEL_PORT":        LABEL_PORT,
		"LABEL_INGRESS":     LABEL_INGRESS,
		"LABEL_VOL_CM":      LABEL_VOL_CM,
		"LABEL_HEALTHCHECK": LABEL_HEALTHCHECK,
		"LABEL_SAMEPOD":     LABEL_SAMEPOD,
		"LABEL_EMPTYDIRS":   LABEL_EMPTYDIRS,
	})
	return buff.String()
}

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
