package helm

import "strings"

var NOTES = `
Congratulations,

Your application is now deployed. This may take a while to be up and responding.

__list__
`

// GenerateNotesFile generates the notes file for the helm chart.
func GenerateNotesFile(ingressess map[string]*Ingress) string {

	list := make([]string, 0)

	for name, ing := range ingressess {
		for _, r := range ing.Spec.Rules {
			list = append(list, "{{ if .Values."+name+".ingress.enabled -}}\n- "+name+" is accessible on : http://"+r.Host+"\n{{- end }}")
		}
	}

	return strings.ReplaceAll(NOTES, "__list__", strings.Join(list, "\n"))
}
