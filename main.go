package main

import (
	"flag"
	"helm-compose/compose"
	"helm-compose/generator"
	"helm-compose/helm"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var ComposeFile = "docker-compose.yaml"
var AppName = "MyApp"

func main() {

	flag.StringVar(&ComposeFile, "compose", ComposeFile, "set the compose file to parse")
	flag.StringVar(&AppName, "appname", AppName, "Give the helm chart app name")
	flag.Parse()

	p := compose.NewParser(ComposeFile)
	p.Parse(AppName)
	wait := sync.WaitGroup{}

	files := make(map[string][]interface{})

	for name, s := range p.Data.Services {
		wait.Add(1)
		go func(name string, s compose.Service) {
			o := generator.CreateReplicaObject(name, s)
			files[name] = o
			wait.Done()
		}(name, s)
	}
	wait.Wait()

	dirname := filepath.Join("chart", AppName)
	templatesDir := filepath.Join(dirname, "templates")
	os.MkdirAll(templatesDir, 0755)

	for n, f := range files {
		for _, c := range f {
			kind := c.(helm.Kinded).Get()
			kind = strings.ToLower(kind)
			fname := filepath.Join(templatesDir, n+"."+kind+".yaml")
			fp, _ := os.Create(fname)
			enc := yaml.NewEncoder(fp)
			enc.SetIndent(2)
			enc.Encode(c)
			fp.Close()
		}
	}

	for name, ing := range generator.Ingresses {

		fname := filepath.Join(templatesDir, name+".ingress.yaml")
		fp, _ := os.Create(fname)
		enc := yaml.NewEncoder(fp)
		enc.SetIndent(2)
		fp.WriteString("{{- if .Values." + name + ".ingress.enabled -}}\n")
		enc.Encode(ing)
		fp.WriteString("\n{{- end -}}")
		fp.Close()
	}

	fp, _ := os.Create(filepath.Join(dirname, "values.yaml"))
	enc := yaml.NewEncoder(fp)
	enc.SetIndent(2)
	enc.Encode(generator.Values)
	fp.Close()

	fp, _ = os.Create(filepath.Join(dirname, "Chart.yaml"))
	enc = yaml.NewEncoder(fp)
	enc.SetIndent(2)
	enc.Encode(map[string]interface{}{
		"apiVersion":  "v2",
		"name":        AppName,
		"description": "A helm chart for " + AppName,
		"type":        "application",
		"version":     "0.1.0",
		"appVersion":  "x",
	})
	fp.Close()
}
