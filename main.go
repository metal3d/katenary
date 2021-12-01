package main

import (
	"bytes"
	"flag"
	"fmt"
	"katenary/compose"
	"katenary/generator"
	"katenary/helm"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var ComposeFile = "docker-compose.yaml"
var AppName = "MyApp"
var AppVersion = "0.0.1"
var Version = "master"
var ChartsDir = "chart"

func main() {

	flag.StringVar(&ChartsDir, "chart-dir", ChartsDir, "set the chart directory")
	flag.StringVar(&ComposeFile, "compose", ComposeFile, "set the compose file to parse")
	flag.StringVar(&AppName, "appname", AppName, "sive the helm chart app name")
	flag.StringVar(&AppVersion, "appversion", AppVersion, "set the chart appVersion")
	version := flag.Bool("version", false, "Show version and exit")
	force := flag.Bool("force", false, "force the removal of the chart-dir")
	flag.Parse()

	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}

	dirname := filepath.Join(ChartsDir, AppName)

	if _, err := os.Stat(dirname); err == nil && !*force {
		response := ""
		for response != "y" && response != "n" {
			response = "n"
			fmt.Printf("The %s directory already exists, it will be \x1b[31;1mremoved\x1b[0m, do you really want to continue ? [y/N]: ", dirname)
			fmt.Scanf("%s", &response)
			response = strings.ToLower(response)
		}
		if response == "n" {
			fmt.Println("Cancelled...")
			os.Exit(0)
		}
	}

	os.RemoveAll(dirname)
	templatesDir := filepath.Join(dirname, "templates")
	os.MkdirAll(templatesDir, 0755)

	helm.Version = Version
	p := compose.NewParser(ComposeFile)
	p.Parse(AppName)
	wait := sync.WaitGroup{}

	files := make(map[string][]interface{})

	for name, s := range p.Data.Services {
		wait.Add(1)
		// it's mandatory to build in goroutines because some dependencies can
		// wait for a port number discovery.
		// So the entire services are built in parallel.
		go func(name string, s compose.Service) {
			o := generator.CreateReplicaObject(name, s)
			files[name] = o
			wait.Done()
		}(name, s)
	}
	wait.Wait()

	// to generate notes, we need to keep an Ingresses list
	ingresses := make(map[string]*helm.Ingress)

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
				buffer := bytes.NewBuffer(nil)
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
			case *helm.Ingress:
				ingresses[n] = c // keep it to generate notes
				enc := yaml.NewEncoder(fp)
				enc.SetIndent(2)
				fp.WriteString("{{- if .Values." + n + ".ingress.enabled -}}\n")
				enc.Encode(c)
				fp.WriteString("{{- end -}}")

			default:
				enc := yaml.NewEncoder(fp)
				enc.SetIndent(2)
				enc.Encode(c)
			}
			fp.Close()
		}
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
	fp.WriteString(helm.GenNotes(ingresses))
	fp.Close()
}
