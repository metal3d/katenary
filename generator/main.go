package generator

import (
	"fmt"
	"io/ioutil"
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

var (
	EmptyDirs   = []string{}
	servicesMap = make(map[string]int)
	locker      = &sync.Mutex{}

	dependScript = `
OK=0
echo "Checking __service__ port"
while [ $OK != 1 ]; do
    echo -n "."
    nc -z __service__ __port__ 2>&1 >/dev/null && OK=1 || sleep 1
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

func setSecretVar(name string, s *types.ServiceConfig, c *helm.Container) *helm.Secret {
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
		AddEnvironment(name, secretvar, *s.Environment[secretvar])

		// Finally remove the secret var from the environment on the service
		// and the helm container definition.
		defer func(secretvar string) { // defered because AddEnvironment locks the memory
			locker.Lock()
			defer locker.Unlock()

			for i, env := range c.Env {
				if env.Name == secretvar {
					c.Env = append(c.Env[:i], c.Env[i+1:]...)
					i--
				}
			}

			delete(s.Environment, secretvar)
		}(secretvar)
	}
	return store
}
