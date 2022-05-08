package writers

import (
	"katenary/helm"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// BuildStorage writes the persistentVolumeClaim.
func BuildStorage(storage *helm.Storage, name, templatesDir string) {
	kind := "pvc"
	name = storage.Metadata.Labels[helm.K+"/component"]
	pvcname := storage.Metadata.Labels[helm.K+"/pvc-name"]
	fname := filepath.Join(templatesDir, name+"-"+pvcname+"."+kind+".yaml")
	fp, err := os.Create(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()
	volname := storage.K8sBase.Metadata.Labels[helm.K+"/pvc-name"]

	fp.WriteString("{{ if .Values." + name + ".persistence." + volname + ".enabled }}\n")
	enc := yaml.NewEncoder(fp)
	enc.SetIndent(IndentSize)
	if err := enc.Encode(storage); err != nil {
		log.Fatal(err)
	}
	fp.WriteString("{{- end -}}")
}
