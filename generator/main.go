package generator

import (
	"fmt"
	"io/ioutil"
	"katenary/compose"
	"katenary/helm"
	"katenary/logger"
	"katenary/tools"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

type EnvVal = helm.EnvValue

const (
	ICON_PACKAGE = "📦"
	ICON_SERVICE = "🔌"
	ICON_SECRET  = "🔏"
	ICON_CONF    = "📝"
	ICON_STORE   = "⚡"
	ICON_INGRESS = "🌐"
	ICON_RBAC    = "🔑"
	ICON_CRON    = "🕒"
)

// Values is kept in memory to create a values.yaml file.
var (
	Values       = make(map[string]map[string]interface{})
	VolumeValues = make(map[string]map[string]map[string]EnvVal)
	EmptyDirs    = []string{}
	servicesMap  = make(map[string]int)
	locker       = &sync.Mutex{}

	dependScript = `
OK=0
echo "Checking __service__ port"
while [ $OK != 1 ]; do
    echo -n "."
    nc -z ` + helm.ReleaseNameTpl + `-__service__ __port__ 2>&1 >/dev/null && OK=1 || sleep 1
done
echo
echo "Done"
`

	madeDeployments = make(map[string]helm.Deployment, 0)
)

// Create a Deployment for a given compose.Service. It returns a list chan
// of HelmFileGenerator which will be used to generate the files (deployment, secrets, configMap...).
func CreateReplicaObject(name string, s types.ServiceConfig, linked map[string]types.ServiceConfig) HelmFileGenerator {
	ret := make(chan HelmFile, runtime.NumCPU())
	// there is a bug woth typs.ServiceConfig if we use the pointer. So we need to dereference it.
	go buildDeployment(name, &s, linked, ret)
	return ret
}

// This function will try to yied deployment and services based on a service from the compose file structure.
func buildDeployment(name string, s *types.ServiceConfig, linked map[string]types.ServiceConfig, fileGeneratorChan HelmFileGenerator) {

	logger.Magenta(ICON_PACKAGE+" Generating deployment for ", name)
	deployment := helm.NewDeployment(name)

	newContainerForDeployment(name, name, deployment, s, fileGeneratorChan)

	// Add selectors
	selectors := buildSelector(name, s)
	selectors[helm.K+"/resource"] = "deployment"
	deployment.Spec.Selector = map[string]interface{}{
		"matchLabels": selectors,
	}
	deployment.Spec.Template.Metadata.Labels = selectors

	// Now, the linked services (same pod)
	for lname, link := range linked {
		newContainerForDeployment(name, lname, deployment, &link, fileGeneratorChan)
		// append ports and expose ports to the deployment,
		// to be able to generate them in the Service file
		if len(link.Ports) > 0 || len(link.Expose) > 0 {
			s.Ports = append(s.Ports, link.Ports...)
			s.Expose = append(s.Expose, link.Expose...)
		}
	}

	// Remove duplicates in volumes
	volumes := make([]map[string]interface{}, 0)
	done := make(map[string]bool)
	for _, vol := range deployment.Spec.Template.Spec.Volumes {
		name := vol["name"].(string)
		if _, ok := done[name]; ok {
			continue
		} else {
			done[name] = true
			volumes = append(volumes, vol)
		}
	}
	deployment.Spec.Template.Spec.Volumes = volumes

	// Then, create Services and possible Ingresses for ingress labels, "ports" and "expose" section
	if len(s.Ports) > 0 || len(s.Expose) > 0 {
		for _, s := range generateServicesAndIngresses(name, s) {
			if s != nil {
				fileGeneratorChan <- s
			}
		}
	}

	// add the volumes in Values
	if len(VolumeValues[name]) > 0 {
		AddValues(name, map[string]EnvVal{"persistence": VolumeValues[name]})
	}

	// the deployment is ready, give it
	fileGeneratorChan <- deployment

	// and then, we can say that it's the end
	fileGeneratorChan <- nil
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

// Create a service (k8s).
func generateServicesAndIngresses(name string, s *types.ServiceConfig) []HelmFile {

	ret := make([]HelmFile, 0) // can handle helm.Service or helm.Ingress
	logger.Magenta(ICON_SERVICE+" Generating service for ", name)
	ks := helm.NewService(name)

	for _, p := range s.Ports {
		target := int(p.Target)
		ks.Spec.Ports = append(ks.Spec.Ports, helm.NewServicePort(target, target))
	}
	ks.Spec.Selector = buildSelector(name, s)

	ret = append(ret, ks)
	if v, ok := s.Labels[helm.LABEL_INGRESS]; ok {
		port, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("The given port \"%v\" as ingress port in \"%s\" service is not an integer\n", v, name)
		}
		logger.Cyanf(ICON_INGRESS+" Create an ingress for port %d on %s service\n", port, name)
		ing := createIngress(name, port, s)
		ret = append(ret, ing)
	}

	if len(s.Expose) > 0 {
		logger.Magenta(ICON_SERVICE+" Generating service for ", name+"-external")
		ks := helm.NewService(name + "-external")
		ks.Spec.Type = "NodePort"
		for _, expose := range s.Expose {

			p, _ := strconv.Atoi(expose)
			ks.Spec.Ports = append(ks.Spec.Ports, helm.NewServicePort(p, p))
		}
		ks.Spec.Selector = buildSelector(name, s)
		ret = append(ret, ks)
	}

	return ret
}

// Create an ingress.
func createIngress(name string, port int, s *types.ServiceConfig) *helm.Ingress {
	ingress := helm.NewIngress(name)

	annotations := map[string]string{}
	ingressVal := map[string]interface{}{
		"class":       "nginx",
		"host":        name + "." + helm.Appname + ".tld",
		"enabled":     false,
		"annotations": annotations,
	}

	// add Annotations in values
	AddValues(name, map[string]EnvVal{"ingress": ingressVal})

	ingress.Spec.Rules = []helm.IngressRule{
		{
			Host: fmt.Sprintf("{{ .Values.%s.ingress.host }}", name),
			Http: helm.IngressHttp{
				Paths: []helm.IngressPath{{
					Path:     "/",
					PathType: "Prefix",
					Backend: &helm.IngressBackend{
						Service: helm.IngressService{
							Name: helm.ReleaseNameTpl + "-" + name,
							Port: map[string]interface{}{
								"number": port,
							},
						},
					},
				}},
			},
		},
	}
	ingress.SetIngressClass(name)

	return ingress
}

// Build the selector for the service.
func buildSelector(name string, s *types.ServiceConfig) map[string]string {
	return map[string]string{
		"katenary.io/component": name,
		"katenary.io/release":   helm.ReleaseNameTpl,
	}
}

// buildConfigMapFromPath generates a ConfigMap from a path.
func buildConfigMapFromPath(name, path string) *helm.ConfigMap {
	stat, err := os.Stat(path)
	if err != nil {
		return nil
	}

	files := make(map[string]string, 0)
	if stat.IsDir() {
		found, _ := filepath.Glob(path + "/*")
		for _, f := range found {
			if s, err := os.Stat(f); err != nil || s.IsDir() {
				if err != nil {
					fmt.Fprintf(os.Stderr, "An error occured reading volume path %s\n", err.Error())
				} else {
					logger.ActivateColors = true
					logger.Yellowf("Warning, %s is a directory, at this time we only "+
						"can create configmap for first level file list\n", f)
					logger.ActivateColors = false
				}
				continue
			}
			_, filename := filepath.Split(f)
			c, _ := ioutil.ReadFile(f)
			files[filename] = string(c)
		}
	} else {
		c, _ := ioutil.ReadFile(path)
		_, filename := filepath.Split(path)
		files[filename] = string(c)
	}

	cm := helm.NewConfigMap(name, tools.GetRelPath(path))
	cm.Data = files
	return cm
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

// prepareVolumes add the volumes of a service.
func prepareVolumes(deployment, name string, s *types.ServiceConfig, container *helm.Container, fileGeneratorChan HelmFileGenerator) []map[string]interface{} {

	volumes := make([]map[string]interface{}, 0)
	mountPoints := make([]interface{}, 0)
	configMapsVolumes := make([]string, 0)
	if v, ok := s.Labels[helm.LABEL_VOL_CM]; ok {
		configMapsVolumes = strings.Split(v, ",")
		for i, cm := range configMapsVolumes {
			configMapsVolumes[i] = strings.TrimSpace(cm)
		}
	}

	for _, vol := range s.Volumes {

		volname := vol.Source
		volepath := vol.Target

		if volname == "" {
			logger.ActivateColors = true
			logger.Yellowf("Warning, volume source to %s is empty for %s -- skipping\n", volepath, name)
			logger.ActivateColors = false
			continue
		}

		isConfigMap := false
		for _, cmVol := range configMapsVolumes {
			if tools.GetRelPath(volname) == cmVol {
				isConfigMap = true
				break
			}
		}

		// local volume cannt be mounted
		if !isConfigMap && (strings.HasPrefix(volname, ".") || strings.HasPrefix(volname, "/")) {
			logger.ActivateColors = true
			logger.Redf("You cannot, at this time, have local volume in %s deployment\n", name)
			logger.ActivateColors = false
			continue
		}
		if isConfigMap {
			// check if the volname path points on a file, if so, we need to add subvolume to the interface
			stat, err := os.Stat(volname)
			if err != nil {
				logger.ActivateColors = true
				logger.Redf("An error occured reading volume path %s\n", err.Error())
				logger.ActivateColors = false
				continue
			}
			pointToFile := ""
			if !stat.IsDir() {
				pointToFile = filepath.Base(volname)
			}

			// the volume is a path and it's explicitally asked to be a configmap in labels
			cm := buildConfigMapFromPath(name, volname)
			cm.K8sBase.Metadata.Name = helm.ReleaseNameTpl + "-" + name + "-" + tools.PathToName(volname)

			// build a configmapRef for this volume
			volname := tools.PathToName(volname)
			volumes = append(volumes, map[string]interface{}{
				"name": volname,
				"configMap": map[string]string{
					"name": cm.K8sBase.Metadata.Name,
				},
			})
			if len(pointToFile) > 0 {
				mountPoints = append(mountPoints, map[string]interface{}{
					"name":      volname,
					"mountPath": volepath,
					"subPath":   pointToFile,
				})
			} else {
				mountPoints = append(mountPoints, map[string]interface{}{
					"name":      volname,
					"mountPath": volepath,
				})
			}
			if cm != nil {
				fileGeneratorChan <- cm
			}
		} else {
			// It's a Volume. Mount this from PVC to declare.

			volname = strings.ReplaceAll(volname, "-", "")

			isEmptyDir := false
			for _, v := range EmptyDirs {
				v = strings.ReplaceAll(v, "-", "")
				if v == volname {
					volumes = append(volumes, map[string]interface{}{
						"name":     volname,
						"emptyDir": map[string]string{},
					})
					mountPoints = append(mountPoints, map[string]interface{}{
						"name":      volname,
						"mountPath": volepath,
					})
					container.VolumeMounts = append(container.VolumeMounts, mountPoints...)
					isEmptyDir = true
					break
				}
			}
			if isEmptyDir {
				continue
			}

			volumes = append(volumes, map[string]interface{}{
				"name": volname,
				"persistentVolumeClaim": map[string]string{
					"claimName": helm.ReleaseNameTpl + "-" + volname,
				},
			})
			mountPoints = append(mountPoints, map[string]interface{}{
				"name":      volname,
				"mountPath": volepath,
			})

			logger.Yellow(ICON_STORE+" Generate volume values", volname, "for container named", name, "in deployment", deployment)
			AddVolumeValues(deployment, volname, map[string]EnvVal{
				"enabled":  false,
				"capacity": "1Gi",
			})

			if pvc := helm.NewPVC(deployment, volname); pvc != nil {
				fileGeneratorChan <- pvc
			}
		}
	}
	// add the volume in the container and return the volume definition to add in Deployment
	container.VolumeMounts = append(container.VolumeMounts, mountPoints...)
	return volumes
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

// prepareProbes generate http/tcp/command probes for a service.
func prepareProbes(name string, s *types.ServiceConfig, container *helm.Container) {
	// first, check if there a label for the probe
	if check, ok := s.Labels[helm.LABEL_HEALTHCHECK]; ok {
		check = strings.TrimSpace(check)
		p := helm.NewProbeFromService(s)
		// get the port of the "url" check
		if checkurl, err := url.Parse(check); err == nil {
			if err == nil {
				container.LivenessProbe = buildProtoProbe(p, checkurl)
			}
		} else {
			// it's a command
			container.LivenessProbe = p
			container.LivenessProbe.Exec = &helm.Exec{
				Command: []string{
					"sh",
					"-c",
					check,
				},
			}
		}
		return // label overrides everything
	}

	// if not, we will use the default one
	if s.HealthCheck != nil {
		container.LivenessProbe = buildCommandProbe(s)
	}
}

// buildProtoProbe builds a probe from a url that can be http or tcp.
func buildProtoProbe(probe *helm.Probe, u *url.URL) *helm.Probe {
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		port = 80
	}

	path := "/"
	if u.Path != "" {
		path = u.Path
	}

	switch u.Scheme {
	case "http", "https":
		probe.HttpGet = &helm.HttpGet{
			Path: path,
			Port: port,
		}
	case "tcp":
		probe.TCP = &helm.TCP{
			Port: port,
		}
	default:
		logger.Redf("Error while parsing healthcheck url %s\n", u.String())
		os.Exit(1)
	}
	return probe
}

func buildCommandProbe(s *types.ServiceConfig) *helm.Probe {

	// Get the first element of the command from ServiceConfig
	first := s.HealthCheck.Test[0]

	p := helm.NewProbeFromService(s)
	switch first {
	case "CMD", "CMD-SHELL":
		// CMD or CMD-SHELL
		p.Exec = &helm.Exec{
			Command: s.HealthCheck.Test[1:],
		}
		return p
	default:
		// badly made but it should work...
		p.Exec = &helm.Exec{
			Command: []string(s.HealthCheck.Test),
		}
		return p
	}
}

// prepareEnvFromFiles generate configMap or secrets from environment files.
func prepareEnvFromFiles(name string, s *types.ServiceConfig, container *helm.Container, fileGeneratorChan HelmFileGenerator) {

	// prepare secrets
	secretsFiles := make([]string, 0)
	if v, ok := s.Labels[helm.LABEL_ENV_SECRET]; ok {
		secretsFiles = strings.Split(v, ",")
	}

	var secretVars []string
	if v, ok := s.Labels[helm.LABEL_SECRETVARS]; ok {
		secretVars = strings.Split(v, ",")
	}

	for i, s := range secretVars {
		secretVars[i] = strings.TrimSpace(s)
	}

	// manage environment files (env_file in compose)
	for _, envfile := range s.EnvFile {
		f := tools.PathToName(envfile)
		f = strings.ReplaceAll(f, ".env", "")
		isSecret := false
		for _, s := range secretsFiles {
			s = strings.TrimSpace(s)
			if s == envfile {
				isSecret = true
			}
		}
		var store helm.InlineConfig
		if !isSecret {
			logger.Bluef(ICON_CONF+" Generating configMap from %s\n", envfile)
			store = helm.NewConfigMap(name, envfile)
		} else {
			logger.Bluef(ICON_SECRET+" Generating secret from %s\n", envfile)
			store = helm.NewSecret(name, envfile)
		}

		envfile = filepath.Join(compose.GetCurrentDir(), envfile)
		if err := store.AddEnvFile(envfile, secretVars); err != nil {
			logger.ActivateColors = true
			logger.Red(err.Error())
			logger.ActivateColors = false
			os.Exit(2)
		}

		section := "configMapRef"
		if isSecret {
			section = "secretRef"
		}

		container.EnvFrom = append(container.EnvFrom, map[string]map[string]string{
			section: {
				"name": store.Metadata().Name,
			},
		})

		// read the envfile and remove them from the container environment or secret
		envs := readEnvFile(envfile)
		for varname := range envs {
			if !isSecret {
				// remove varname from container
				for i, s := range container.Env {
					if s.Name == varname {
						container.Env = append(container.Env[:i], container.Env[i+1:]...)
						i--
					}
				}
			}
		}

		if store != nil {
			fileGeneratorChan <- store.(HelmFile)
		}
	}
}

// AddValues adds values to the values.yaml map.
func AddValues(servicename string, values map[string]EnvVal) {
	locker.Lock()
	defer locker.Unlock()

	if _, ok := Values[servicename]; !ok {
		Values[servicename] = make(map[string]interface{})
	}

	for k, v := range values {
		Values[servicename][k] = v
	}
}

// AddVolumeValues add a volume to the values.yaml map for the given deployment name.
func AddVolumeValues(deployment string, volname string, values map[string]EnvVal) {
	locker.Lock()
	defer locker.Unlock()

	if _, ok := VolumeValues[deployment]; !ok {
		VolumeValues[deployment] = make(map[string]map[string]EnvVal)
	}
	VolumeValues[deployment][volname] = values
}

func readEnvFile(envfilename string) map[string]EnvVal {
	env := make(map[string]EnvVal)
	content, err := ioutil.ReadFile(envfilename)
	if err != nil {
		logger.ActivateColors = true
		logger.Red(err.Error())
		logger.ActivateColors = false
		os.Exit(2)
	}
	// each value is on a separate line with KEY=value
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, "=") {
			kv := strings.SplitN(line, "=", 2)
			env[kv[0]] = kv[1]
		}
	}
	return env
}

