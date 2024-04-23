package generator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"katenary/utils"

	"github.com/compose-spec/compose-go/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var _ Yaml = (*Deployment)(nil)

type mountPathConfig struct {
	mountPath string
	subPath   string
}

type ConfigMapMount struct {
	configMap *ConfigMap
	mountPath []mountPathConfig
}

// Deployment is a kubernetes Deployment.
type Deployment struct {
	*appsv1.Deployment `yaml:",inline"`
	chart              *HelmChart                 `yaml:"-"`
	configMaps         map[string]*ConfigMapMount `yaml:"-"`
	service            *types.ServiceConfig       `yaml:"-"`
	defaultTag         string                     `yaml:"-"`
	isMainApp          bool                       `yaml:"-"`
}

// NewDeployment creates a new Deployment from a compose service. The appName is the name of the application taken from the project name.
// It also creates the Values map that will be used to create the values.yaml file.
func NewDeployment(service types.ServiceConfig, chart *HelmChart) *Deployment {
	isMainApp := false
	if mainLabel, ok := service.Labels[LABEL_MAIN_APP]; ok {
		main := strings.ToLower(mainLabel)
		isMainApp = main == "true" || main == "yes" || main == "1"
	}

	defaultTag := `default "latest"`
	if isMainApp {
		defaultTag = `default .Chart.AppVersion`
	}

	chart.Values[service.Name] = NewValue(service, isMainApp)
	appName := chart.Name

	dep := &Deployment{
		isMainApp:  isMainApp,
		defaultTag: defaultTag,
		service:    &service,
		chart:      chart,
		Deployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        utils.TplName(service.Name, appName),
				Labels:      GetLabels(service.Name, appName),
				Annotations: Annotations,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: utils.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: GetMatchLabels(service.Name, appName),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: GetMatchLabels(service.Name, appName),
					},
					Spec: corev1.PodSpec{
						NodeSelector: map[string]string{
							"katenary.v3/node-selector": "replace",
						},
					},
				},
			},
		},
		configMaps: make(map[string]*ConfigMapMount),
	}

	// add containers
	dep.AddContainer(service)

	// add volumes
	dep.AddVolumes(service, appName)

	if service.Environment != nil {
		dep.SetEnvFrom(service, appName)
	}

	return dep
}

// DependsOn adds a initContainer to the deployment that will wait for the service to be up.
func (d *Deployment) DependsOn(to *Deployment, servicename string) error {
	// Add a initContainer with busybox:latest using netcat to check if the service is up
	// it will wait until the service responds to all ports
	for _, container := range to.Spec.Template.Spec.Containers {
		commands := []string{}
		if len(container.Ports) == 0 {
			utils.Warn("No ports found for service ", servicename, ". You should declare a port in the service or use "+LABEL_PORTS+" label.")
			os.Exit(1)
		}
		for _, port := range container.Ports {
			command := fmt.Sprintf("until nc -z %s %d; do\n  sleep 1;\ndone", to.Name, port.ContainerPort)
			commands = append(commands, command)
		}

		command := []string{"/bin/sh", "-c", strings.Join(commands, "\n")}
		d.Spec.Template.Spec.InitContainers = append(d.Spec.Template.Spec.InitContainers, corev1.Container{
			Name:    "wait-for-" + to.service.Name,
			Image:   "busybox:latest",
			Command: command,
		})
	}

	return nil
}

