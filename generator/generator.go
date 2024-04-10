package generator

// TODO: configmap from files 20%

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"katenary/utils"

	"github.com/compose-spec/compose-go/types"
	goyaml "gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// Generate a chart from a compose project.
// This does not write files to disk, it only creates the HelmChart object.
//
// The Generate function will create the HelmChart object this way:
//
//   - Detect the service port name or leave the port number if not found.
//   - Create a deployment for each service that are not ingnore.
//   - Create a service and ingresses for each service that has ports and/or declared ingresses.
//   - Create a PVC or Configmap volumes for each volume.
//   - Create init containers for each service which has dependencies to other services.
//   - Create a chart dependencies.
//   - Create a configmap and secrets from the environment variables.
//   - Merge the same-pod services.
func Generate(project *types.Project) (*HelmChart, error) {
	var (
		appName     = project.Name
		deployments = make(map[string]*Deployment, len(project.Services))
		services    = make(map[string]*Service)
		podToMerge  = make(map[string]*types.ServiceConfig)
	)
	chart := NewChart(appName)

	// Add the compose files hash to the chart annotations
	hash, err := utils.HashComposefiles(project.ComposeFiles)
	if err != nil {
		return nil, err
	}
	Annotations[KATENARY_PREFIX+"compose-hash"] = hash
	chart.composeHash = &hash

	// find the "main-app" label, and set chart.AppVersion to the tag if exists
	mainCount := 0
	for _, service := range project.Services {
		if serviceIsMain(service) {
			mainCount++
			if mainCount > 1 {
				return nil, fmt.Errorf("found more than one main app")
			}
			setChartVersion(chart, service)
		}
	}
	if mainCount == 0 {
		chart.AppVersion = "0.1.0"
	}

	// first pass, create all deployments whatewer they are.
	for _, service := range project.Services {
		// check the "ports" label from container and add it to the service
		if err := fixPorts(&service); err != nil {
			return nil, err
		}

		// isgnored service
		if isIgnored(service) {
			fmt.Printf("%s Ignoring service %s\n", utils.IconInfo, service.Name)
			continue
		}

		// helm dependency
		if isHelmDependency, err := setDependencies(chart, service); err != nil {
			return nil, err
		} else if isHelmDependency {
			continue
		}

		// create all deployments
		d := NewDeployment(service, chart)
		deployments[service.Name] = d

		// generate the cronjob if needed
		setCronJob(service, chart, appName)

		// get the same-pod label if exists, add it to the list.
		// We later will copy some parts to the target deployment and remove this one.
		if samePod, ok := service.Labels[LABEL_SAME_POD]; ok && samePod != "" {
			podToMerge[samePod] = &service
		}

		// create the needed service for the container port
		if len(service.Ports) > 0 {
			s := NewService(service, appName)
			services[service.Name] = s
		}

		// create all ingresses
		if ingress := d.AddIngress(service, appName); ingress != nil {
			y, _ := ingress.Yaml()
			chart.Templates[ingress.Filename()] = &ChartTemplate{
				Content:     y,
				Servicename: service.Name,
			}
		}
	}

	// now we have all deployments, we can create PVC if needed (it's separated from
	// the above loop because we need all deployments to not duplicate PVC for "same-pod" services)
	for _, service := range project.Services {
		if err := buildVolumes(service, chart, deployments); err != nil {
			return nil, err
		}
	}
	// drop all "same-pod" deployments because the containers and volumes are already
	// in the target deployment
	for _, service := range podToMerge {
		if samepod, ok := service.Labels[LABEL_SAME_POD]; ok && samepod != "" {
			// move this deployment volumes to the target deployment
			if target, ok := deployments[samepod]; ok {
				target.AddContainer(*service)
				target.BindFrom(*service, deployments[service.Name])
				delete(deployments, service.Name)
			} else {
				log.Printf("service %[1]s is declared as %[2]s, but %[2]s is not defined", service.Name, LABEL_SAME_POD)
			}
		}
	}

	// create init containers for all DependsOn
	for _, s := range project.Services {
		for _, d := range s.GetDependencies() {
			if dep, ok := deployments[d]; ok {
				deployments[s.Name].DependsOn(dep, d)
			} else {
				log.Printf("service %[1]s depends on %[2]s, but %[2]s is not defined", s.Name, d)
			}
		}
	}

	// generate configmaps with environment variables
	generateConfigMapsAndSecrets(project, chart)

	// if the env-from label is set, we need to add the env vars from the configmap
	// to the environment of the service
	for _, s := range project.Services {
		setSharedConf(s, chart, deployments)
	}

	// generate yaml files
	for _, d := range deployments {
		y, _ := d.Yaml()
		chart.Templates[d.Filename()] = &ChartTemplate{
			Content:     y,
			Servicename: d.service.Name,
		}
	}

	// generate all services
	for _, s := range services {
		// add the service ports to the target service if it's a "same-pod" service
		if samePod, ok := podToMerge[s.service.Name]; ok {
			// get the target service
			target := services[samePod.Name]
			// merge the services
			s.Spec.Ports = append(s.Spec.Ports, target.Spec.Ports...)
		}
		y, _ := s.Yaml()
		chart.Templates[s.Filename()] = &ChartTemplate{
			Content:     y,
			Servicename: s.service.Name,
		}
	}

	// drop all "same-pod" services
	for _, s := range podToMerge {
		// get the target service
		target := services[s.Name]
		delete(chart.Templates, target.Filename())
	}

	// compute all needed resplacements in YAML templates
	for n, v := range chart.Templates {
		v.Content = removeReplaceString(v.Content)
		v.Content = computeNIndent(v.Content)
		chart.Templates[n].Content = v.Content
	}

	// generate helper
	chart.Helper = Helper(appName)

	return chart, nil
}