// applyEnvMapLabel will get all LABEL_MAP_ENV to rebuild the env map with tpl.
func applyEnvMapLabel(s *types.ServiceConfig, c *helm.Container) {

	locker.Lock()
	defer locker.Unlock()
	mapenv, ok := s.Labels[helm.LABEL_MAP_ENV]
	if !ok {
		return
	}

	// the mapenv is a YAML string
	var envmap map[string]EnvVal
	err := yaml.Unmarshal([]byte(mapenv), &envmap)
	if err != nil {
		logger.ActivateColors = true
		logger.Red(err.Error())
		logger.ActivateColors = false
		return
	}

	// add in envmap
	for k, v := range envmap {
		vstring := fmt.Sprintf("%v", v)
		s.Environment[k] = &vstring
		touched := false
		if c.Env != nil {
			c.Env = make([]*helm.Value, 0)
		}
		for _, env := range c.Env {
			if env.Name == k {
				env.Value = v
				touched = true
			}
		}
		if !touched {
			c.Env = append(c.Env, &helm.Value{Name: k, Value: v})
		}
	}
}

// setEnvToValues will set the environment variables to the values.yaml map.
func setEnvToValues(name string, s *types.ServiceConfig, c *helm.Container) {
	// crete the "environment" key

	env := make(map[string]EnvVal)
	for k, v := range s.Environment {
		env[k] = v
	}
	if len(env) == 0 {
		return
	}

	valuesEnv := make(map[string]interface{})
	for k, v := range env {
		k = strings.ReplaceAll(k, ".", "_")
		valuesEnv[k] = v
	}

	AddValues(name, map[string]EnvVal{"environment": valuesEnv})
	for k := range env {
		fixedK := strings.ReplaceAll(k, ".", "_")
		v := "{{ tpl .Values." + name + ".environment." + fixedK + " . }}"
		s.Environment[k] = &v
		touched := false
		for _, c := range c.Env {
			if c.Name == k {
				c.Value = v
				touched = true
			}
		}
		if !touched {
			c.Env = append(c.Env, &helm.Value{Name: k, Value: v})
		}
	}
}