// AddContainer adds a container to the deployment.
func (d *Deployment) AddContainer(service types.ServiceConfig) {
	ports := []corev1.ContainerPort{}

	for _, port := range service.Ports {
		name := utils.GetServiceNameByPort(int(port.Target))
		if name == "" {
			utils.Warn("Port name not found for port ", port.Target, " in service ", service.Name, ". Using port number instead")
			name = fmt.Sprintf("port-%d", port.Target)
		}
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: int32(port.Target),
			Name:          name,
		})
	}

	container := corev1.Container{
		Image: utils.TplValue(service.Name, "repository.image") + ":" +
			utils.TplValue(service.Name, "repository.tag", d.defaultTag),
		Ports:           ports,
		Name:            service.Name,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
		},
	}
	if _, ok := d.chart.Values[service.Name]; !ok {
		d.chart.Values[service.Name] = NewValue(service, d.isMainApp)
	}
	d.chart.Values[service.Name].(*Value).ImagePullPolicy = string(corev1.PullIfNotPresent)

	// add an imagePullSecret, it actually does not work because the secret is not
	// created but it add the reference in the YAML file. We'll change it in Yaml()
	// method.
	d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{
		Name: `{{ .Values.pullSecrets | toYaml | indent __indent__ }}`,
	}}

	// add ServiceAccount to the deployment
	d.Spec.Template.Spec.ServiceAccountName = `{{ .Values.` + service.Name + `.serviceAccount | quote }}`

	d.AddHealthCheck(service, &container)

	d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, container)
}

// AddIngress adds an ingress to the deployment. It creates the ingress object.
func (d *Deployment) AddIngress(service types.ServiceConfig, appName string) *Ingress {
	return NewIngress(service, d.chart)
}

// AddVolumes adds a volume to the deployment. It does not create the PVC, it only adds the volumes to the deployment.
// If the volume is a bind volume it will warn the user that it is not supported yet.
func (d *Deployment) AddVolumes(service types.ServiceConfig, appName string) {
	tobind := map[string]bool{}
	if v, ok := service.Labels[LABEL_CM_FILES]; ok {
		binds := []string{}
		if err := yaml.Unmarshal([]byte(v), &binds); err != nil {
			log.Fatal(err)
		}
		for _, bind := range binds {
			tobind[bind] = true
		}
	}

	isSamePod := false
	if v, ok := service.Labels[LABEL_SAME_POD]; !ok {
		isSamePod = false
	} else {
		isSamePod = v != ""
	}

	container, index := utils.GetContainerByName(service.Name, d.Spec.Template.Spec.Containers)
	defer func(d *Deployment, container *corev1.Container, index int) {
		d.Spec.Template.Spec.Containers[index] = *container
	}(d, container, index)

	for _, volume := range service.Volumes {
		// not declared as a bind volume, skip
		if _, ok := tobind[volume.Source]; !isSamePod && volume.Type == "bind" && !ok {
			utils.Warn(
				"Bind volumes are not supported yet, " +
					"excepting for those declared as " +
					LABEL_CM_FILES +
					", skipping volume " + volume.Source +
					" from service " + service.Name,
			)
			continue
		}

		if container == nil {
			utils.Warn("Container not found for volume", volume.Source)
			continue
		}

		// ensure that the volume is not already present in the container
		for _, vm := range container.VolumeMounts {
			if vm.Name == volume.Source {
				continue
			}
		}

		switch volume.Type {
		case "volume":
			// Add volume to container
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      volume.Source,
				MountPath: volume.Target,
			})
			// Add volume to values.yaml only if it the service is not in the same pod that another service.
			// If it is in the same pod, the volume will be added to the other service later
			if _, ok := service.Labels[LABEL_SAME_POD]; !ok {
				d.chart.Values[service.Name].(*Value).AddPersistence(volume.Source)
			}
			// Add volume to deployment
			d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: volume.Source,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: utils.TplName(service.Name, appName, volume.Source),
					},
				},
			})
		case "bind":
			// Add volume to container
			stat, err := os.Stat(volume.Source)
			if err != nil {
				log.Fatal(err)
			}

			if stat.IsDir() {
				pathnme := utils.PathToName(volume.Source)
				if _, ok := d.configMaps[pathnme]; !ok {
					d.configMaps[pathnme] = &ConfigMapMount{
						mountPath: []mountPathConfig{},
					}
				}

				// TODO: make it recursive to add all files in the directory and subdirectories
				_, err := os.ReadDir(volume.Source)
				if err != nil {
					log.Fatal(err)
				}
				cm := NewConfigMapFromDirectory(service, appName, volume.Source)
				d.configMaps[pathnme] = &ConfigMapMount{
					configMap: cm,
					mountPath: append(d.configMaps[pathnme].mountPath, mountPathConfig{
						mountPath: volume.Target,
					}),
				}
			} else {
				dirname := filepath.Dir(volume.Source)
				pathname := utils.PathToName(dirname)
				var cm *ConfigMap
				if v, ok := d.configMaps[pathname]; !ok {
					cm = NewConfigMap(*d.service, appName)
					cm.usage = FileMapUsageFiles
					cm.path = dirname
					cm.Name = utils.TplName(service.Name, appName) + "-" + pathname
					// assign a new mountPathConfig to the configMap
					d.configMaps[pathname] = &ConfigMapMount{
						configMap: cm,
						mountPath: []mountPathConfig{{
							mountPath: volume.Target,
							subPath:   filepath.Base(volume.Source),
						}},
					}
				} else {
					cm = v.configMap
					mp := d.configMaps[pathname].mountPath
					mp = append(mp, mountPathConfig{
						mountPath: volume.Target,
						subPath:   filepath.Base(volume.Source),
					})
					d.configMaps[pathname].mountPath = mp

				}
				cm.AppendFile(volume.Source)
			}
		}
	}
}

