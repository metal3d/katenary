package generator

import (
	"katenary/compose"
	"katenary/generator/writers"
	"katenary/helm"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

type HelmFile interface{}
type HelmFileGenerator chan HelmFile

var PrefixRE = regexp.MustCompile(`\{\{.*\}\}-?`)

func portExists(port int, ports []types.ServicePortConfig) bool {
	for _, p := range ports {
		if p.Target == uint32(port) {
			log.Println("portExists:", port, p.Target)
			return true
		}
	}
	return false
}

func Generate(p *compose.Parser, katernayVersion, appName, appVersion, chartVersion, composeFile, dirName string) {

	// make the appname global (yes... ugly but easy)
	helm.Appname = appName
	helm.Version = katernayVersion
	templatesDir := filepath.Join(dirName, "templates")

	// try to create the directory
	err := os.MkdirAll(templatesDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	files := make(map[string]HelmFileGenerator)

	// Manage services, avoid linked pods and store all services port in servicesMap
	avoids := make(map[string]bool)

	// Manage services to not export
	skips := make(map[string]bool)

	for _, s := range p.Data.Services {
		if s.Labels[helm.LABEL_IGNORE] == "true" {
			skips[s.Name] = true
		}
	}

	// remove skipped services
	for s := range skips {
		for i, service := range p.Data.Services {
			if service.Name == s {
				p.Data.Services = append(p.Data.Services[:i], p.Data.Services[i+1:]...)
				break
			}
		}
	}

	for i, service := range p.Data.Services {
		n := service.Name

		// if the service depends on a skipped service, remove the link
		for dep := range service.DependsOn {
			if skips[dep] {
				delete(service.DependsOn, dep)
			}
		}

		// if the service port is declared in labels, add it to the service.
		if ports, ok := service.Labels[helm.LABEL_PORT]; ok {
			if service.Ports == nil {
				service.Ports = make([]types.ServicePortConfig, 0)
			}
			for _, port := range strings.Split(ports, ",") {
				target, err := strconv.Atoi(port)
				if err != nil {
					log.Fatal(err)
				}
				if portExists(target, service.Ports) {
					continue
				}
				service.Ports = append(service.Ports, types.ServicePortConfig{
					Target: uint32(target),
				})
			}
		}
		// find port and store it in servicesMap
		for _, port := range service.Ports {
			target := int(port.Target)
			if target != 0 {
				servicesMap[n] = target
				break
			}
		}

		// avoid linked pods
		if _, ok := service.Labels[helm.LABEL_SAMEPOD]; ok {
			avoids[n] = true
		}

		// manage emptyDir volumes
		if empty, ok := service.Labels[helm.LABEL_EMPTYDIRS]; ok {
			//split empty list by coma
			emptyDirs := strings.Split(empty, ",")
			//append them in EmptyDirs
			EmptyDirs = append(EmptyDirs, emptyDirs...)
		}
		p.Data.Services[i] = service

	}

	// for all services in linked map, and not in avoids map, generate the service
	for _, s := range p.Data.Services {
		name := s.Name

		if _, found := avoids[name]; found {
			continue
		}
		linked := make(map[string]types.ServiceConfig, 0)
		// find service
		for _, service := range p.Data.Services {
			n := service.Name
			if linkname, ok := service.Labels[helm.LABEL_SAMEPOD]; ok && linkname == name {
				linked[n] = service
			}
		}

		files[name] = CreateReplicaObject(name, s, linked)
	}

	// to generate notes, we need to keep an Ingresses list
	ingresses := make(map[string]*helm.Ingress)

	for n, generator := range files {
		for helmFile := range generator {
			if helmFile == nil {
				break
			}
			kind := helmFile.(helm.Kinded).Get()
			kind = strings.ToLower(kind)

			// Add a SHA inside the generated file, it's only
			// to make it easy to check it the compose file corresponds to the
			// generated helm chart
			helmFile.(helm.Signable).BuildSHA(composeFile)

			// Some types need special fixes in yaml generation
			switch c := helmFile.(type) {
			case *helm.Storage:
				// For storage, we need to add a "condition" to activate it
				writers.BuildStorage(c, n, templatesDir)

			case *helm.Deployment:
				// for the deployment, we need to fix persitence volumes
				// to be activated only when the storage is "enabled",
				// either we use an "emptyDir"
				writers.BuildDeployment(c, n, templatesDir)

			case *helm.Service:
				// Change the type for service if it's an "exposed" port
				writers.BuildService(c, n, templatesDir)

			case *helm.Ingress:
				// we need to make ingresses "activable" from values
				ingresses[n] = c // keep it to generate notes
				writers.BuildIngress(c, n, templatesDir)

			case *helm.ConfigMap, *helm.Secret:
				// there could be several files, so let's force the filename
				name := c.(helm.Named).Name()
				name = PrefixRE.ReplaceAllString(name, "")
				writers.BuildConfigMap(c, kind, n, name, templatesDir)

			default:
				fname := filepath.Join(templatesDir, n+"."+kind+".yaml")
				fp, err := os.Create(fname)
				if err != nil {
					log.Fatal(err)
				}
				defer fp.Close()
				enc := yaml.NewEncoder(fp)
				enc.SetIndent(writers.IndentSize)
				enc.Encode(c)
			}
		}
	}
	// Create the values.yaml file
	valueFile, err := os.Create(filepath.Join(dirName, "values.yaml"))
	if err != nil {
		log.Fatal(err)
	}
	defer valueFile.Close()
	enc := yaml.NewEncoder(valueFile)
	enc.SetIndent(writers.IndentSize)
	enc.Encode(Values)

	// Create tht Chart.yaml file
	chartFile, err := os.Create(filepath.Join(dirName, "Chart.yaml"))
	if err != nil {
		log.Fatal(err)
	}
	defer chartFile.Close()
	chartFile.WriteString(`# Create on ` + time.Now().Format(time.RFC3339) + "\n")
	chartFile.WriteString(`# Katenary command line: ` + strings.Join(os.Args, " ") + "\n")
	enc = yaml.NewEncoder(chartFile)
	enc.SetIndent(writers.IndentSize)
	enc.Encode(map[string]interface{}{
		"apiVersion":  "v2",
		"name":        appName,
		"description": "A helm chart for " + appName,
		"type":        "application",
		"version":     chartVersion,
		"appVersion":  appVersion,
	})

	// And finally, create a NOTE.txt file
	noteFile, err := os.Create(filepath.Join(templatesDir, "NOTES.txt"))
	if err != nil {
		log.Fatal(err)
	}
	defer noteFile.Close()
	noteFile.WriteString(helm.GenerateNotesFile(ingresses))
}