// computeNIndentm replace all __indent__ labels with the number of spaces before the label.
func computeNIndent(b []byte) []byte {
	lines := bytes.Split(b, []byte("\n"))
	for i, line := range lines {
		if !bytes.Contains(line, []byte("__indent__")) {
			continue
		}
		startSpaces := ""
		spaces := regexp.MustCompile(`^\s+`).FindAllString(string(line), -1)
		if len(spaces) > 0 {
			startSpaces = spaces[0]
		}
		line = []byte(startSpaces + strings.TrimLeft(string(line), " "))
		line = bytes.ReplaceAll(line, []byte("__indent__"), []byte(fmt.Sprintf("%d", len(startSpaces))))
		lines[i] = line
	}
	return bytes.Join(lines, []byte("\n"))
}

// removeReplaceString replace all __replace_ labels with the value of the
// capture group and remove all new lines and repeated spaces.
//
// we created:
//
//	__replace_bar: '{{ include "foo.labels" .
//	   }}'
//
// note the new line and spaces...
//
// we now want to replace it with {{ include "foo.labels" . }}, without the label name.
func removeReplaceString(b []byte) []byte {
	// replace all matches with the value of the capture group
	// and remove all new lines and repeated spaces
	b = replaceLabelRegexp.ReplaceAllFunc(b, func(b []byte) []byte {
		inc := replaceLabelRegexp.FindSubmatch(b)[1]
		inc = bytes.ReplaceAll(inc, []byte("\n"), []byte(""))
		inc = bytes.ReplaceAll(inc, []byte("\r"), []byte(""))
		inc = regexp.MustCompile(`\s+`).ReplaceAll(inc, []byte(" "))
		return inc
	})
	return b
}

// serviceIsMain returns true if the service is the main app.
func serviceIsMain(service types.ServiceConfig) bool {
	if main, ok := service.Labels[LABEL_MAIN_APP]; ok {
		return main == "true" || main == "yes" || main == "1"
	}
	return false
}

// setChartVersion sets the chart version from the service image tag.
func setChartVersion(chart *HelmChart, service types.ServiceConfig) {
	if chart.Version == "" {
		image := service.Image
		parts := strings.Split(image, ":")
		if len(parts) > 1 {
			chart.AppVersion = parts[1]
		} else {
			chart.AppVersion = "0.1.0"
		}
	}
}

// fixPorts checks the "ports" label from container and add it to the service.
func fixPorts(service *types.ServiceConfig) error {
	// check the "ports" label from container and add it to the service
	if portsLabel, ok := service.Labels[LABEL_PORTS]; ok {
		ports := []uint32{}
		if err := goyaml.Unmarshal([]byte(portsLabel), &ports); err != nil {
			// maybe it's a string, comma separated
			parts := strings.Split(portsLabel, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				port, err := strconv.ParseUint(part, 10, 32)
				if err != nil {
					return err
				}
				ports = append(ports, uint32(port))
			}
		}
		for _, port := range ports {
			service.Ports = append(service.Ports, types.ServicePortConfig{
				Target: port,
			})
		}
	}
	return nil
}

