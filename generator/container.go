package generator

import (
	"fmt"
	"katenary/helm"
	"katenary/logger"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/types"
)

// Generate a container in deployment with all needed objects (volumes, secrets, env, ...).
// The deployName shoud be the name of the deployment, we cannot get it from Metadata as this is a variable name.
func newContainerForDeployment(
	deployName, containerName string,
	deployment *helm.Deployment,
	s *types.ServiceConfig,
	fileGeneratorChan HelmFileGenerator) *helm.Container {

	buildCrontab(deployName, deployment, s, fileGeneratorChan)

	container := helm.NewContainer(containerName, s.Image, s.Environment, s.Labels)

	applyEnvMapLabel(s, container)
	if secretFile := setSecretVar(containerName, s, container); secretFile != nil {
		fileGeneratorChan <- secretFile
		container.EnvFrom = append(container.EnvFrom, map[string]map[string]string{
			"secretRef": {
				"name": secretFile.Metadata().Name,
			},
		})
	}
	setEnvToValues(containerName, s, container)
	prepareContainer(container, s, containerName)
	prepareEnvFromFiles(deployName, s, container, fileGeneratorChan)

	// add the container in deployment
	if deployment.Spec.Template.Spec.Containers == nil {
		deployment.Spec.Template.Spec.Containers = make([]*helm.Container, 0)
	}
	deployment.Spec.Template.Spec.Containers = append(
		deployment.Spec.Template.Spec.Containers,
		container,
	)

	// add the volumes
	if deployment.Spec.Template.Spec.Volumes == nil {
		deployment.Spec.Template.Spec.Volumes = make([]map[string]interface{}, 0)
	}
	// manage LABEL_VOLUMEFROM
	addVolumeFrom(deployment, container, s)
	// and then we can add other volumes
	deployment.Spec.Template.Spec.Volumes = append(
		deployment.Spec.Template.Spec.Volumes,
		prepareVolumes(deployName, containerName, s, container, fileGeneratorChan)...,
	)

	// add init containers
	if deployment.Spec.Template.Spec.InitContainers == nil {
		deployment.Spec.Template.Spec.InitContainers = make([]*helm.Container, 0)
	}
	deployment.Spec.Template.Spec.InitContainers = append(
		deployment.Spec.Template.Spec.InitContainers,
		prepareInitContainers(containerName, s, container)...,
	)

	// check if there is containerPort assigned in label, add it, and do
	// not create service for this.
	if ports, ok := s.Labels[helm.LABEL_CONTAINER_PORT]; ok {
		for _, port := range strings.Split(ports, ",") {
			func(port string, container *helm.Container, s *types.ServiceConfig) {
				port = strings.TrimSpace(port)
				if port == "" {
					return
				}
				portNumber, err := strconv.Atoi(port)
				if err != nil {
					return
				}
				// avoid already declared ports
				for _, p := range s.Ports {
					if int(p.Target) == portNumber {
						return
					}
				}
				container.Ports = append(container.Ports, &helm.ContainerPort{
					Name:          deployName + "-" + port,
					ContainerPort: portNumber,
				})
			}(port, container, s)
		}
	}

	return container
}

// prepareContainer assigns image, command, env, and labels to a container.
func prepareContainer(container *helm.Container, service *types.ServiceConfig, servicename string) {
	// if there is no image name, this should fail!
	if service.Image == "" {
		log.Fatal(ICON_PACKAGE+" No image name for service ", servicename)
	}

	// Get the image tag
	imageParts := strings.Split(service.Image, ":")
	tag := ""
	if len(imageParts) == 2 {
		container.Image = imageParts[0]
		tag = imageParts[1]
	}

	vtag := ".Values." + servicename + ".repository.tag"
	container.Image = `{{ .Values.` + servicename + `.repository.image }}` +
		`{{ if ne ` + vtag + ` "" }}:{{ ` + vtag + ` }}{{ end }}`
	container.Command = service.Command
	AddValues(servicename, map[string]EnvVal{
		"repository": map[string]EnvVal{
			"image": imageParts[0],
			"tag":   tag,
		},
	})
	prepareProbes(servicename, service, container)
	generateContainerPorts(service, servicename, container)
}

// generateContainerPorts add the container ports of a service.
func generateContainerPorts(s *types.ServiceConfig, name string, container *helm.Container) {

	exists := make(map[int]string)
	for _, port := range s.Ports {
		portName := name
		for _, n := range exists {
			if name == n {
				portName = fmt.Sprintf("%s-%d", name, port.Target)
			}
		}
		container.Ports = append(container.Ports, &helm.ContainerPort{
			Name:          portName,
			ContainerPort: int(port.Target),
		})
		exists[int(port.Target)] = name
	}

	// manage the "expose" section to be a NodePort in Kubernetes
	for _, expose := range s.Expose {

		port, _ := strconv.Atoi(expose)

		if _, exist := exists[port]; exist {
			continue
		}
		container.Ports = append(container.Ports, &helm.ContainerPort{
			Name:          name,
			ContainerPort: port,
		})
	}
}

// prepareInitContainers add the init containers of a service.
func prepareInitContainers(name string, s *types.ServiceConfig, container *helm.Container) []*helm.Container {

	// We need to detect others services, but we probably not have parsed them yet, so
	// we will wait for them for a while.
	initContainers := make([]*helm.Container, 0)
	for dp := range s.DependsOn {
		c := helm.NewContainer("check-"+dp, "busybox", nil, s.Labels)
		command := strings.ReplaceAll(strings.TrimSpace(dependScript), "__service__", dp)

		foundPort := -1
		locker.Lock()
		if defaultPort, ok := servicesMap[dp]; !ok {
			logger.Redf("Error while getting port for service %s\n", dp)
			os.Exit(1)
		} else {
			foundPort = defaultPort
		}
		locker.Unlock()
		if foundPort == -1 {
			log.Fatalf(
				"ERROR, the %s service is waiting for %s port number, "+
					"but it is never discovered. You must declare at least one port in "+
					"the \"ports\" section of the service in the docker-compose file",
				name,
				dp,
			)
		}
		command = strings.ReplaceAll(command, "__port__", strconv.Itoa(foundPort))

		c.Command = []string{
			"sh",
			"-c",
			command,
		}
		initContainers = append(initContainers, c)
	}
	return initContainers
}
