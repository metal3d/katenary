package generator

import (
	"fmt"
	"katenary/generator/labels"
	"katenary/generator/labels/labelStructs"
	"katenary/utils"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/compose-spec/compose-go/types"
	corev1 "k8s.io/api/core/v1"
)

// ChartTemplate is a template of a chart. It contains the content of the template and the name of the service.
// This is used internally to generate the templates.
type ChartTemplate struct {
	Servicename string
	Content     []byte
}

// ConvertOptions are the options to convert a compose project to a helm chart.
type ConvertOptions struct {
	AppVersion   *string
	OutputDir    string
	ChartVersion string
	Icon         string
	Profiles     []string
	EnvFiles     []string
	Force        bool
	HelmUpdate   bool
}

// HelmChart is a Helm Chart representation. It contains all the
// templates, values, versions, helpers...
type HelmChart struct {
	Templates    map[string]*ChartTemplate `yaml:"-"`
	Values       map[string]any            `yaml:"-"`
	VolumeMounts map[string]any            `yaml:"-"`
	composeHash  *string                   `yaml:"-"`
	Name         string                    `yaml:"name"`
	Icon         string                    `yaml:"icon,omitempty"`
	APIVersion   string                    `yaml:"apiVersion"`
	Version      string                    `yaml:"version"`
	AppVersion   string                    `yaml:"appVersion"`
	Description  string                    `yaml:"description"`
	Helper       string                    `yaml:"-"`
	Dependencies []labelStructs.Dependency `yaml:"dependencies,omitempty"`
}

// NewChart creates a new empty chart with the given name.
func NewChart(name string) *HelmChart {
	return &HelmChart{
		Name:        name,
		Templates:   make(map[string]*ChartTemplate, 0),
		Description: "A Helm chart for " + name,
		APIVersion:  "v2",
		Version:     "",
		AppVersion:  "", // set to 0.1.0 by default if no "main-app" label is found
		Values: map[string]any{
			"pullSecrets": []string{},
		},
	}
}

// SaveTemplates the templates of the chart to the given directory.
func (chart *HelmChart) SaveTemplates(templateDir string) {
	for name, template := range chart.Templates {
		t := template.Content
		t = removeNewlinesInsideBrackets(t)
		t = removeUnwantedLines(t)
		// t = addModeline(t)

		kind := utils.GetKind(name)
		var icon utils.Icon
		switch kind {
		case "deployment":
			icon = utils.IconPackage
		case "service":
			icon = utils.IconPlug
		case "ingress":
			icon = utils.IconWorld
		case "volumeclaim":
			icon = utils.IconCabinet
		case "configmap":
			icon = utils.IconConfig
		case "secret":
			icon = utils.IconSecret
		default:
			icon = utils.IconInfo
		}

		servicename := template.Servicename
		if err := os.MkdirAll(filepath.Join(templateDir, servicename), utils.DirectoryPermission); err != nil {
			fmt.Println(utils.IconFailure, err)
			os.Exit(1)
		}
		fmt.Println(icon, "Creating", kind, servicename)
		// if the name is a path, create the directory
		if strings.Contains(name, string(filepath.Separator)) {
			name = filepath.Join(templateDir, name)
			err := os.MkdirAll(filepath.Dir(name), utils.DirectoryPermission)
			if err != nil {
				fmt.Println(utils.IconFailure, err)
				os.Exit(1)
			}
		} else {
			// remove the serivce name from the template name
			name = strings.Replace(name, servicename+".", "", 1)
			name = filepath.Join(templateDir, servicename, name)
		}
		f, err := os.Create(name)
		if err != nil {
			fmt.Println(utils.IconFailure, err)
			os.Exit(1)
		}
		defer f.Close()
		if _, err := f.Write(t); err != nil {
			log.Fatal("error writing template file:", err)
		}

	}
}

