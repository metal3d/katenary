package writers

import (
	"bytes"
	"katenary/helm"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	classAndVersionCondition = `{{- if and .Values.__name__.ingress.class (semverCompare ">=1.18-0" .Capabilities.KubeVersion.GitVersion) }}` + "\n"
	versionCondition118      = `{{- if semverCompare ">=1.18-0" .Capabilities.KubeVersion.GitVersion }}` + "\n"
	versionCondition119      = `{{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion }}` + "\n"
	apiVersion               = `{{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1
{{- else if semverCompare ">=1.14-0" .Capabilities.KubeVersion.GitVersion -}}
apiVersion: networking.k8s.io/v1beta1
{{- else -}}
apiVersion: extensions/v1beta1
{{- end }}`
)

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
	enc.SetIndent(IndentSize)
	buffer.WriteString("{{- if .Values." + name + ".ingress.enabled -}}\n")
	enc.Encode(ingress)
	buffer.WriteString("{{- end -}}")

	fp, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	content := string(buffer.Bytes())
	lines := strings.Split(content, "\n")

	backendHit := false
	for _, l := range lines {
		// apiVersion is a pain...
		if strings.Contains(l, "apiVersion:") {
			l = apiVersion
		}

		// pathTyype is ony for 1.19+
		if strings.Contains(l, "pathType:") {
			n := CountSpaces(l)
			l = strings.Repeat(" ", n) + versionCondition118 +
				l + "\n" +
				strings.Repeat(" ", n) + "{{- end -}}"
		}

		if strings.Contains(l, "ingressClassName") {
			// should be set only if the version of Kubernetes is 1.18-0 or higher
			cond := strings.ReplaceAll(classAndVersionCondition, "__name__", name)
			l = `  ` + cond + l + "\n" + `  {{- end }}`
		}

		// manage the backend format following the Kubernetes 1.19-0 version or higher
		if strings.Contains(l, "service:") {
			n := CountSpaces(l)
			l = strings.Repeat(" ", n) + versionCondition119 + l
		}
		if strings.Contains(l, "serviceName:") || strings.Contains(l, "servicePort:") {
			n := CountSpaces(l)
			if !backendHit {
				l = strings.Repeat(" ", n) + "{{- else }}\n" + l
			} else {
				l = l + "\n" + strings.Repeat(" ", n) + "{{- end }}"
			}
			backendHit = true
		}
		fp.WriteString(l + "\n")
	}
}
