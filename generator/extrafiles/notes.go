package extrafiles

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed notes.tpl
var notesTemplate string

// NotesFile returns the content of the note.txt file.
func NotesFile(services []string) string {
	// build a list of ingress URLs if there are any
	ingresses := make([]string, len(services))
	for i, service := range services {
		condition := fmt.Sprintf(`{{- if and .Values.%[1]s.ingress .Values.%[1]s.ingress.enabled }}`, service)
		line := fmt.Sprintf(`{{- $count = add1 $count -}}{{- $listOfURL = printf "%%s\n- http://%%s" $listOfURL .Values.%s.ingress.host -}}`, service)
		ingresses[i] = fmt.Sprintf("%s\n%s\n{{- end }}", condition, line)
	}

	// inject the list of ingress URLs into the notes template
	notes := strings.Split(notesTemplate, "\n")
	for i, line := range notes {
		if strings.Contains(line, "ingress_list") {
			notes[i] = strings.Join(ingresses, "\n")
		}
	}

	return strings.Join(notes, "\n")
}
