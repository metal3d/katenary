package helm

import (
	"bytes"
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

// Kinded represent an object with a kind.
type Kinded interface {

	// Get must resturn the kind name.
	Get() string
}

// Signable represents an object with a signature.
type Signable interface {

	// BuildSHA must return the signature.
	BuildSHA(filename string)
}

// Named represents an object with a name.
type Named interface {

	// Name must return the name of the object (from metadata).
	Name() string
}

// GetProjectName returns the name of the project.
func GetProjectName() string {
	if len(Appname) > 0 {
		return Appname
	}
	p, _ := os.Getwd()
	path := strings.Split(p, string(os.PathSeparator))
	return path[len(path)-1]
}