func setSecretVar(name string, s *types.ServiceConfig, c *helm.Container) *helm.Secret {
	locker.Lock()
	defer locker.Unlock()
	// get the list of secret vars
	secretvars, ok := s.Labels[helm.LABEL_SECRETVARS]
	if !ok {
		return nil
	}

	store := helm.NewSecret(name, "")
	for _, secretvar := range strings.Split(secretvars, ",") {
		secretvar = strings.TrimSpace(secretvar)
		// get the value from env
		_, ok := s.Environment[secretvar]
		if !ok {
			continue
		}
		// add the secret
		store.AddEnv(secretvar, ".Values."+name+".environment."+secretvar)
		for i, env := range c.Env {
			if env.Name == secretvar {
				c.Env = append(c.Env[:i], c.Env[i+1:]...)
				i--
			}
		}
		// remove env from ServiceConfig
		delete(s.Environment, secretvar)
	}
	return store
}

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

	return container
}

// addVolumeFrom takes the LABEL_VOLUMEFROM to get volumes from another container. This can only work with
// container that has got LABEL_SAMEPOD as we need to get the volumes from another container in the same deployment.
func addVolumeFrom(deployment *helm.Deployment, container *helm.Container, s *types.ServiceConfig) {
	labelfrom, ok := s.Labels[helm.LABEL_VOLUMEFROM]
	if !ok {
		return
	}

	// decode Yaml from the label
	var volumesFrom map[string]map[string]string
	err := yaml.Unmarshal([]byte(labelfrom), &volumesFrom)
	if err != nil {
		logger.ActivateColors = true
		logger.Red(err.Error())
		logger.ActivateColors = false
		return
	}

	// for each declared volume "from", we will find it from the deployment volumes and add it to the container.
	// Then, to avoid duplicates, we will remove it from the ServiceConfig object.
	for name, volumes := range volumesFrom {
		for volumeName := range volumes {
			initianame := volumeName
			volumeName = tools.PathToName(volumeName)
			// get the volume from the deployment container "name"
			var ctn *helm.Container
			for _, c := range deployment.Spec.Template.Spec.Containers {
				if c.Name == name {
					ctn = c
					break
				}
			}
			if ctn == nil {
				logger.ActivateColors = true
				logger.Redf("VolumeFrom: container %s not found", name)
				logger.ActivateColors = false
				continue
			}
			// get the volume from the container
			for _, v := range ctn.VolumeMounts {
				switch v := v.(type) {
				case map[string]interface{}:
					if v["name"] == volumeName {
						if container.VolumeMounts == nil {
							container.VolumeMounts = make([]interface{}, 0)
						}
						// make a copy of the volume mount and then add it to the VolumeMounts
						var mountpoint = make(map[string]interface{})
						for k, v := range v {
							mountpoint[k] = v
						}
						container.VolumeMounts = append(container.VolumeMounts, mountpoint)

						// remove the volume from the ServiceConfig
						for i, vol := range s.Volumes {
							if vol.Source == initianame {
								s.Volumes = append(s.Volumes[:i], s.Volumes[i+1:]...)
								i--
								break
							}
						}
					}
				}
			}
		}
	}
}
