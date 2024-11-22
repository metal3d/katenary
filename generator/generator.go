package generator

import (
	"bytes"
	"fmt"
	"katenary/generator/labels"
	"katenary/utils"
	"log"
	"regexp"
	"strings"

	"github.com/compose-spec/compose-go/types"
	corev1 "k8s.io/api/core/v1"
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
	Annotations[labels.LabelName("compose-hash")] = hash
	chart.composeHash = &hash

	// find the "main-app" label, and set chart.AppVersion to the tag if exists
	mainCount := 0
	for _, service := range project.Services {
		if serviceIsMain(service) {
			mainCount++
			if mainCount > 1 {
				return nil, fmt.Errorf("found more than one main app")
			}
			chart.setChartVersion(service)
		}
	}
	if mainCount == 0 {
		chart.AppVersion = "0.1.0"
	}

	// first pass, create all deployments whatewer they are.
	for _, service := range project.Services {
		err := chart.generateDeployment(service, deployments, services, podToMerge, appName)
		if err != nil {
			return nil, err
		}
	}

	// now we have all deployments, we can create PVC if needed (it's separated from
	// the above loop because we need all deployments to not duplicate PVC for "same-pod" services)
	// bind static volumes
	for _, service := range project.Services {
		addStaticVolumes(deployments, service)
	}
	for _, service := range project.Services {
		err := buildVolumes(service, chart, deployments)
		if err != nil {
			return nil, err
		}
	}

	// if we have built exchange volumes, we need to moint them in each deployment
	for _, d := range deployments {
		d.MountExchangeVolumes()
	}

	// drop all "same-pod" deployments because the containers and volumes are already
	// in the target deployment
	for _, service := range podToMerge {
		if samepod, ok := service.Labels[labels.LabelSamePod]; ok && samepod != "" {
			// move this deployment volumes to the target deployment
			if target, ok := deployments[samepod]; ok {
				target.AddContainer(*service)
				target.BindFrom(*service, deployments[service.Name])
				target.SetEnvFrom(*service, appName, true)
				// copy all init containers
				initContainers := deployments[service.Name].Spec.Template.Spec.InitContainers
				target.Spec.Template.Spec.InitContainers = append(target.Spec.Template.Spec.InitContainers, initContainers...)
				delete(deployments, service.Name)
			} else {
				log.Printf("service %[1]s is declared as %[2]s, but %[2]s is not defined", service.Name, labels.LabelSamePod)
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
	chart.generateConfigMapsAndSecrets(project)

	// if the env-from label is set, we need to add the env vars from the configmap
	// to the environment of the service
	for _, s := range project.Services {
		chart.setSharedConf(s, deployments)
	}

	// generate yaml files
	for _, d := range deployments {
		y, err := d.Yaml()
		if err != nil {
			return nil, err
		}
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
		if target != nil {
			delete(chart.Templates, target.Filename())
		}
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

// serviceIsMain returns true if the service is the main app.
func serviceIsMain(service types.ServiceConfig) bool {
	if main, ok := service.Labels[labels.LabelMainApp]; ok {
		return main == "true" || main == "yes" || main == "1"
	}
	return false
}

func addStaticVolumes(deployments map[string]*Deployment, service types.ServiceConfig) {
	// add the bound configMaps files to the deployment containers
	var d *Deployment
	var ok bool
	if d, ok = deployments[service.Name]; !ok {
		log.Printf("service %s not found in deployments", service.Name)
		return
	}

	container, index := utils.GetContainerByName(service.Name, d.Spec.Template.Spec.Containers)
	if container == nil { // may append for the same-pod services
		return
	}
	for volumeName, config := range d.configMaps {
		var y []byte
		var err error
		if y, err = config.configMap.Yaml(); err != nil {
			log.Fatal(err)
		}
		// add the configmap to the chart
		d.chart.Templates[config.configMap.Filename()] = &ChartTemplate{
			Content:     y,
			Servicename: d.service.Name,
		}
		// add the moint path to the container
		for _, m := range config.mountPath {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      utils.PathToName(volumeName),
				MountPath: m.mountPath,
				SubPath:   m.subPath,
			})
		}

		d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: utils.PathToName(volumeName),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: config.configMap.Name,
					},
				},
			},
		})
	}

	d.Spec.Template.Spec.Containers[index] = *container
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
			v.Source = utils.AsResourceName(v.Source)
			pvc := NewVolumeClaim(service, v.Source, appName)

			// if the service is integrated in another deployment, we need to add the volume
			// to the target deployment
			if override, ok := service.Labels[labels.LabelSamePod]; ok {
				pvc.nameOverride = override
				pvc.Spec.StorageClassName = utils.StrPtr(`{{ .Values.` + override + `.persistence.` + v.Source + `.storageClass }}`)
				chart.Values[override].(*Value).AddPersistence(v.Source)
			}
			y, _ := pvc.Yaml()
			chart.Templates[pvc.Filename()] = &ChartTemplate{
				Content:     y,
				Servicename: service.Name,
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
	if targetName, ok := service.Labels[labels.LabelSamePod]; !ok {
		return false
	} else {
		targetDeployment = targetName
	}

	// get the target deployment
	target := findDeployment(targetDeployment, deployments)
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
