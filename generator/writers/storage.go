package writers

import (
	"katenary/helm"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func BuildStorage(storage *helm.Storage, name, templatesDir string) {
	kind := "pvc"
	fname := filepath.Join(templatesDir, name+"."+kind+".yaml")
	fp, _ := os.Create(fname)
	volname := storage.K8sBase.Metadata.Labels[helm.K+"/pvc-name"]
	fp.WriteString("{{ if .Values." + name + ".persistence." + volname + ".enabled }}\n")
	enc := yaml.NewEncoder(fp)
	enc.SetIndent(IndentSize)
	enc.Encode(storage)
	fp.WriteString("{{- end -}}")
}
