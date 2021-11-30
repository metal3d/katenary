package generator

import (
	"fmt"
	"helm-compose/compose"
	"helm-compose/helm"
	"os"
	"strconv"
	"strings"
	"sync"

	"errors"
)

var servicesMap = make(map[string]int)
var serviceWaiters = make(map[string][]chan int)
var locker = &sync.Mutex{}
var serviceTick = make(chan int, 0)

// Ingresses is kept in memory to create ingresses.
var Ingresses = make(map[string]*helm.Ingress, 0)

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
func CreateReplicaObject(name string, s compose.Service) (ret []interface{}) {

	Magenta("Generating deployment for ", name)
	o := helm.NewDeployment()
	ret = append(ret, o)
	o.Metadata.Name = "{{ .Release.Name }}-" + name

	container := helm.NewContainer(name, s.Image, s.Environment, s.Labels)

	container.Image = "{{ .Values." + name + ".image }}"
	Values[name] = map[string]interface{}{
		"image": s.Image,
	}

	exists := make(map[int]string)
	for _, port := range s.Ports {
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
	for _, port := range s.Expose {
		if _, exist := exists[port]; exist {
			continue
		}
		container.Ports = append(container.Ports, &helm.ContainerPort{
			Name:          name,
			ContainerPort: port,
		})
	}

	volumes := make([]map[string]interface{}, 0)
	mountPoints := make([]interface{}, 0)
	for _, volume := range s.Volumes {
		parts := strings.Split(volume, ":")
		volname := parts[0]
		volepath := parts[1]
		if strings.HasPrefix(volname, ".") || strings.HasPrefix(volname, "/") {
			Redf("You cannot, at this time, have local volume in %s service", name)
			os.Exit(1)
		}

		pvc := helm.NewPVC(name, volname)
		ret = append(ret, pvc)
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

		Yellow("Generate volume values for ", volname)
		locker.Lock()
		if _, ok := VolumeValues[name]; !ok {
			VolumeValues[name] = make(map[string]map[string]interface{})
		}
		VolumeValues[name][volname] = map[string]interface{}{
			"enabled":  false,
			"capacity": "1Gi",
		}
		locker.Unlock()
	}
	container.VolumeMounts = mountPoints

	o.Spec.Template.Spec.Volumes = volumes
	o.Spec.Template.Spec.Containers = []*helm.Container{container}

	o.Spec.Selector = map[string]interface{}{
		"matchLabels": buildSelector(name, s),
	}

	o.Spec.Template.Metadata.Labels = buildSelector(name, s)

	wait := &sync.WaitGroup{}
	initContainers := make([]*helm.Container, 0)
	for _, dp := range s.DependsOn {
		if len(s.Ports) == 0 && len(s.Expose) == 0 {
			Redf("No port exposed for %s that is in dependency", name)
			os.Exit(1)
		}
		c := helm.NewContainer("check-"+dp, "busybox", nil, s.Labels)
		command := strings.ReplaceAll(strings.TrimSpace(dependScript), "__service__", dp)

		wait.Add(1)
		go func(dp string) {
			defer wait.Done()
			p := -1
			if defaultPort, err := getPort(dp); err != nil {
				p = <-waitPort(dp)
			} else {
				p = defaultPort
			}
			command = strings.ReplaceAll(command, "__port__", strconv.Itoa(p))

			c.Command = []string{
				"sh",
				"-c",
				command,
			}
			initContainers = append(initContainers, c)
		}(dp)
	}
	wait.Wait()
	o.Spec.Template.Spec.InitContainers = initContainers

	if len(s.Ports) > 0 || len(s.Expose) > 0 {
		ks := createService(name, s)
		ret = append(ret, ks)
	}

	if len(VolumeValues[name]) > 0 {
		Values[name]["persistence"] = VolumeValues[name]
	}

	Green("Done deployment ", name)

	return
}

// Create a service (k8s).
func createService(name string, s compose.Service) *helm.Service {

	Magenta("Generating service for ", name)
	ks := helm.NewService()
	ks.Metadata.Name = "{{ .Release.Name }}-" + name
	defaultPort := 0
	names := make(map[int]int)

	for i, p := range s.Ports {
		port := strings.Split(p, ":")
		src, _ := strconv.Atoi(port[0])
		target := src
		if len(port) > 1 {
			target, _ = strconv.Atoi(port[1])
		}
		ks.Spec.Ports = append(ks.Spec.Ports, helm.NewServicePort(src, target))
		names[target] = 1
		if i == 0 {
			defaultPort = target
			detected(name, target)
		}
	}
	for i, p := range s.Expose {
		if _, ok := names[p]; ok {
			continue
		}
		ks.Spec.Ports = append(ks.Spec.Ports, helm.NewServicePort(p, p))
		if i == 0 {
			defaultPort = p
			detected(name, p)
		}
	}

	ks.Spec.Selector = buildSelector(name, s)

	if v, ok := s.Labels[helm.K+"/expose-ingress"]; ok && v == "true" {
		createIngress(name, defaultPort, s)
	}

	Green("Done service ", name)
	return ks
}

// Create an ingress.
func createIngress(name string, port int, s compose.Service) {
	ingress := helm.NewIngress(name)
	Values[name]["ingress"] = map[string]interface{}{
		"class":   "nginx",
		"host":    "chart.example.tld",
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

	locker.Lock()
	Ingresses[name] = ingress
	locker.Unlock()
}

// This function is called when a possible service is detected, it append the port in a map to make others to be able to get the service name. It also try to send the data to any "waiter" for this service.
func detected(name string, port int) {
	locker.Lock()
	servicesMap[name] = port
	go func() {
		cx := serviceWaiters[name]
		for _, c := range cx {
			if v, ok := servicesMap[name]; ok {
				c <- v
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
		}
	}()
	locker.Unlock()
	return c
}

func buildSelector(name string, s compose.Service) map[string]string {
	return map[string]string{
		"katenary.io/component": name,
		"katenary.io/release":   "{{ .Release.Name }}",
	}
}
