package helm

import (
	"bytes"
	"html/template"
)

const ReleaseNameTpl = "{{ .Release.Name }}"
const (
	LABEL_MAP_ENV        = K + "/mapenv"
	LABEL_ENV_SECRET     = K + "/secret-envfiles"
	LABEL_PORT           = K + "/ports"
	LABEL_CONTAINER_PORT = K + "/container-ports"
	LABEL_INGRESS        = K + "/ingress"
	LABEL_VOL_CM         = K + "/configmap-volumes"
	LABEL_HEALTHCHECK    = K + "/healthcheck"
	LABEL_SAMEPOD        = K + "/same-pod"
	LABEL_VOLUMEFROM     = K + "/volume-from"
	LABEL_EMPTYDIRS      = K + "/empty-dirs"
	LABEL_IGNORE         = K + "/ignore"
	LABEL_SECRETVARS     = K + "/secret-vars"
	LABEL_CRON           = K + "/crontabs"
	LABEL_DEPENDENCIES   = K + "/dependency"

	//deprecated: use LABEL_MAP_ENV instead
	LABEL_ENV_SERVICE = K + "/env-to-service"
)

// GetLabelsDocumentation returns the documentation for the labels.
func GetLabelsDocumentation() string {
	t, err := template.New("labels").Parse(`# Labels
{{.LABEL_IGNORE         | printf "%-33s"}}: ignore the container, it will not yied any object in the helm chart (bool)
{{.LABEL_SECRETVARS     | printf "%-33s"}}: secret variables to push on a secret file (coma separated)
{{.LABEL_ENV_SECRET     | printf "%-33s"}}: set the given file names as a secret instead of configmap (coma separated)
{{.LABEL_MAP_ENV        | printf "%-33s"}}: map environment variable to a template string (yaml style, object)
{{.LABEL_PORT           | printf "%-33s"}}: set the ports to assign on the container in pod + expose as a service (coma separated)
{{.LABEL_CONTAINER_PORT | printf "%-33s"}}: set the ports to assign on the contaienr in pod but avoid service (coma separated)
{{.LABEL_INGRESS        | printf "%-33s"}}: set the port to expose in an ingress (coma separated)
{{.LABEL_VOL_CM         | printf "%-33s"}}: specifies that the volumes points on a configmap (coma separated)
{{.LABEL_SAMEPOD        | printf "%-33s"}}: specifies that the pod should be deployed in the same pod than the
{{ printf "%-34s" ""}} given service name (string)
{{.LABEL_VOLUMEFROM     | printf "%-33s"}}: specifies that the volumes to be mounted from the given service (yaml style)
{{.LABEL_EMPTYDIRS      | printf "%-33s"}}: specifies that the given volume names should be "emptyDir" instead of
{{ printf "%-34s" ""}} persistentVolumeClaim (coma separated)
{{.LABEL_DEPENDENCIES   | printf "%-33s"}}: specifies that the given service is actually a Helm Dependency (yaml style)
{{ printf "%-34s" ""}} The form is the following:
{{ printf "%-34s" ""}} - name: name of the dependency
{{ printf "%-34s" ""}}   version: version of the dependency
{{ printf "%-34s" ""}}   repository: repository of the dependency
{{ printf "%-34s" ""}}   alias: alias of the dependency (optional)
{{ printf "%-34s" ""}}   config: config of the dependency (map, optional)
{{ printf "%-34s" ""}}     environment: map for environment
{{ printf "%-34s" ""}}     serviceName: the service name as defined in the chart that replace the current service name (default to the compose service name)
{{.LABEL_CRON           | printf "%-33s"}}: specifies a cronjobs to create (yaml style, array) - this will create a
{{ printf "%-34s" ""}} cronjob, a service account, a role and a rolebinding to start the command with "kubectl"
{{ printf "%-34s" ""}} The form is the following:
{{ printf "%-34s" ""}} - command: the command to run
{{ printf "%-34s" ""}}   schedule: the schedule to run the command (e.g. "@daily" or "*/1 * * * *")
{{ printf "%-34s" ""}}   image: the image to use for the command (default to "bitnami/kubectl")
{{ printf "%-34s" ""}}   allPods: true if you want to run the command on all pods (default to false)
{{.LABEL_HEALTHCHECK | printf "%-33s"}}: specifies that the container should be monitored by a healthcheck,
{{ printf "%-34s" ""}} **it overrides the docker-compose healthcheck**. 
{{ printf "%-34s" ""}} You can use these form of label values:
{{ printf "%-35s" ""}}  -> http://[ignored][:port][/path] to specify an http healthcheck
{{ printf "%-35s" ""}}  -> tcp://[ignored]:port to specify a tcp healthcheck
{{ printf "%-35s" ""}}  -> other string is condidered as a "command" healthcheck`)
	if err != nil {
		panic(err)
	}
	buff := bytes.NewBuffer(nil)
	t.Execute(buff, map[string]string{
		"LABEL_ENV_SECRET":     LABEL_ENV_SECRET,
		"LABEL_PORT":           LABEL_PORT,
		"LABEL_CONTAINER_PORT": LABEL_CONTAINER_PORT,
		"LABEL_INGRESS":        LABEL_INGRESS,
		"LABEL_VOL_CM":         LABEL_VOL_CM,
		"LABEL_HEALTHCHECK":    LABEL_HEALTHCHECK,
		"LABEL_SAMEPOD":        LABEL_SAMEPOD,
		"LABEL_VOLUMEFROM":     LABEL_VOLUMEFROM,
		"LABEL_EMPTYDIRS":      LABEL_EMPTYDIRS,
		"LABEL_IGNORE":         LABEL_IGNORE,
		"LABEL_MAP_ENV":        LABEL_MAP_ENV,
		"LABEL_SECRETVARS":     LABEL_SECRETVARS,
		"LABEL_CRON":           LABEL_CRON,
		"LABEL_DEPENDENCIES":   LABEL_DEPENDENCIES,
	})
	return buff.String()
}
