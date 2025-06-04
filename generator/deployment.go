package generator

import (
	"fmt"
	"katenary/generator/labels"
	"katenary/generator/labels/labelStructs"
	"katenary/utils"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	chart              *HelmChart                              `yaml:"-"`
	configMaps         map[string]*ConfigMapMount              `yaml:"-"`
	volumeMap          map[string]string                       `yaml:"-"` // keep map of fixed named to original volume name
	service            *types.ServiceConfig                    `yaml:"-"`
	defaultTag         string                                  `yaml:"-"`
	isMainApp          bool                                    `yaml:"-"`
	exchangesVolumes   map[string]*labelStructs.ExchangeVolume `yaml:"-"`
	boundEnvVar        []string                                `yaml:"-"` // environement to remove
}

// NewDeployment creates a new Deployment from a compose service. The appName is the name of the application taken from the project name.
// It also creates the Values map that will be used to create the values.yaml file.
func NewDeployment(service types.ServiceConfig, chart *HelmChart) *Deployment {
	isMainApp := false
	if mainLabel, ok := service.Labels[labels.LabelMainApp]; ok {
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
							labels.LabelName("node-selector"): "replace",
						},
					},
				},
			},
		},
		configMaps:       make(map[string]*ConfigMapMount),
		volumeMap:        make(map[string]string),
		exchangesVolumes: map[string]*labelStructs.ExchangeVolume{},
		boundEnvVar:      []string{},
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