// setCronJob creates a cronjob from the service labels.
func setCronJob(service types.ServiceConfig, chart *HelmChart, appName string) *CronJob {
	if _, ok := service.Labels[LABEL_CRONJOB]; !ok {
		return nil
	}
	cronjob, rbac := NewCronJob(service, chart, appName)
	y, _ := cronjob.Yaml()
	chart.Templates[cronjob.Filename()] = &ChartTemplate{
		Content:     y,
		Servicename: service.Name,
	}

	if rbac != nil {
		y, _ := rbac.RoleBinding.Yaml()
		chart.Templates[rbac.RoleBinding.Filename()] = &ChartTemplate{
			Content:     y,
			Servicename: service.Name,
		}
		y, _ = rbac.Role.Yaml()
		chart.Templates[rbac.Role.Filename()] = &ChartTemplate{
			Content:     y,
			Servicename: service.Name,
		}
		y, _ = rbac.ServiceAccount.Yaml()
		chart.Templates[rbac.ServiceAccount.Filename()] = &ChartTemplate{
			Content:     y,
			Servicename: service.Name,
		}
	}

	return cronjob
}

// setDependencies sets the dependencies from the service labels.
func setDependencies(chart *HelmChart, service types.ServiceConfig) (bool, error) {
	// helm dependency
	if v, ok := service.Labels[LABEL_DEPENDENCIES]; ok {
		d := []Dependency{}
		if err := yaml.Unmarshal([]byte(v), &d); err != nil {
			return false, err
		}

		for _, dep := range d {
			fmt.Printf("%s Adding dependency to %s\n", utils.IconDependency, dep.Name)
			chart.Dependencies = append(chart.Dependencies, dep)
			name := dep.Name
			if dep.Alias != "" {
				name = dep.Alias
			}
			// add the dependency env vars to the values.yaml
			chart.Values[name] = dep.Values
		}

		return true, nil
	}
	return false, nil
}

// isIgnored returns true if the service is ignored.
func isIgnored(service types.ServiceConfig) bool {
	if v, ok := service.Labels[LABEL_IGNORE]; ok {
		return v == "true" || v == "yes" || v == "1"
	}
	return false
}

// buildVolumes creates the volumes for the service.
func buildVolumes(service types.ServiceConfig, chart *HelmChart, deployments map[string]*Deployment) error {
	appName := chart.Name
	for _, v := range service.Volumes {
		// Do not add volumes if the pod is injected in a deployments
		// via "same-pod" and the volume in destination deployment exists
		if samePodVolume(service, v, deployments) {
			continue
		}
		switch v.Type {
		case "volume":
			pvc := NewVolumeClaim(service, v.Source, appName)

			// if the service is integrated in another deployment, we need to add the volume
			// to the target deployment
			if override, ok := service.Labels[LABEL_SAME_POD]; ok {
				pvc.nameOverride = override
				pvc.Spec.StorageClassName = utils.StrPtr(`{{ .Values.` + override + `.persistence.` + v.Source + `.storageClass }}`)
				chart.Values[override].(*Value).AddPersistence(v.Source)
			}
			y, _ := pvc.Yaml()
			chart.Templates[pvc.Filename()] = &ChartTemplate{
				Content:     y,
				Servicename: service.Name, // TODO, use name
			}

		case "bind":
			// ensure the path is in labels
			bindPath := map[string]string{}
			if _, ok := service.Labels[LABEL_CM_FILES]; ok {
				files := []string{}
				if err := yaml.Unmarshal([]byte(service.Labels[LABEL_CM_FILES]), &files); err != nil {
					return err
				}
				for _, f := range files {
					bindPath[f] = f
				}
			}
			if _, ok := bindPath[v.Source]; !ok {
				continue
			}

			cm := NewConfigMapFromFiles(service, appName, v.Source)
			var err error
			var y []byte
			if y, err = cm.Yaml(); err != nil {
				log.Fatal(err)
			}
			chart.Templates[cm.Filename()] = &ChartTemplate{
				Content:     y,
				Servicename: service.Name,
			}

			// continue with subdirectories
			stat, err := os.Stat(v.Source)
			if err != nil {
				return err
			}
			if stat.IsDir() {
				files, err := filepath.Glob(filepath.Join(v.Source, "*"))
				if err != nil {
					return err
				}
				for _, f := range files {
					if f == v.Source {
						continue
					}
					if stat, err := os.Stat(f); err != nil || !stat.IsDir() {
						continue
					}
					cm := NewConfigMapFromFiles(service, appName, f)
					var err error
					var y []byte
					if y, err = cm.Yaml(); err != nil {
						log.Fatal(err)
					}
					log.Printf("Adding configmap %s %s", cm.Filename(), f)
					chart.Templates[cm.Filename()] = &ChartTemplate{
						Content:     y,
						Servicename: service.Name,
					}
				}
			}

		}
	}
	return nil
}

