package writers

import (
	"bytes"
	"katenary/helm"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func BuildIngress(ingress *helm.Ingress, name, templatesDir string) {
	kind := "ingress"
	buffer := bytes.NewBuffer(nil)
	fname := filepath.Join(templatesDir, name+"."+kind+".yaml")
	enc := yaml.NewEncoder(buffer)
	enc.SetIndent(2)
	buffer.WriteString("{{- if .Values." + name + ".ingress.enabled -}}\n")
	enc.Encode(ingress)
	buffer.WriteString("{{- end -}}")

	fp, _ := os.Create(fname)
	content := string(buffer.Bytes())
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		if strings.Contains(l, "ingressClassName") {
			p := strings.Split(l, ":")
			condition := p[1]
			condition = strings.ReplaceAll(condition, "'", "")
			condition = strings.ReplaceAll(condition, "{{", "")
			condition = strings.ReplaceAll(condition, "}}", "")
			condition = strings.TrimSpace(condition)
			condition = "{{- if " + condition + " }}"
			l = "  " + condition + "\n" + l + "\n  {{- end }}"
		}
		fp.WriteString(l + "\n")
	}
	fp.Close()
}
