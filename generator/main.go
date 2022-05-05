package generator

import (
	"fmt"
	"io/ioutil"
	"katenary/compose"
	"katenary/helm"
	"katenary/logger"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

const (
	ICON_PACKAGE = "📦"
	ICON_SERVICE = "🔌"
	ICON_SECRET  = "🔏"
	ICON_CONF    = "📝"
	ICON_STORE   = "⚡"
	ICON_INGRESS = "🌐"
)

// Values is kept in memory to create a values.yaml file.
var (
	Values         = make(map[string]map[string]interface{})
	VolumeValues   = make(map[string]map[string]map[string]interface{})
	EmptyDirs      = []string{}
	servicesMap    = make(map[string]int)
	serviceWaiters = make(map[string][]chan int)
	locker         = &sync.Mutex{}

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

// Create a Deployment for a given compose.Service. It returns a list of objects: a Deployment and a possible Service (kubernetes represnetation as maps).
func CreateReplicaObject(name string, s types.ServiceConfig, linked map[string]types.ServiceConfig) chan interface{} {
	ret := make(chan interface{}, len(s.Ports)+len(s.Expose)+2)
	go parseService(name, s, linked, ret)
	return ret
}

// This function will try to yied deployment and services based on a service from the compose file structure.
func parseService(name string, s types.ServiceConfig, linked map[string]types.ServiceConfig, ret chan interface{}) {
	logger.Magenta(ICON_PACKAGE+" Generating deployment for ", name)

	// adapt env
	applyEnvMapLabel(&s)
	setEnvToValues(name, &s)

	deployment := helm.NewDeployment(name)
	container := helm.NewContainer(name, s.Image, s.Environment, s.Labels)
	prepareContainer(container, s, name)
	prepareEnvFromFiles(name, s, container, ret)

	// Set the containers to the deployment
	deployment.Spec.Template.Spec.Containers = []*helm.Container{container}

	// Prepare volumes
	madePVC := make(map[string]bool)
	deployment.Spec.Template.Spec.Volumes = prepareVolumes(name, name, s, container, madePVC, ret)

	// Now, for "depends_on" section, it's a bit tricky to get dependencies, see the function below.
	deployment.Spec.Template.Spec.InitContainers = prepareInitContainers(name, s, container)

	// Add selectors
	selectors := buildSelector(name, s)
	deployment.Spec.Selector = map[string]interface{}{
		"matchLabels": selectors,
	}
	deployment.Spec.Template.Metadata.Labels = selectors

	// Now, the linked services
	for lname, link := range linked {
		applyEnvMapLabel(&link)
		setEnvToValues(lname, &link)
		container := helm.NewContainer(lname, link.Image, link.Environment, link.Labels)
		prepareContainer(container, link, lname)
		prepareEnvFromFiles(lname, link, container, ret)
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, container)
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, prepareVolumes(name, lname, link, container, madePVC, ret)...)
		deployment.Spec.Template.Spec.InitContainers = append(deployment.Spec.Template.Spec.InitContainers, prepareInitContainers(lname, link, container)...)
		//append ports and expose ports to the deployment, to be able to generate them in the Service file
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
			ret <- s
		}
	}

	// add the volumes in Values
	if len(VolumeValues[name]) > 0 {
		AddValues(name, map[string]interface{}{"persistence": VolumeValues[name]})
	}

	// the deployment is ready, give it
	ret <- deployment

	// and then, we can say that it's the end
	ret <- nil
}

// prepareContainer assigns image, command, env, and labels to a container.
func prepareContainer(container *helm.Container, service types.ServiceConfig, servicename string) {
	// if there is no image name, this should fail!
	if service.Image == "" {
		log.Fatal(ICON_PACKAGE+" No image name for service ", servicename)
	}
	container.Image = "{{ .Values." + servicename + ".image }}"
	container.Command = service.Command
	AddValues(servicename, map[string]interface{}{"image": service.Image})
	prepareProbes(servicename, service, container)
	generateContainerPorts(service, servicename, container)
}