// generateConfigMapsAndSecrets creates the configmaps and secrets from the environment variables.
func generateConfigMapsAndSecrets(project *types.Project, chart *HelmChart) error {
	appName := chart.Name
	for _, s := range project.Services {
		if s.Environment == nil || len(s.Environment) == 0 {
			continue
		}

		originalEnv := types.MappingWithEquals{}
		secretsVar := types.MappingWithEquals{}

		// copy env to originalEnv
		for k, v := range s.Environment {
			originalEnv[k] = v
		}

		if v, ok := s.Labels[LABEL_SECRETS]; ok {
			list := []string{}
			if err := yaml.Unmarshal([]byte(v), &list); err != nil {
				log.Fatal("error unmarshaling secrets label:", err)
			}
			for _, secret := range list {
				if secret == "" {
					continue
				}
				if _, ok := s.Environment[secret]; !ok {
					fmt.Printf("%s secret %s not found in environment", utils.IconWarning, secret)
					continue
				}
				secretsVar[secret] = s.Environment[secret]
			}
		}

		if len(secretsVar) > 0 {
			s.Environment = secretsVar
			sec := NewSecret(s, appName)
			y, _ := sec.Yaml()
			name := sec.service.Name
			chart.Templates[name+".secret.yaml"] = &ChartTemplate{
				Content:     y,
				Servicename: s.Name,
			}
		}

		// remove secrets from env
		s.Environment = originalEnv // back to original
		for k := range secretsVar {
			delete(s.Environment, k)
		}
		if len(s.Environment) > 0 {
			cm := NewConfigMap(s, appName)
			y, _ := cm.Yaml()
			name := cm.service.Name
			chart.Templates[name+".configmap.yaml"] = &ChartTemplate{
				Content:     y,
				Servicename: s.Name,
			}
		}
	}
	return nil
}

// samePodVolume returns true if the volume is already in the target deployment.
func samePodVolume(service types.ServiceConfig, v types.ServiceVolumeConfig, deployments map[string]*Deployment) bool {
	// if the service has volumes, and it has "same-pod" label
	// - get the target deployment
	// - check if it has the same volume
	// if not, return false

	if v.Source == "" {
		return false
	}

	if service.Volumes == nil || len(service.Volumes) == 0 {
		return false
	}

	targetDeployment := ""
	if targetName, ok := service.Labels[LABEL_SAME_POD]; !ok {
		return false
	} else {
		targetDeployment = targetName
	}

	// get the target deployment
	var target *Deployment
	for _, d := range deployments {
		if d.service.Name == targetDeployment {
			target = d
			break
		}
	}
	if target == nil {
		return false
	}

	// check if it has the same volume
	for _, tv := range target.Spec.Template.Spec.Volumes {
		if tv.Name == v.Source {
			log.Printf("found same pod volume %s in deployment %s and %s", tv.Name, service.Name, targetDeployment)
			return true
		}
	}
	return false
}

// setSharedConf sets the shared configmap to the service.
func setSharedConf(service types.ServiceConfig, chart *HelmChart, deployments map[string]*Deployment) {
	// if the service has the "shared-conf" label, we need to add the configmap
	// to the chart and add the env vars to the service
	if _, ok := service.Labels[LABEL_ENV_FROM]; !ok {
		return
	}
	fromservices := []string{}
	if err := yaml.Unmarshal([]byte(service.Labels[LABEL_ENV_FROM]), &fromservices); err != nil {
		log.Fatal("error unmarshaling env-from label:", err)
	}
	// find the configmap in the chart templates
	for _, fromservice := range fromservices {
		if _, ok := chart.Templates[fromservice+".configmap.yaml"]; !ok {
			log.Printf("configmap %s not found in chart templates", fromservice)
			continue
		}
		// find the corresponding target deployment
		var target *Deployment
		for _, d := range deployments {
			if d.service.Name == service.Name {
				target = d
				break
			}
		}
		if target == nil {
			continue
		}
		// add the configmap to the service
		for i, c := range target.Spec.Template.Spec.Containers {
			if c.Name != service.Name {
				continue
			}
			c.EnvFrom = append(c.EnvFrom, corev1.EnvFromSource{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: utils.TplName(fromservice, chart.Name),
					},
				},
			})
			target.Spec.Template.Spec.Containers[i] = c
		}
	}
}
