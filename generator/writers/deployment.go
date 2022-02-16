package writers

import (
	"bytes"
	"katenary/helm"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func BuildDeployment(deployment *helm.Deployment, name, templatesDir string) {
	kind := "deployment"
	fname := filepath.Join(templatesDir, name+"."+kind+".yaml")
	fp, _ := os.Create(fname)
	buffer := bytes.NewBuffer(nil)
	enc := yaml.NewEncoder(buffer)
	enc.SetIndent(IndentSize)
	enc.Encode(deployment)
	_content := string(buffer.Bytes())
	content := strings.Split(string(_content), "\n")
	dataname := ""
	component := deployment.Spec.Selector["matchLabels"].(map[string]string)[helm.K+"/component"]
	n := 0 // will be count of lines only on "persistentVolumeClaim" line, to indent "else" and "end" at the right place
	for _, line := range content {
		if strings.Contains(line, "name:") {
			dataname = strings.Split(line, ":")[1]
			dataname = strings.TrimSpace(dataname)
		} else if strings.Contains(line, "persistentVolumeClaim") {
			n = CountSpaces(line)
			line = strings.Repeat(" ", n) + "{{- if  .Values." + component + ".persistence." + dataname + ".enabled }}\n" + line
		} else if strings.Contains(line, "claimName") {
			spaces := strings.Repeat(" ", n)
			line += "\n" + spaces + "{{ else }}"
			line += "\n" + spaces + "emptyDir: {}"
			line += "\n" + spaces + "{{- end }}"
		}
		fp.WriteString(line + "\n")
	}
	fp.Close()

}
