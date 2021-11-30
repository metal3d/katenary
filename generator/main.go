package generator

import (
	"fmt"
	"helm-compose/compose"
	"helm-compose/helm"
	"log"
	"strconv"
	"strings"
	"sync"

	"errors"
)

var servicesMap = make(map[string]int)
var serviceWaiters = make(map[string][]chan int)
var locker = &sync.Mutex{}
var serviceTick = make(chan int, 0)

var Ingresses = make(map[string]*helm.Ingress, 0)
var Values = make(map[string]map[string]interface{})

var DependScript = `
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

func CreateReplicaObject(name string, s compose.Service) (ret []interface{}) {

	o := helm.NewDeployment()
	ret = append(ret, o)
	o.Metadata.Name = "{{ .Release.Name }}-" + name

	container := helm.NewContainer(name, s.Image, s.Environment, s.Labels)

	container.Image = "{{ .Values." + name + ".image }}"
	Values[name] = map[string]interface{}{
		"image": s.Image,
	}

	for _, port := range s.Ports {
		portNumber, _ := strconv.Atoi(port)
		container.Ports = append(container.Ports, &helm.ContainerPort{
			Name:          name,
			ContainerPort: portNumber,
		})
	}
	for _, port := range s.Expose {
		container.Ports = append(container.Ports, &helm.ContainerPort{
			Name:          name,
			ContainerPort: port,
		})
	}
	o.Spec.Template.Spec.Containers = []*helm.Container{container}

	o.Spec.Selector = map[string]interface{}{
		"matchLabels": buildSelector(name, s),
	}

	o.Spec.Template.Metadata.Labels = buildSelector(name, s)

	wait := &sync.WaitGroup{}
	initContainers := make([]*helm.Container, 0)
	for _, dp := range s.DependsOn {
		if len(s.Ports) == 0 && len(s.Expose) == 0 {
			log.Fatalf("Sorry, you need to expose or declare at least one port for the %s service to check \"depends_on\"", name)
		}
		c := helm.NewContainer("check-"+name, "busybox", nil, s.Labels)
		command := strings.ReplaceAll(strings.TrimSpace(DependScript), "__service__", dp)

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

	return
}

func createService(name string, s compose.Service) *helm.Service {

	ks := helm.NewService()
	ks.Metadata.Name = "{{ .Release.Name }}-" + name
	defaultPort := 0
	for i, p := range s.Ports {
		port := strings.Split(p, ":")
		src, _ := strconv.Atoi(port[0])
		target := src
		if len(port) > 1 {
			target, _ = strconv.Atoi(port[1])
		}
		ks.Spec.Ports = append(ks.Spec.Ports, helm.NewServicePort(src, target))
		if i == 0 {
			defaultPort = target
			detected(name, target)
		}
	}
	for i, p := range s.Expose {
		ks.Spec.Ports = append(ks.Spec.Ports, helm.NewServicePort(p, p))
		if i == 0 {
			defaultPort = p
			detected(name, p)
		}
	}

	ks.Spec.Selector = buildSelector(name, s)

	if v, ok := s.Labels[helm.K+"/expose-ingress"]; ok && v == "true" {
		log.Println("Expose ingress for ", name)
		createIngress(name, defaultPort, s)
	}

	return ks
}

func createIngress(name string, port int, s compose.Service) {
	ingress := helm.NewIngress(name)
	Values[name]["ingress"] = map[string]interface{}{
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

	locker.Lock()
	Ingresses[name] = ingress
	locker.Unlock()
}

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