func (d *Deployment) BindFrom(service types.ServiceConfig, binded *Deployment) {
	// find the volume in the binded deployment
	for _, bindedVolume := range binded.Spec.Template.Spec.Volumes {
		skip := false
		for _, targetVol := range d.Spec.Template.Spec.Volumes {
			if targetVol.Name == bindedVolume.Name {
				skip = true
				break
			}
		}
		if !skip {
			// add the volume to the current deployment
			d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, bindedVolume)
			// get the container
		}
		// add volume mount to the container
		targetContainer, ti := utils.GetContainerByName(service.Name, d.Spec.Template.Spec.Containers)
		sourceContainer, _ := utils.GetContainerByName(service.Name, binded.Spec.Template.Spec.Containers)
		for _, bindedMount := range sourceContainer.VolumeMounts {
			if bindedMount.Name == bindedVolume.Name {
				targetContainer.VolumeMounts = append(targetContainer.VolumeMounts, bindedMount)
			}
		}
		d.Spec.Template.Spec.Containers[ti] = *targetContainer
	}
}

// SetEnvFrom sets the environment variables to a configmap. The configmap is created.
func (d *Deployment) SetEnvFrom(service types.ServiceConfig, appName string) {
	if len(service.Environment) == 0 {
		return
	}

	drop := []string{}
	secrets := []string{}

	// secrets from label
	labelSecrets := []string{}
	if v, ok := service.Labels[LABEL_SECRETS]; ok {
		err := yaml.Unmarshal([]byte(v), &labelSecrets)
		if err != nil {
			log.Fatal(err)
		}
	}

	// values from label
	varDescriptons := utils.GetValuesFromLabel(service, LABEL_VALUES)
	labelValues := []string{}
	for v := range varDescriptons {
		labelValues = append(labelValues, v)
	}

	for _, secret := range labelSecrets {
		// get the secret name
		_, ok := service.Environment[secret]
		if !ok {
			drop = append(drop, secret)
			utils.Warn("Secret " + secret + " not found in service " + service.Name + " - skpped")
			continue
		}
		secrets = append(secrets, secret)
	}

	// for each values from label "values", add it to Values map and change the envFrom
	// value to {{ .Values.<service>.<value> }}
	for _, value := range labelValues {
		// get the environment variable name
		val, ok := service.Environment[value]
		if !ok {
			drop = append(drop, value)
			utils.Warn("Environment variable " + value + " not found in service " + service.Name + " - skpped")
			continue
		}
		if d.chart.Values[service.Name].(*Value).Environment == nil {
			d.chart.Values[service.Name].(*Value).Environment = make(map[string]any)
		}
		d.chart.Values[service.Name].(*Value).Environment[value] = *val
		// set the environment variable to bind to the values.yaml file
		v := utils.TplValue(service.Name, "environment."+value)
		service.Environment[value] = &v
	}

	for _, value := range drop {
		delete(service.Environment, value)
	}

	fromSources := []corev1.EnvFromSource{}

	if len(service.Environment) > 0 {
		fromSources = append(fromSources, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: utils.TplName(service.Name, appName),
				},
			},
		})
	}

	if len(secrets) > 0 {
		fromSources = append(fromSources, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: utils.TplName(service.Name, appName),
				},
			},
		})
	}

	container, index := utils.GetContainerByName(service.Name, d.Spec.Template.Spec.Containers)
	if container == nil {
		utils.Warn("Container not found for service " + service.Name)
		return
	}

	container.EnvFrom = append(container.EnvFrom, fromSources...)

	if container.Env == nil {
		container.Env = []corev1.EnvVar{}
	}

	d.Spec.Template.Spec.Containers[index] = *container
}

