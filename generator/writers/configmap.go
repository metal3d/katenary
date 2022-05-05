package writers

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// BuildConfigMap writes the configMap.
func BuildConfigMap(c interface{}, kind, servicename, name, templatesDir string) {
	fname := filepath.Join(templatesDir, servicename+"."+name+"."+kind+".yaml")
	fp, _ := os.Create(fname)
	enc := yaml.NewEncoder(fp)
	enc.SetIndent(IndentSize)
	enc.Encode(c)
	fp.Close()
}
