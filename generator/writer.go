package generator

import (
	"katenary/compose"
	"katenary/generator/writers"
	"katenary/helm"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

var PrefixRE = regexp.MustCompile(`\{\{.*\}\}-?`)

func Generate(p *compose.Parser, katernayVersion, appName, appVersion, composeFile, dirName string) {

	// make the appname global (yes... ugly but easy)
	helm.Appname = appName
	helm.Version = katernayVersion
	templatesDir := filepath.Join(dirName, "templates")

	// try to create the directory
	err := os.MkdirAll(templatesDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	files := make(map[string]chan interface{})

	// Manage services, avoid linked pods and store all services port in servicesMap
	avoids := make(map[string]bool)
	linked := make(map[string]types.ServiceConfig, 0)
	for _, service := range p.Data.Services {
		n := service.Name

		// find port and store it in servicesMap
		for _, port := range service.Ports {
			target := int(port.Target)
			if target != 0 {
				servicesMap[n] = target
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

		// find service linked to this one
		for _, service := range p.Data.Services {
			n := service.Name
			for _, label := range service.Labels {
				if label == helm.LABEL_SAMEPOD {
					linked[n] = service
				}
			}
		}
	}

	// for all services in linked map, and not in avoids map, generate the service
	for _, s := range p.Data.Services {
		name := s.Name

		if _, found := avoids[name]; found {
			continue
		}
		files[name] = CreateReplicaObject(name, s, linked)
	}

	// to generate notes, we need to keep an Ingresses list
	ingresses := make(map[string]*helm.Ingress)

	for n, f := range files {
		for c := range f {
			if c == nil {
				break
			}
			kind := c.(helm.Kinded).Get()
			kind = strings.ToLower(kind)

			// Add a SHA inside the generated file, it's only
			// to make it easy to check it the compose file corresponds to the
			// generated helm chart
			c.(helm.Signable).BuildSHA(composeFile)

			// Some types need special fixes in yaml generation
			switch c := c.(type) {
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
				fp, _ := os.Create(fname)
				enc := yaml.NewEncoder(fp)
				enc.SetIndent(2)
				enc.Encode(c)
				fp.Close()
			}
		}
	}
	// Create the values.yaml file
	fp, _ := os.Create(filepath.Join(dirName, "values.yaml"))
	enc := yaml.NewEncoder(fp)
	enc.SetIndent(2)
	enc.Encode(Values)
	fp.Close()

	// Create tht Chart.yaml file
	fp, _ = os.Create(filepath.Join(dirName, "Chart.yaml"))
	fp.WriteString(`# Create on ` + time.Now().Format(time.RFC3339) + "\n")
	fp.WriteString(`# Katenary command line: ` + strings.Join(os.Args, " ") + "\n")
	enc = yaml.NewEncoder(fp)
	enc.SetIndent(writers.IndentSize)
	enc.Encode(map[string]interface{}{
		"apiVersion":  "v2",
		"name":        appName,
		"description": "A helm chart for " + appName,
		"type":        "application",
		"version":     "0.1.0",
		"appVersion":  appVersion,
	})
	fp.Close()

	// And finally, create a NOTE.txt file
	fp, _ = os.Create(filepath.Join(templatesDir, "NOTES.txt"))
	fp.WriteString(helm.GenerateNotesFile(ingresses))
	fp.Close()
}
