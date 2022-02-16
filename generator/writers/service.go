package writers

import (
	"katenary/helm"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func BuildService(service *helm.Service, name, templatesDir string) {
	kind := "service"
	suffix := ""
	if service.Spec.Type == "NodePort" {
		suffix = "-external"
	}
	fname := filepath.Join(templatesDir, name+suffix+"."+kind+".yaml")
	fp, _ := os.Create(fname)
	enc := yaml.NewEncoder(fp)
	enc.SetIndent(IndentSize)
	enc.Encode(service)
	fp.Close()
}