// Create a service (k8s).
func generateServicesAndIngresses(name string, s types.ServiceConfig) []interface{} {

	ret := make([]interface{}, 0) // can handle helm.Service or helm.Ingress
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
func createIngress(name string, port int, s types.ServiceConfig) *helm.Ingress {
	ingress := helm.NewIngress(name)

	ingressVal := map[string]interface{}{
		"class":   "nginx",
		"host":    name + "." + helm.Appname + ".tld",
		"enabled": false,
	}
	AddValues(name, map[string]interface{}{"ingress": ingressVal})

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
func buildSelector(name string, s types.ServiceConfig) map[string]string {
	return map[string]string{
		"katenary.io/component": name,
		"katenary.io/release":   helm.ReleaseNameTpl,
	}
}

// buildCMFromPath generates a ConfigMap from a path.
func buildCMFromPath(path string) *helm.ConfigMap {
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
	}

	cm := helm.NewConfigMap("")
	cm.Data = files
	return cm
}

// generateContainerPorts add the container ports of a service.
func generateContainerPorts(s types.ServiceConfig, name string, container *helm.Container) {

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
func prepareVolumes(deployment, name string, s types.ServiceConfig, container *helm.Container, madePVC map[string]bool, ret chan interface{}) []map[string]interface{} {

	volumes := make([]map[string]interface{}, 0)
	mountPoints := make([]interface{}, 0)
	configMapsVolumes := make([]string, 0)
	if v, ok := s.Labels[helm.LABEL_VOL_CM]; ok {
		configMapsVolumes = strings.Split(v, ",")
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

		isCM := false
		for _, cmVol := range configMapsVolumes {
			cmVol = strings.TrimSpace(cmVol)
			if volname == cmVol {
				isCM = true
				break
			}
		}

		if !isCM && (strings.HasPrefix(volname, ".") || strings.HasPrefix(volname, "/")) {
			// local volume cannt be mounted
			logger.ActivateColors = true
			logger.Redf("You cannot, at this time, have local volume in %s deployment\n", name)
			logger.ActivateColors = false
			continue
		}
		if isCM {
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
				volname = filepath.Dir(volname)
			}

			// the volume is a path and it's explicitally asked to be a configmap in labels
			cm := buildCMFromPath(volname)
			volname = strings.Replace(volname, "./", "", 1)
			volname = strings.ReplaceAll(volname, "/", "-")
			volname = strings.ReplaceAll(volname, ".", "-")
			cm.K8sBase.Metadata.Name = helm.ReleaseNameTpl + "-" + volname + "-" + name

			// build a configmap from the volume path
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
			ret <- cm
		} else {
			// rmove minus sign from volume name
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
					container.VolumeMounts = mountPoints
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
			AddVolumeValues(deployment, volname, map[string]interface{}{
				"enabled":  false,
				"capacity": "1Gi",
			})

			if _, ok := madePVC[deployment+volname]; !ok {
				madePVC[deployment+volname] = true
				pvc := helm.NewPVC(deployment, volname)
				ret <- pvc
			}
		}
	}
	container.VolumeMounts = mountPoints
	return volumes
}

// prepareInitContainers add the init containers of a service.
func prepareInitContainers(name string, s types.ServiceConfig, container *helm.Container) []*helm.Container {

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
func prepareProbes(name string, s types.ServiceConfig, container *helm.Container) {
	// first, check if there a label for the probe
	if check, ok := s.Labels[helm.LABEL_HEALTHCHECK]; ok {
		check = strings.TrimSpace(check)
		p := helm.NewProbeFromService(&s)
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

func buildCommandProbe(s types.ServiceConfig) *helm.Probe {

	// Get the first element of the command from ServiceConfig
	first := s.HealthCheck.Test[0]

	p := helm.NewProbeFromService(&s)
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
func prepareEnvFromFiles(name string, s types.ServiceConfig, container *helm.Container, ret chan interface{}) {

	// prepare secrets
	secretsFiles := make([]string, 0)
	if v, ok := s.Labels[helm.LABEL_ENV_SECRET]; ok {
		secretsFiles = strings.Split(v, ",")
	}

	// manage environment files (env_file in compose)
	for _, envfile := range s.EnvFile {
		f := strings.ReplaceAll(envfile, "_", "-")
		f = strings.ReplaceAll(f, ".env", "")
		f = strings.ReplaceAll(f, ".", "")
		f = strings.ReplaceAll(f, "/", "")
		cf := f + "-" + name
		isSecret := false
		for _, s := range secretsFiles {
			if s == envfile {
				isSecret = true
			}
		}
		var store helm.InlineConfig
		if !isSecret {
			logger.Bluef(ICON_CONF+" Generating configMap %s\n", cf)
			store = helm.NewConfigMap(cf)
		} else {
			logger.Bluef(ICON_SECRET+" Generating secret %s\n", cf)
			store = helm.NewSecret(cf)
		}

		envfile = filepath.Join(compose.GetCurrentDir(), envfile)
		if err := store.AddEnvFile(envfile); err != nil {
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
					}
				}
			}
		}

		ret <- store
	}
}

// AddValues adds values to the values.yaml map.
func AddValues(servicename string, values map[string]interface{}) {
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
func AddVolumeValues(deployment string, volname string, values map[string]interface{}) {
	locker.Lock()
	defer locker.Unlock()

	if _, ok := VolumeValues[deployment]; !ok {
		VolumeValues[deployment] = make(map[string]map[string]interface{})
	}
	VolumeValues[deployment][volname] = values
}

func readEnvFile(envfilename string) map[string]string {
	env := make(map[string]string)
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
func applyEnvMapLabel(s *types.ServiceConfig) {
	mapenv, ok := s.Labels[helm.LABEL_MAP_ENV]
	if !ok {
		return
	}

	// the mapenv is a YAML string
	var envmap map[string]string
	err := yaml.Unmarshal([]byte(mapenv), &envmap)
	if err != nil {
		logger.ActivateColors = true
		logger.Red(err.Error())
		logger.ActivateColors = false
		return
	}

	// add in envmap
	for k, v := range envmap {
		s.Environment[k] = &v
	}
}

// setEnvToValues will set the environment variables to the values.yaml map.
func setEnvToValues(name string, s *types.ServiceConfig) {
	// crete the "environment" key
	env := make(map[string]interface{})
	for k, v := range s.Environment {
		env[k] = v
	}

	AddValues(name, map[string]interface{}{"environment": env})
	for k := range s.Environment {
		v := "{{ .Values." + name + ".environment." + k + " }}"
		s.Environment[k] = &v
	}
}
