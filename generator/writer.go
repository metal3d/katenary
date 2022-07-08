package generator

import (
	"bytes"
	"katenary/compose"
	"katenary/generator/writers"
	"katenary/helm"
	"katenary/tools"
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

// HelmFile represents a helm file from helm package that has got some necessary methods
// to generate a helm file.
type HelmFile interface {
	GetType() string
	GetPathRessource() string
}

// HelmFileGenerator is a chanel of HelmFile.
type HelmFileGenerator chan HelmFile

var PrefixRE = regexp.MustCompile(`\{\{.*\}\}-?`)
var helmDependencies = map[string]*helm.Dependency{}

func portExists(port int, ports []types.ServicePortConfig) bool {
	for _, p := range ports {
		if p.Target == uint32(port) {
			log.Println("portExists:", port, p.Target)
			return true
		}
	}
	return false
}

// Generate get a parsed compose file, and generate the helm files.
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

	generators := make(map[string]HelmFileGenerator)

	// Remove skipped services from the parsed data.
	for i, service := range p.Data.Services {
		if v, ok := service.Labels[helm.LABEL_IGNORE]; !ok || v != "true" {
			continue
		}
		p.Data.Services = append(p.Data.Services[:i], p.Data.Services[i+1:]...)
		i--

		// find this service in others as "depends_on" and remove it
		for _, service2 := range p.Data.Services {
			delete(service2.DependsOn, service.Name)
		}
	}

	for i, service := range p.Data.Services {
		if _, ok := service.Labels[helm.LABEL_DEPENDENCIES]; ok {
			if v, ok := service.Labels[helm.LABEL_SAMEPOD]; ok && v != "" {
				log.Fatal("You cannot set a service in a same pod and have dependencies:", service.Name)
			}
			dependency := manageDependencies(service)
			// the service name is sometimes different from the dependency name.
			var servicename string
			if dependency.Config.ServiceName == "" {
				servicename = helm.ReleaseNameTpl + "-" + dependency.Name
			} else {
				servicename = dependency.Config.ServiceName
			}
			for j, service2 := range p.Data.Services {
				for name, dep := range service2.DependsOn {
					if name == service.Name {
						p.Data.Services[j].DependsOn[servicename] = dep
						delete(p.Data.Services[j].DependsOn, name)
					}
				}
			}
			// force the service name to the dependency name
			p.Data.Services[i].Name = servicename
			service.Name = servicename
			// add environment to the values
			AddValues(dependency.Name, *dependency.Config.Environment)
			// to no export this in Chart.yaml file
			dependency.Config = nil
			//helmDependencies = append(helmDependencies, d)
			helmDependencies[service.Name] = dependency
		}

		// if the service port is declared in labels, add it to the service.
		if ports, ok := service.Labels[helm.LABEL_PORT]; ok {
			if service.Ports == nil {
				service.Ports = make([]types.ServicePortConfig, 0)
			}
			for _, port := range strings.Split(ports, ",") {
				port = strings.TrimSpace(port)
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
				servicesMap[service.Name] = target
				break
			}
		}

		// manage emptyDir volumes
		if empty, ok := service.Labels[helm.LABEL_EMPTYDIRS]; ok {
			//split empty list by coma
			emptyDirs := strings.Split(empty, ",")
			for i, emptyDir := range emptyDirs {
				emptyDirs[i] = strings.TrimSpace(emptyDir)
			}
			//append them in EmptyDirs
			EmptyDirs = append(EmptyDirs, emptyDirs...)
		}
		p.Data.Services[i] = service

	}

	// for all services in linked map, and not in samePods map, generate the service
	for _, s := range p.Data.Services {
		// do not make a deployment for services declared to be in the same pod than another
		if _, ok := s.Labels[helm.LABEL_SAMEPOD]; ok {
			continue
		}

		// find services that is in the same pod
		linked := make(map[string]types.ServiceConfig, 0)
		for _, service := range p.Data.Services {
			n := service.Name
			if linkname, ok := service.Labels[helm.LABEL_SAMEPOD]; ok && linkname == s.Name {
				linked[n] = service
				delete(s.DependsOn, n)
			}
		}

		// do not generate a deployment for services that are replaced by dependencies
		if _, found := helmDependencies[s.Name]; found {
			continue
		}

		generators[s.Name] = CreateReplicaObject(s.Name, s, linked)
	}

	// to generate notes, we need to keep an Ingresses list
	ingresses := make(map[string]*helm.Ingress)

	for serviceName, generator := range generators { // generators is a map : name -> generator
		for helmFile := range generator { // generator is a chan
			if helmFile == nil { // generator finished
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
				writers.BuildStorage(c, serviceName, templatesDir)

			case *helm.Deployment:
				// for the deployment, we need to fix persitence volumes
				// to be activated only when the storage is "enabled",
				// either we use an "emptyDir"
				writers.BuildDeployment(c, serviceName, templatesDir)

			case *helm.Service:
				// Change the type for service if it's an "exposed" port
				writers.BuildService(c, serviceName, templatesDir)

			case *helm.Ingress:
				// we need to make ingresses "activable" from values
				ingresses[serviceName] = c // keep it to generate notes
				writers.BuildIngress(c, serviceName, templatesDir)

			case *helm.ConfigMap, *helm.Secret:
				// there could be several files, so let's force the filename
				name := c.(helm.Named).Name() + "." + c.GetType()
				suffix := c.GetPathRessource()
				suffix = tools.PathToName(suffix)
				name += suffix
				name = PrefixRE.ReplaceAllString(name, "")
				writers.BuildConfigMap(c, kind, serviceName, name, templatesDir)

			default:
				name := c.(helm.Named).Name() + "." + c.GetType()
				name = PrefixRE.ReplaceAllString(name, "")
				fname := filepath.Join(templatesDir, name+".yaml")
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

	// Create a buffer to write lines with a number (damned yaml order)
	buff := bytes.NewBuffer(nil)
	enc = yaml.NewEncoder(buff)
	enc.SetIndent(writers.IndentSize)
	chart := map[string]interface{}{
		"0-apiVersion":  "v2",
		"1-name":        appName,
		"2-description": "A helm chart for " + appName,
		"3-type":        "application",
		"4-version":     chartVersion,
		"5-appVersion":  appVersion,
	}
	if len(helmDependencies) > 0 {
		dep := make([]interface{}, len(helmDependencies))
		count := 0
		for _, d := range helmDependencies {
			dep[count] = d
			count++
		}
		chart["6-dependencies"] = dep
	}
	enc.Encode(chart)

	// and now that the yaml is written in right order, remove each number
	// of lines to write them in chart file...
	lineNumRegExp := regexp.MustCompile(`^(\d+)-`)
	lines := []string{}
	for _, line := range strings.Split(buff.String(), "\n") {
		line = lineNumRegExp.ReplaceAllString(line, "")
		lines = append(lines, line)
	}
	chartFile.WriteString(strings.Join(lines, "\n"))

	// And finally, create a NOTE.txt file
	noteFile, err := os.Create(filepath.Join(templatesDir, "NOTES.txt"))
	if err != nil {
		log.Fatal(err)
	}
	defer noteFile.Close()
	noteFile.WriteString(helm.GenerateNotesFile(ingresses))
}

func manageDependencies(s types.ServiceConfig) *helm.Dependency {
	dep, ok := s.Labels[helm.LABEL_DEPENDENCIES]
	if !ok {
		return nil
	}

	var dependencies *helm.Dependency
	if err := yaml.Unmarshal([]byte(dep), &dependencies); err != nil {
		log.Fatal(err)
	}
	return dependencies
}
