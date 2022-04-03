package generator

import (
	"fmt"
	"io/ioutil"
	"katenary/helm"
	"katenary/logger"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"errors"

	"github.com/compose-spec/compose-go/types"
)

var servicesMap = make(map[string]int)
var serviceWaiters = make(map[string][]chan int)
var locker = &sync.Mutex{}

const (
	ICON_PACKAGE = "📦"
	ICON_SERVICE = "🔌"
	ICON_SECRET  = "🔏"
	ICON_CONF    = "📝"
	ICON_STORE   = "⚡"
	ICON_INGRESS = "🌐"
)

const (
	RELEASE_NAME = helm.RELEASE_NAME
)

// Values is kept in memory to create a values.yaml file.
var Values = make(map[string]map[string]interface{})
var VolumeValues = make(map[string]map[string]map[string]interface{})
var EmptyDirs = []string{}

var dependScript = `
OK=0
echo "Checking __service__ port"
while [ $OK != 1 ]; do
    echo -n "."
    nc -z ` + RELEASE_NAME + `-__service__ __port__ 2>&1 >/dev/null && OK=1 || sleep 1
done
echo
echo "Done"
`

var madeDeployments = make(map[string]helm.Deployment, 0)

// Create a Deployment for a given compose.Service. It returns a list of objects: a Deployment and a possible Service (kubernetes represnetation as maps).
func CreateReplicaObject(name string, s types.ServiceConfig, linked map[string]types.ServiceConfig) chan interface{} {
	ret := make(chan interface{}, len(s.Ports)+len(s.Expose)+1)
	go parseService(name, s, linked, ret)
	return ret
}

// This function will try to yied deployment and services based on a service from the compose file structure.
func parseService(name string, s types.ServiceConfig, linked map[string]types.ServiceConfig, ret chan interface{}) {
	logger.Magenta(ICON_PACKAGE+" Generating deployment for ", name)

	o := helm.NewDeployment(name)

	container := helm.NewContainer(name, s.Image, s.Environment, s.Labels)
	prepareContainer(container, s, name)
	prepareEnvFromFiles(name, s, container, ret)

	// Set the container to the deployment
	o.Spec.Template.Spec.Containers = []*helm.Container{container}

	// Prepare volumes
	madePVC := make(map[string]bool)
	o.Spec.Template.Spec.Volumes = prepareVolumes(name, name, s, container, madePVC, ret)

	// Now, for "depends_on" section, it's a bit tricky to get dependencies, see the function below.
	o.Spec.Template.Spec.InitContainers = prepareInitContainers(name, s, container)

	// Add selectors
	selectors := buildSelector(name, s)
	o.Spec.Selector = map[string]interface{}{
		"matchLabels": selectors,
	}
	o.Spec.Template.Metadata.Labels = selectors

	// Now, the linked services
	for lname, link := range linked {
		container := helm.NewContainer(lname, link.Image, link.Environment, link.Labels)
		prepareContainer(container, link, lname)
		prepareEnvFromFiles(lname, link, container, ret)
		o.Spec.Template.Spec.Containers = append(o.Spec.Template.Spec.Containers, container)
		o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, prepareVolumes(name, lname, link, container, madePVC, ret)...)
		o.Spec.Template.Spec.InitContainers = append(o.Spec.Template.Spec.InitContainers, prepareInitContainers(lname, link, container)...)
		//append ports and expose ports to the deployment, to be able to generate them in the Service file
		if len(link.Ports) > 0 || len(link.Expose) > 0 {
			s.Ports = append(s.Ports, link.Ports...)
			s.Expose = append(s.Expose, link.Expose...)
		}
	}

	// Remove duplicates in volumes
	volumes := make([]map[string]interface{}, 0)
	done := make(map[string]bool)
	for _, vol := range o.Spec.Template.Spec.Volumes {
		name := vol["name"].(string)
		if _, ok := done[name]; ok {
			continue
		} else {
			done[name] = true
			volumes = append(volumes, vol)
		}
	}
	o.Spec.Template.Spec.Volumes = volumes

	// Then, create Services and possible Ingresses for ingress labels, "ports" and "expose" section
	if len(s.Ports) > 0 || len(s.Expose) > 0 {
		for _, s := range generateServicesAndIngresses(name, s) {
			ret <- s
		}
	}

	// Special case, it there is no "ports", so there is no associated services...
	// But... some other deployment can wait for it, so we alert that this deployment hasn't got any
	// associated service.
	if len(s.Ports) == 0 {
		// alert any current or **future** waiters that this service is not exposed
		go func() {
			defer func() {
				// recover from panic
				if r := recover(); r != nil {
					// log the stack trace
					fmt.Println(r)
				}
			}()
			for {
				select {
				case <-time.Tick(1 * time.Millisecond):
					locker.Lock()
					for _, c := range serviceWaiters[name] {
						c <- -1
						close(c)
					}
					locker.Unlock()
				}
			}
		}()
	}

	// add the volumes in Values
	if len(VolumeValues[name]) > 0 {
		locker.Lock()
		Values[name]["persistence"] = VolumeValues[name]
		locker.Unlock()
	}

	// the deployment is ready, give it
	ret <- o

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
	Values[servicename] = map[string]interface{}{
		"image": service.Image,
	}
	prepareProbes(servicename, service, container)
	generateContainerPorts(service, servicename, container)
}