func (d *Deployment) AddHealthCheck(service types.ServiceConfig, container *corev1.Container) {
	// get the label for healthcheck
	if v, ok := service.Labels[LABEL_HEALTHCHECK]; ok {
		probes := struct {
			LivenessProbe  *corev1.Probe `yaml:"livenessProbe"`
			ReadinessProbe *corev1.Probe `yaml:"readinessProbe"`
		}{}
		err := yaml.Unmarshal([]byte(v), &probes)
		if err != nil {
			log.Fatal(err)
		}
		container.LivenessProbe = probes.LivenessProbe
		container.ReadinessProbe = probes.ReadinessProbe
		return
	}

	if service.HealthCheck != nil {
		period := 30.0
		if service.HealthCheck.Interval != nil {
			period = time.Duration(*service.HealthCheck.Interval).Seconds()
		}
		container.LivenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: service.HealthCheck.Test[1:],
				},
			},
			PeriodSeconds: int32(period),
		}
	}
}

// Yaml returns the yaml representation of the deployment.
func (d *Deployment) Yaml() ([]byte, error) {
	serviceName := d.service.Name
	y, err := yaml.Marshal(d)
	if err != nil {
		return nil, err
	}

	// for each volume mount, add a condition "if values has persistence"
	changing := false
	content := strings.Split(string(y), "\n")
	spaces := ""
	volumeName := ""

	// this loop add condition for each volume mount
	for line, volume := range content {
		// find the volume name
		for i := line; i < len(content); i++ {
			if strings.Contains(content[i], "name: ") {
				volumeName = strings.TrimSpace(strings.Replace(content[i], "name: ", "", 1))
				break
			}
		}
		if volumeName == "" {
			continue
		}

		if _, ok := d.configMaps[volumeName]; ok {
			continue
		}

		if strings.Contains(volume, "mountPath: ") {
			spaces = strings.Repeat(" ", utils.CountStartingSpaces(volume))
			content[line] = spaces + `{{- if .Values.` + serviceName + `.persistence.` + volumeName + `.enabled }}` + "\n" + volume
			changing = true
		}
		if strings.Contains(volume, "name: ") && changing {
			content[line] = volume + "\n" + spaces + "{{- end }}"
			changing = false
		}
	}

	changing = false
	inVolumes := false
	volumeName = ""
	// this loop changes imagePullPolicy to {{ .Values.<service>.imagePullPolicy }}
	// and the volume definition adding the condition "if values has persistence"
	for i, line := range content {

		if strings.Contains(line, "imagePullPolicy:") {
			spaces = strings.Repeat(" ", utils.CountStartingSpaces(line))
			content[i] = spaces + "imagePullPolicy: {{ .Values." + serviceName + ".imagePullPolicy }}"
		}

		// find the volume name
		for i := i; i < len(content); i++ {
			if strings.Contains(content[i], "- name: ") {
				volumeName = strings.TrimSpace(strings.Replace(content[i], "- name: ", "", 1))
				break
			}
		}
		if strings.Contains(line, "volumes:") {
			inVolumes = true
		}

		if volumeName == "" {
			continue
		}

		if _, ok := d.configMaps[volumeName]; ok {
			continue
		}

		if strings.Contains(line, "- name: ") && inVolumes {
			spaces = strings.Repeat(" ", utils.CountStartingSpaces(line))
			content[i] = spaces + `{{- if .Values.` + serviceName + `.persistence.` + volumeName + `.enabled }}` + "\n" + line
			changing = true
		}
		if strings.Contains(line, "claimName: ") && changing {
			content[i] = line + "\n" + spaces + "{{- end }}"
			changing = false
		}
	}

	// for impagePullSecrets, replace the name with the value from values.yaml
	for i, line := range content {
		if strings.Contains(line, "imagePullSecrets:") {
			spaces = strings.Repeat(" ", utils.CountStartingSpaces(line))
			line = spaces + "{{- if .Values.pullSecrets }}"
			line += "\n" + spaces + "imagePullSecrets:\n"
			line += spaces + "{{- .Values.pullSecrets | toYaml | nindent __indent__ }}"
			line += "\n" + spaces + "{{- end }}"
			content[i] = line
		}
	}

	// Find the replicas line and replace it with the value from values.yaml
	for i, line := range content {
		// manage nodeSelector
		if strings.Contains(line, "nodeSelector:") {
			spaces = strings.Repeat(" ", utils.CountStartingSpaces(line))
			pre := spaces + `{{- if .Values.` + serviceName + `.nodeSelector }}`
			post := spaces + "{{- end }}"
			ns := spaces + "nodeSelector:\n"
			ns += spaces + `  {{- .Values.` + serviceName + `.nodeSelector | toYaml | nindent __indent__ }}`
			//line = strings.Replace(line, "katenary.v3/node-selector: replace", ns, 1)
			line = pre + "\n" + ns + "\n" + post
		}
		// manage replicas
		if strings.Contains(line, "replicas:") {
			line = regexp.MustCompile("replicas: .*$").ReplaceAllString(line, "replicas: {{ .Values."+serviceName+".replicas }}")
		}

		// manage serviceAccount, add condition to use the serviceAccount from values.yaml
		if strings.Contains(line, "serviceAccountName:") {
			spaces = strings.Repeat(" ", utils.CountStartingSpaces(line))
			pre := spaces + `{{- if ne .Values.` + serviceName + `.serviceAccount "" }}`
			post := spaces + "{{- end }}"
			line = strings.ReplaceAll(line, "'", "")
			line = pre + "\n" + line + "\n" + post
		}

		if strings.Contains(line, "resources: {}") {
			spaces = strings.Repeat(" ", utils.CountStartingSpaces(line))
			pre := spaces + `{{- if .Values.` + serviceName + `.resources }}`
			post := spaces + "{{- end }}"

			line = strings.ReplaceAll(line, "resources: {}", "resources:")
			line += "\n" + spaces + "  {{ .Values." + serviceName + ".resources | toYaml | nindent __indent__ }}"
			line = pre + "\n" + line + "\n" + post
		}

		content[i] = line
	}

	// find the katenary.v3/node-selector line, and remove it
	for i, line := range content {
		if strings.Contains(line, "katenary.v3/node-selector") {
			content = append(content[:i], content[i+1:]...)
			continue
		}
		if strings.Contains(line, "- name: '{{ .Values.pullSecrets ") {
			content = append(content[:i], content[i+1:]...)
			continue
		}
	}

	return []byte(strings.Join(content, "\n")), nil
}

// Filename returns the filename of the deployment.
func (d *Deployment) Filename() string {
	return d.service.Name + ".deployment.yaml"
}