// generateConfigMapsAndSecrets creates the configmaps and secrets from the environment variables.
func (chart *HelmChart) generateConfigMapsAndSecrets(project *types.Project) error {
	appName := chart.Name
	for _, s := range project.Services {
		if len(s.Environment) == 0 {
			continue
		}

		originalEnv := types.MappingWithEquals{}
		secretsVar := types.MappingWithEquals{}

		// copy env to originalEnv
		maps.Copy(originalEnv, s.Environment)

		if v, ok := s.Labels[labels.LabelSecrets]; ok {
			list, err := labelStructs.SecretsFrom(v)
			if err != nil {
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
			cm := NewConfigMap(s, appName, false)
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

func (chart *HelmChart) generateDeployment(service types.ServiceConfig, deployments map[string]*Deployment, services map[string]*Service, podToMerge map[string]*types.ServiceConfig, appName string) error {
	// check the "ports" label from container and add it to the service
	if err := fixPorts(&service); err != nil {
		return err
	}

	// isgnored service
	if isIgnored(service) {
		fmt.Printf("%s Ignoring service %s\n", utils.IconInfo, service.Name)
		return nil
	}

	// helm dependency
	if isHelmDependency, err := chart.setDependencies(service); err != nil {
		return err
	} else if isHelmDependency {
		return nil
	}

	// create all deployments
	d := NewDeployment(service, chart)
	deployments[service.Name] = d

	// generate the cronjob if needed
	chart.setCronJob(service, appName)

	if exchange, ok := service.Labels[labels.LabelExchangeVolume]; ok {
		// we need to add a volume and a mount point
		ex, err := labelStructs.NewExchangeVolumes(exchange)
		if err != nil {
			return err
		}
		for _, exchangeVolume := range ex {
			d.AddLegacyVolume("exchange-"+exchangeVolume.Name, exchangeVolume.Type)
			d.exchangesVolumes[service.Name] = exchangeVolume
		}
	}

	// get the same-pod label if exists, add it to the list.
	// We later will copy some parts to the target deployment and remove this one.
	if samePod, ok := service.Labels[labels.LabelSamePod]; ok && samePod != "" {
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

	return nil
}

// setChartVersion sets the chart version from the service image tag.
func (chart *HelmChart) setChartVersion(service types.ServiceConfig) {
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

// setCronJob creates a cronjob from the service labels.
func (chart *HelmChart) setCronJob(service types.ServiceConfig, appName string) *CronJob {
	if _, ok := service.Labels[labels.LabelCronJob]; !ok {
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
func (chart *HelmChart) setDependencies(service types.ServiceConfig) (bool, error) {
	// helm dependency
	if v, ok := service.Labels[labels.LabelDependencies]; ok {
		d, err := labelStructs.DependenciesFrom(v)
		if err != nil {
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

// setSharedConf sets the shared configmap to the service.
func (chart *HelmChart) setSharedConf(service types.ServiceConfig, deployments map[string]*Deployment) {
	// if the service has the "shared-conf" label, we need to add the configmap
	// to the chart and add the env vars to the service
	if _, ok := service.Labels[labels.LabelEnvFrom]; !ok {
		return
	}
	fromservices, err := labelStructs.EnvFromFrom(service.Labels[labels.LabelEnvFrom])
	if err != nil {
		log.Fatal("error unmarshaling env-from label:", err)
	}
	// find the configmap in the chart templates
	for _, fromservice := range fromservices {
		if _, ok := chart.Templates[fromservice+".configmap.yaml"]; !ok {
			log.Printf("configmap %s not found in chart templates", fromservice)
			continue
		}
		// find the corresponding target deployment
		target := findDeployment(service.Name, deployments)
		if target == nil {
			continue
		}
		// add the configmap to the service
		addConfigMapToService(service.Name, fromservice, chart.Name, target)
	}
}

// setEnvironmentValuesFrom sets the environment values from another service.
func (chart *HelmChart) setEnvironmentValuesFrom(service types.ServiceConfig, deployments map[string]*Deployment) {
	if _, ok := service.Labels[labels.LabelValueFrom]; !ok {
		return
	}
	mapping, err := labelStructs.GetValueFrom(service.Labels[labels.LabelValueFrom])
	if err != nil {
		log.Fatal("error unmarshaling values-from label:", err)
	}

	findDeployment := func(name string) *Deployment {
		for _, dep := range deployments {
			if dep.service.Name == name {
				return dep
			}
		}
		return nil
	}

	// each mapping key is the environment, and the value is serivename.variable name
	for env, from := range *mapping {
		// find the deployment that has the variable
		depName := strings.Split(from, ".")
		dep := findDeployment(depName[0])
		target := findDeployment(service.Name)
		if dep == nil || target == nil {
			log.Fatalf("deployment %s or %s not found", depName[0], service.Name)
		}
		container, index := utils.GetContainerByName(target.service.ContainerName, target.Spec.Template.Spec.Containers)
		if container == nil {
			log.Fatalf("Container %s not found", target.GetName())
		}
		reourceName := fmt.Sprintf(`{{ include "%s.fullname" . }}-%s`, chart.Name, depName[0])
		// add environment with from

		// is it a secret?
		isSecret := false
		secrets, err := labelStructs.SecretsFrom(dep.service.Labels[labels.LabelSecrets])
		if err == nil {
			if slices.Contains(secrets, depName[1]) {
				isSecret = true
			}
		}

		if !isSecret {
			container.Env = append(container.Env, corev1.EnvVar{
				Name: env,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: reourceName,
						},
						Key: depName[1],
					},
				},
			})
		} else {
			container.Env = append(container.Env, corev1.EnvVar{
				Name: env,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: reourceName,
						},
						Key: depName[1],
					},
				},
			})
		}
		// the environment is bound, so we shouldn't add it to the values.yaml or in any other place
		delete(service.Environment, env)
		// also, remove the values
		target.boundEnvVar = append(target.boundEnvVar, env)
		// and save the container
		target.Spec.Template.Spec.Containers[index] = *container

	}
}
