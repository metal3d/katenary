package writers

import (
	"bytes"
	"katenary/helm"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const classAndVersionCondition = `{{- if and .Values.__name__.ingress.class (semverCompare ">=1.18-0" .Capabilities.KubeVersion.GitVersion) }}` + "\n"
const versionCondition = `{{- if semverCompare ">=1.18-0" .Capabilities.KubeVersion.GitVersion }}` + "\n"

func BuildIngress(ingress *helm.Ingress, name, templatesDir string) {
	// Set the backend for 1.18
	for _, b := range ingress.Spec.Rules {
		for _, p := range b.Http.Paths {
			p.Backend.ServiceName = p.Backend.Service.Name
			if n, ok := p.Backend.Service.Port["number"]; ok {
				p.Backend.ServicePort = n
			}
		}
	}
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

	backendHit := false
	for _, l := range lines {
		if strings.Contains(l, "ingressClassName") {
			// should be set only if the version of Kubernetes is 1.18-0 or higher
			cond := strings.ReplaceAll(classAndVersionCondition, "__name__", name)
			l = `  ` + cond + l + "\n" + `  {{- end }}`
		}

		// manage the backend format following the Kubernetes 1.18-0 version or higher
		if strings.Contains(l, "service:") {
			n := CountSpaces(l)
			l = strings.Repeat(" ", n) + versionCondition + l
		}
		if strings.Contains(l, "serviceName:") || strings.Contains(l, "servicePort:") {
			n := CountSpaces(l)
			if !backendHit {
				l = strings.Repeat(" ", n) + "{{- else }}\n" + l
			} else {
				l = l + "\n" + strings.Repeat(" ", n) + "{{- end }}\n"
			}
			backendHit = true
		}

		fp.WriteString(l + "\n")
	}

	fp.Close()
}