func (d *Deployment) AddHealthCheck(service types.ServiceConfig, container *corev1.Container) {
	// get the label for healthcheck
	if v, ok := service.Labels[labels.LabelHealthCheck]; ok {
		probes, err := labelStructs.ProbeFrom(v)
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

// AddIngress adds an ingress to the deployment. It creates the ingress object.
func (d *Deployment) AddIngress(service types.ServiceConfig, appName string) *Ingress {
	return NewIngress(service, d.chart)
}

// AddVolumes adds a volume to the deployment. It does not create the PVC, it only adds the volumes to the deployment.
// If the volume is a bind volume it will warn the user that it is not supported yet.
func (d *Deployment) AddVolumes(service types.ServiceConfig, appName string) {
	tobind := map[string]bool{}
	if v, ok := service.Labels[labels.LabelConfigMapFiles]; ok {
		binds, err := labelStructs.ConfigMapFileFrom(v)
		if err != nil {
			log.Fatal(err)
		}
		for _, bind := range binds {
			tobind[bind] = true
		}
	}

	isSamePod := false
	if v, ok := service.Labels[labels.LabelSamePod]; !ok {
		isSamePod = false
	} else {
		isSamePod = v != ""
	}

	for _, volume := range service.Volumes {
		d.bindVolumes(volume, isSamePod, tobind, service, appName)
	}
}

func (d *Deployment) AddLegacyVolume(name, kind string) {
	// ensure the volume is not present
	for _, v := range d.Spec.Template.Spec.Volumes {
		if v.Name == name {
			return
		}
	}

	// init
	if d.Spec.Template.Spec.Volumes == nil {
		d.Spec.Template.Spec.Volumes = []corev1.Volume{}
	}

	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
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

// DependsOn adds a initContainer to the deployment that will wait for the service to be up.
func (d *Deployment) DependsOn(to *Deployment, servicename string) error {
	// Add a initContainer with busybox:latest using netcat to check if the service is up
	// it will wait until the service responds to all ports
	for _, container := range to.Spec.Template.Spec.Containers {
		commands := []string{}
		if len(container.Ports) == 0 {
			utils.Warn("No ports found for service ",
				servicename,
				". You should declare a port in the service or use "+
					labels.LabelPorts+
					" label.",
			)
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

// Filename returns the filename of the deployment.
func (d *Deployment) Filename() string {
	return d.service.Name + ".deployment.yaml"
}

// SetEnvFrom sets the environment variables to a configmap. The configmap is created.
func (d *Deployment) SetEnvFrom(service types.ServiceConfig, appName string, samePod ...bool) {
	if len(service.Environment) == 0 {
		return
	}
	inSamePod := len(samePod) > 0 && samePod[0]

	drop := []string{}
	secrets := []string{}

	defer func() {
		c, index := d.BindMapFilesToContainer(service, secrets, appName)
		if c == nil || index == -1 {
			log.Println("Container not found for service ", service.Name)
			return
		}
		d.Spec.Template.Spec.Containers[index] = *c
	}()

	// secrets from label
	labelSecrets, err := labelStructs.SecretsFrom(service.Labels[labels.LabelSecrets])
	if err != nil {
		log.Fatal(err)
	}

	// values from label
	varDescriptons := utils.GetValuesFromLabel(service, labels.LabelValues)
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

	if inSamePod {
		return
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
}

func (d *Deployment) BindMapFilesToContainer(service types.ServiceConfig, secrets []string, appName string) (*corev1.Container, int) {
	fromSources := []corev1.EnvFromSource{}

	envSize := len(service.Environment)

	for _, secret := range secrets {
		for k := range service.Environment {
			if k == secret {
				envSize--
			}
		}
	}

	if envSize > 0 {
		if service.Name == "db" {
			log.Println("Service ", service.Name, " has environment variables")
			log.Println(service.Environment)
		}
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
		return nil, -1
	}

	container.EnvFrom = append(container.EnvFrom, fromSources...)

	if container.Env == nil {
		container.Env = []corev1.EnvVar{}
	}
	return container, index
}

func (d *Deployment) MountExchangeVolumes() {
	for name, ex := range d.exchangesVolumes {
		for i, c := range d.Spec.Template.Spec.Containers {
			c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
				Name:      "exchange-" + ex.Name,
				MountPath: ex.MountPath,
			})
			if len(ex.Init) > 0 && name == c.Name {
				d.Spec.Template.Spec.InitContainers = append(d.Spec.Template.Spec.InitContainers, corev1.Container{
					Command: []string{"/bin/sh", "-c", ex.Init},
					Image:   c.Image,
					Name:    "exhange-init-" + name,
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "exchange-" + ex.Name,
						MountPath: ex.MountPath,
					}},
				})
			}
			d.Spec.Template.Spec.Containers[i] = c
		}
	}
}

// Yaml returns the yaml representation of the deployment.
func (d *Deployment) Yaml() ([]byte, error) {
	var y []byte
	var err error
	serviceName := d.service.Name

	if y, err = ToK8SYaml(d); err != nil {
		return nil, err
	}

	// for each volume mount, add a condition "if values has persistence"
	changing := false
	content := strings.Split(string(y), "\n")
	spaces := ""
	volumeName := ""

	nameDirective := "name: "

	// this loop add condition for each volume mount
	for line, volume := range content {
		// find the volume name
		for i := line; i < len(content); i++ {
			if strings.Contains(content[i], nameDirective) {
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
			varName, ok := d.volumeMap[volumeName]
			if !ok {
				// this case happens when the volume is a "bind" volume comming from a "same-pod" service.
				continue
			}
			varName = strings.ReplaceAll(varName, "-", "_")
			content[line] = spaces + `{{- if .Values.` + serviceName + `.persistence.` + varName + `.enabled }}` + "\n" + volume
			changing = true
		}
		if strings.Contains(volume, nameDirective) && changing {
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
			varName := d.volumeMap[volumeName]
			varName = strings.ReplaceAll(varName, "-", "_")
			content[i] = spaces + `{{- if .Values.` + serviceName + `.persistence.` + varName + `.enabled }}` + "\n" + line
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
		if strings.Contains(line, labels.LabelName("node-selector")) {
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

func (d *Deployment) appendDirectoryToConfigMap(service types.ServiceConfig, appName string, volume types.ServiceVolumeConfig) {
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
}

func (d *Deployment) appendFileToConfigMap(service types.ServiceConfig, appName string, volume types.ServiceVolumeConfig) {
	// In case of a file, add it to the configmap and use "subPath" to mount it
	// Note that the volumes and volume mounts are not added to the deployment yet, they will be added later
	// in generate.go
	dirname := filepath.Dir(volume.Source)
	pathname := utils.PathToName(dirname)
	var cm *ConfigMap
	if v, ok := d.configMaps[pathname]; !ok {
		cm = NewConfigMap(*d.service, appName, true)
		cm.usage = FileMapUsageFiles
		cm.path = dirname
		cm.Name = utils.TplName(service.Name, appName) + "-" + pathname
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
	if err := cm.AppendFile(volume.Source); err != nil {
		log.Fatal("Error adding file to configmap:", err)
	}
}

func (d *Deployment) bindVolumes(volume types.ServiceVolumeConfig, isSamePod bool, tobind map[string]bool, service types.ServiceConfig, appName string) {
	container, index := utils.GetContainerByName(service.Name, d.Spec.Template.Spec.Containers)

	defer func(d *Deployment, container *corev1.Container, index int) {
		d.Spec.Template.Spec.Containers[index] = *container
	}(d, container, index)

	if _, found := tobind[volume.Source]; !isSamePod && volume.Type == "bind" && !found {
		utils.Warn(
			"Bind volumes are not supported yet, " +
				"excepting for those declared as " +
				labels.LabelConfigMapFiles +
				", skipping volume " + volume.Source +
				" from service " + service.Name,
		)
		return
	}

	if container == nil {
		utils.Warn("Container not found for volume", volume.Source)
		return
	}

	// ensure that the volume is not already present in the container
	for _, vm := range container.VolumeMounts {
		if vm.Name == volume.Source {
			return
		}
	}

	switch volume.Type {
	case "volume":
		// Add volume to container
		fixedName := utils.FixedResourceName(volume.Source)
		d.volumeMap[fixedName] = volume.Source
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      fixedName,
			MountPath: volume.Target,
		})
		// Add volume to values.yaml only if it the service is not in the same pod that another service.
		// If it is in the same pod, the volume will be added to the other service later
		if _, ok := service.Labels[labels.LabelSamePod]; !ok {
			d.chart.Values[service.Name].(*Value).AddPersistence(volume.Source)
		}
		// Add volume to deployment
		d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: fixedName,
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
			d.appendDirectoryToConfigMap(service, appName, volume)
		} else {
			d.appendFileToConfigMap(service, appName, volume)
		}
	}
}
