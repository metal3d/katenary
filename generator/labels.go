package generator

import (
	"fmt"
)

// LabelType identifies the type of label to generate in objects.
// TODO: is this still needed?
type LabelType uint8

const (
	DeploymentLabel LabelType = iota
	ServiceLabel
)

func GetLabels(serviceName, appName string) map[string]string {
	labels := map[string]string{
		KATENARY_PREFIX + "component": serviceName,
	}

	key := `{{- include "%s.labels" . | nindent __indent__ }}`
	labels[`__replace_`+serviceName] = fmt.Sprintf(key, appName)

	return labels
}

func GetMatchLabels(serviceName, appName string) map[string]string {
	labels := map[string]string{
		KATENARY_PREFIX + "component": serviceName,
	}

	key := `{{- include "%s.selectorLabels" . | nindent __indent__ }}`
	labels[`__replace_`+serviceName] = fmt.Sprintf(key, appName)

	return labels
}
