package extrafiles

import (
	"strings"
	"testing"
)

// override the embedded template for testing
var testTemplate = `
Some header
{{ ingress_list }}
Some footer
`

func init() {
	notesTemplate = testTemplate
}

func TestNotesFile_NoServices(t *testing.T) {
	result := NotesFile([]string{})
	if !strings.Contains(result, "Some header") || !strings.Contains(result, "Some footer") {
		t.Errorf("Expected template header/footer in output, got: %s", result)
	}
}

func TestNotesFile_WithServices(t *testing.T) {
	services := []string{"svc1", "svc2"}
	result := NotesFile(services)

	for _, svc := range services {
		cond := "{{- if and .Values." + svc + ".ingress .Values." + svc + ".ingress.enabled }}"
		line := "{{- $count = add1 $count -}}{{- $listOfURL = printf \"%s\\n- http://%s\" $listOfURL (tpl .Values." + svc + ".ingress.host .) -}}"
		if !strings.Contains(result, cond) {
			t.Errorf("Expected condition for service %s in output", svc)
		}
		if !strings.Contains(result, line) {
			t.Errorf("Expected line for service %s in output", svc)
		}
	}
}
