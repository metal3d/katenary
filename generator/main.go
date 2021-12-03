package generator

import (
	"fmt"
	"io/ioutil"
	"katenary/compose"
	"katenary/helm"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"errors"
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

// Values is kept in memory to create a values.yaml file.
var Values = make(map[string]map[string]interface{})
var VolumeValues = make(map[string]map[string]map[string]interface{})

var dependScript = `
OK=0
echo "Checking __service__ port"
while [ $OK != 1 ]; do
    echo -n "."
    nc -z {{ .Release.Name }}-__service__ __port__ && OK=1
    sleep 1
done
echo
echo "Done"
`

// Create a Deployment for a given compose.Service. It returns a list of objects: a Deployment and a possible Service (kubernetes represnetation as maps).
func CreateReplicaObject(name string, s *compose.Service) chan interface{} {
	ret := make(chan interface{}, len(s.Ports)+len(s.Expose)+1)
	go parseService(name, s, ret)
	return ret
}

// This function will try to yied deployment and services based on a service from the compose file structure.
func parseService(name string, s *compose.Service, ret chan interface{}) {
	Magenta(ICON_PACKAGE+" Generating deployment for ", name)

	o := helm.NewDeployment(name)
	container := helm.NewContainer(name, s.Image, s.Environment, s.Labels)

	// prepare secrets
	secretsFiles := make([]string, 0)
	if v, ok := s.Labels[helm.LABEL_ENV_SECRET]; ok {
		secretsFiles = strings.Split(v, ",")
	}

	// manage environment files (env_file in compose)
	for _, envfile := range s.EnvFiles {
		f := strings.ReplaceAll(envfile, "_", "-")
		f = strings.ReplaceAll(f, ".env", "")
		f = strings.ReplaceAll(f, ".", "-")
		cf := f + "-" + name
		isSecret := false
		for _, s := range secretsFiles {
			if s == envfile {
				isSecret = true
			}
		}
		var store helm.InlineConfig
		if !isSecret {
			Bluef(ICON_CONF+" Generating configMap %s\n", cf)
			store = helm.NewConfigMap(cf)
		} else {
			Bluef(ICON_SECRET+" Generating secret %s\n", cf)
			store = helm.NewSecret(cf)
		}
		if err := store.AddEnvFile(envfile); err != nil {
			ActivateColors = true
			Red(err.Error())
			ActivateColors = false
			os.Exit(2)
		}
		container.EnvFrom = append(container.EnvFrom, map[string]map[string]string{
			"configMapRef": {
				"name": store.Metadata().Name,
			},
		})

		ret <- store
	}

	// check the image, and make it "variable" in values.yaml
	container.Image = "{{ .Values." + name + ".image }}"
	Values[name] = map[string]interface{}{
		"image": s.Image,
	}

	// manage ports
	exists := make(map[int]string)
	for _, port := range s.Ports {
		_p := strings.Split(port, ":")
		port = _p[0]
		if len(_p) > 1 {
			port = _p[1]
		}
		portNumber, _ := strconv.Atoi(port)
		portName := name
		for _, n := range exists {
			if name == n {
				portName = fmt.Sprintf("%s-%d", name, portNumber)
			}
		}
		container.Ports = append(container.Ports, &helm.ContainerPort{
			Name:          portName,
			ContainerPort: portNumber,
		})
		exists[portNumber] = name
	}

	// manage the "expose" section to be a NodePort in Kubernetes
	for _, port := range s.Expose {
		if _, exist := exists[port]; exist {
			continue
		}
		container.Ports = append(container.Ports, &helm.ContainerPort{
			Name:          name,
			ContainerPort: port,
		})
	}

	// Prepare volumes
	volumes := make([]map[string]interface{}, 0)
	mountPoints := make([]interface{}, 0)
	configMapsVolumes := make([]string, 0)
	if v, ok := s.Labels[helm.LABEL_VOL_CM]; ok {
		configMapsVolumes = strings.Split(v, ",")
	}
	for _, volume := range s.Volumes {
		parts := strings.Split(volume, ":")
		volname := parts[0]
		volepath := parts[1]

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
			ActivateColors = true
			Redf("You cannot, at this time, have local volume in %s deployment\n", name)
			ActivateColors = false
			continue
		}
		if isCM {
			// the volume is a path and it's explicitally asked to be a configmap in labels
			cm := buildCMFromPath(volname)
			volname = strings.Replace(volname, "./", "", 1)
			volname = strings.ReplaceAll(volname, ".", "-")
			cm.K8sBase.Metadata.Name = "{{ .Release.Name }}-" + volname + "-" + name
			// build a configmap from the volume path
			volumes = append(volumes, map[string]interface{}{
				"name": volname,
				"configMap": map[string]string{
					"name": cm.K8sBase.Metadata.Name,
				},
			})
			mountPoints = append(mountPoints, map[string]interface{}{
				"name":      volname,
				"mountPath": volepath,
			})
			ret <- cm
		} else {

			pvc := helm.NewPVC(name, volname)
			volumes = append(volumes, map[string]interface{}{
				"name": volname,
				"persistentVolumeClaim": map[string]string{
					"claimName": "{{ .Release.Name }}-" + volname,
				},
			})
			mountPoints = append(mountPoints, map[string]interface{}{
				"name":      volname,
				"mountPath": volepath,
			})

			Yellow(ICON_STORE+" Generate volume values for ", volname, " in deployment ", name)
			locker.Lock()
			if _, ok := VolumeValues[name]; !ok {
				VolumeValues[name] = make(map[string]map[string]interface{})
			}
			VolumeValues[name][volname] = map[string]interface{}{
				"enabled":  false,
				"capacity": "1Gi",
			}
			locker.Unlock()
			ret <- pvc
		}
	}
	container.VolumeMounts = mountPoints

	o.Spec.Template.Spec.Volumes = volumes
	o.Spec.Template.Spec.Containers = []*helm.Container{container}

	// Add some labels
	o.Spec.Selector = map[string]interface{}{
		"matchLabels": buildSelector(name, s),
	}
	o.Spec.Template.Metadata.Labels = buildSelector(name, s)

	// Now, for "depends_on" section, it's a bit tricky...
	// We need to detect "others" services, but we probably not have parsed them yet, so
	// we will wait for them for a while.
	initContainers := make([]*helm.Container, 0)
	for _, dp := range s.DependsOn {
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
	o.Spec.Template.Spec.InitContainers = initContainers

	// Then, create services for "ports" and "expose" section
	if len(s.Ports) > 0 || len(s.Expose) > 0 {
		for _, s := range createService(name, s) {
			ret <- s
		}
	}

	// Special case, it there is no "ports", so there is no associated services...
	// But... some other deployment can wait for it, so we alert that this deployment hasn't got any
	// associated service.
	if len(s.Ports) == 0 {
		locker.Lock()
		// alert any current or **futur** waiters that this service is not exposed
		go func() {
			for {
				select {
				case <-time.Tick(1 * time.Millisecond):
					for _, c := range serviceWaiters[name] {
						c <- -1
						close(c)
					}
				}
			}
		}()
		locker.Unlock()
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

// Create a service (k8s).
func createService(name string, s *compose.Service) []interface{} {

	ret := make([]interface{}, 0)
	Magenta(ICON_SERVICE+" Generating service for ", name)
	ks := helm.NewService(name)

	for i, p := range s.Ports {
		port := strings.Split(p, ":")
		src, _ := strconv.Atoi(port[0])
		target := src
		if len(port) > 1 {
			target, _ = strconv.Atoi(port[1])
		}
		ks.Spec.Ports = append(ks.Spec.Ports, helm.NewServicePort(target, target))
		if i == 0 {
			detected(name, target)
		}
	}
	ks.Spec.Selector = buildSelector(name, s)

	ret = append(ret, ks)
	if v, ok := s.Labels[helm.LABEL_INGRESS]; ok {
		port, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("The given port \"%v\" as ingress port in %s service is not an integer\n", v, name)
		}
		Cyanf(ICON_INGRESS+" Create an ingress for port %d on %s service\n", port, name)
		ing := createIngress(name, port, s)
		ret = append(ret, ing)
	}

	if len(s.Expose) > 0 {
		Magenta(ICON_SERVICE+" Generating service for ", name+"-external")
		ks := helm.NewService(name + "-external")
		ks.Spec.Type = "NodePort"
		for _, p := range s.Expose {
			ks.Spec.Ports = append(ks.Spec.Ports, helm.NewServicePort(p, p))
		}
		ks.Spec.Selector = buildSelector(name, s)
		ret = append(ret, ks)
	}

	return ret
}

// Create an ingress.
func createIngress(name string, port int, s *compose.Service) *helm.Ingress {
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
					Backend: helm.IngressBackend{
						Service: helm.IngressService{
							Name: "{{ .Release.Name }}-" + name,
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
	servicesMap[name] = port
	go func() {
		cx := serviceWaiters[name]
		for _, c := range cx {
			if v, ok := servicesMap[name]; ok {
				c <- v
				//close(c)
			}
		}
	}()
	locker.Unlock()
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
	c := make(chan int, 0)
	serviceWaiters[name] = append(serviceWaiters[name], c)
	go func() {
		if v, ok := servicesMap[name]; ok {
			c <- v
			//close(c)
		}
	}()
	locker.Unlock()
	return c
}

func buildSelector(name string, s *compose.Service) map[string]string {
	return map[string]string{
		"katenary.io/component": name,
		"katenary.io/release":   "{{ .Release.Name }}",
	}
}

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
					ActivateColors = true
					Yellowf("Warning, %s is a directory, at this time we only "+
						"can create configmap for first level file list\n", f)
					ActivateColors = false
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