// Create a service (k8s).
func generateServicesAndIngresses(name string, s types.ServiceConfig) []interface{} {

	ret := make([]interface{}, 0) // can handle helm.Service or helm.Ingress
	logger.Magenta(ICON_SERVICE+" Generating service for ", name)
	ks := helm.NewService(name)

	for i, p := range s.Ports {
		ks.Spec.Ports = append(ks.Spec.Ports, helm.NewServicePort(int(p.Target), int(p.Target)))
		if i == 0 {
			detected(name, int(p.Target))
		}
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
	Values[name]["ingress"] = map[string]interface{}{
		"class":   "nginx",
		"host":    name + "." + helm.Appname + ".tld",
		"enabled": false,
	}
	ingress.Spec.Rules = []helm.IngressRule{
		{
			Host: fmt.Sprintf("{{ .Values.%s.ingress.host }}", name),
			Http: helm.IngressHttp{
				Paths: []helm.IngressPath{{
					Path:     "/",
					PathType: "Prefix",
					Backend: &helm.IngressBackend{
						Service: helm.IngressService{
							Name: RELEASE_NAME + "-" + name,
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

// This function is called when a possible service is detected, it append the port in a map to make others
// to be able to get the service name. It also try to send the data to any "waiter" for this service.
func detected(name string, port int) {
	locker.Lock()
	defer locker.Unlock()
	if _, ok := servicesMap[name]; ok {
		return
	}
	servicesMap[name] = port
	go func() {
		locker.Lock()
		defer locker.Unlock()
		if cx, ok := serviceWaiters[name]; ok {
			for _, c := range cx {
				c <- port
			}
		}
	}()
}

func getPort(name string) (int, error) {
	if v, ok := servicesMap[name]; ok {
		return v, nil
	}
	return -1, errors.New("Not found")
}

// Waits for a service to be discovered. Sometimes, a deployment depends on another one. See the detected() function.
func waitPort(name string) chan int {
	locker.Lock()
	defer locker.Unlock()
	c := make(chan int, 0)
	serviceWaiters[name] = append(serviceWaiters[name], c)
	go func() {
		locker.Lock()
		defer locker.Unlock()
		if v, ok := servicesMap[name]; ok {
			c <- v
		}
	}()
	return c
}

// Build the selector for the service.
func buildSelector(name string, s types.ServiceConfig) map[string]string {
	return map[string]string{
		"katenary.io/component": name,
		"katenary.io/release":   RELEASE_NAME,
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
			stat, _ := os.Stat(volname)
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
			cm.K8sBase.Metadata.Name = RELEASE_NAME + "-" + volname + "-" + name

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
					"claimName": RELEASE_NAME + "-" + volname,
				},
			})
			mountPoints = append(mountPoints, map[string]interface{}{
				"name":      volname,
				"mountPath": volepath,
			})

			logger.Yellow(ICON_STORE+" Generate volume values", volname, "for container named", name, "in deployment", deployment)
			locker.Lock()
			if _, ok := VolumeValues[deployment]; !ok {
				VolumeValues[deployment] = make(map[string]map[string]interface{})
			}
			VolumeValues[deployment][volname] = map[string]interface{}{
				"enabled":  false,
				"capacity": "1Gi",
			}
			locker.Unlock()

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
	for dp, _ := range s.DependsOn {
		c := helm.NewContainer("check-"+dp, "busybox", nil, s.Labels)
		command := strings.ReplaceAll(strings.TrimSpace(dependScript), "__service__", dp)

		foundPort := -1
		if defaultPort, err := getPort(dp); err != nil {
			// BUG: Sometimes the chan remains opened
			foundPort = <-waitPort(dp)
		} else {
			foundPort = defaultPort
		}
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

		ret <- store
	}
}
