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

// GetLabels returns the labels for a service. It uses the appName to replace the __replace__ in the labels.
// This is used to generate the labels in the templates.
func GetLabels(serviceName, appName string) map[string]string {
	labels := map[string]string{
		KATENARY_PREFIX + "component": serviceName,
	}

	key := `{{- include "%s.labels" . | nindent __indent__ }}`
	labels[`__replace_`+serviceName] = fmt.Sprintf(key, appName)

	return labels
}

// GetMatchLabels returns the matchLabels for a service. It uses the appName to replace the __replace__ in the labels.
// This is used to generate the matchLabels in the templates.
func GetMatchLabels(serviceName, appName string) map[string]string {
	labels := map[string]string{
		KATENARY_PREFIX + "component": serviceName,
	}

	key := `{{- include "%s.selectorLabels" . | nindent __indent__ }}`
	labels[`__replace_`+serviceName] = fmt.Sprintf(key, appName)

	return labels
}
