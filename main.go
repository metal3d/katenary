package main

import (
	"bytes"
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
var AppVersion = "0.0.1"

func main() {

	flag.StringVar(&ComposeFile, "compose", ComposeFile, "set the compose file to parse")
	flag.StringVar(&AppName, "appname", AppName, "Give the helm chart app name")
	flag.StringVar(&AppVersion, "appversion", AppVersion, "Set the chart appVersion")
	flag.Parse()

	p := compose.NewParser(ComposeFile)
	p.Parse(AppName)
	wait := sync.WaitGroup{}

	files := make(map[string][]interface{})

	for name, s := range p.Data.Services {
		wait.Add(1)
		// it's mandatory to make the build in goroutines because some dependencies can
		// wait for a port number. So the entire services are built in parallel.
		go func(name string, s compose.Service) {
			o := generator.CreateReplicaObject(name, s)
			files[name] = o
			wait.Done()
		}(name, s)
	}
	wait.Wait()

	dirname := filepath.Join("chart", AppName)
	os.RemoveAll(dirname)
	templatesDir := filepath.Join(dirname, "templates")
	os.MkdirAll(templatesDir, 0755)

	for n, f := range files {
		for _, c := range f {
			kind := c.(helm.Kinded).Get()
			kind = strings.ToLower(kind)
			fname := filepath.Join(templatesDir, n+"."+kind+".yaml")
			fp, _ := os.Create(fname)
			switch c := c.(type) {
			case *helm.Storage:
				volname := c.K8sBase.Metadata.Labels[helm.K+"/pvc-name"]
				fp.WriteString("{{ if .Values." + n + ".persistence." + volname + ".enabled }}\n")
				enc := yaml.NewEncoder(fp)
				enc.SetIndent(2)
				enc.Encode(c)
				fp.WriteString("{{- end -}}")
			case *helm.Deployment:
				var buff []byte
				buffer := bytes.NewBuffer(buff)
				enc := yaml.NewEncoder(buffer)
				enc.SetIndent(2)
				enc.Encode(c)
				_content := string(buffer.Bytes())
				content := strings.Split(string(_content), "\n")
				dataname := ""
				component := c.Spec.Selector["matchLabels"].(map[string]string)[helm.K+"/component"]
				for _, line := range content {
					if strings.Contains(line, "name:") {
						dataname = strings.Split(line, ":")[1]
						dataname = strings.TrimSpace(dataname)
					} else if strings.Contains(line, "persistentVolumeClaim") {
						line = "          {{- if  .Values." + component + ".persistence." + dataname + ".enabled }}\n" + line
					} else if strings.Contains(line, "claimName") {
						line += "\n          {{ else }}"
						line += "\n          emptyDir: {}"
						line += "\n          {{- end }}"
					}
					fp.WriteString(line + "\n")
				}
			default:
				enc := yaml.NewEncoder(fp)
				enc.SetIndent(2)
				enc.Encode(c)
			}
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
		fp.WriteString("{{- end -}}")
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
		"appVersion":  AppVersion,
	})
	fp.Close()

	fp, _ = os.Create(filepath.Join(templatesDir, "NOTES.txt"))
	fp.WriteString(helm.GenNotes(generator.Ingresses))
	fp.Close()

}
