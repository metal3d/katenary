package generator

import (
	"log"
	"strings"

	"github.com/compose-spec/compose-go/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"katenary/generator/labelStructs"
	"katenary/utils"
)

// only used to check interface implementation
var (
	_ Yaml = (*CronJob)(nil)
)

// CronJob is a kubernetes CronJob.
type CronJob struct {
	*batchv1.CronJob
	service *types.ServiceConfig
}

// NewCronJob creates a new CronJob from a compose service. The appName is the name of the application taken from the project name.
func NewCronJob(service types.ServiceConfig, chart *HelmChart, appName string) (*CronJob, *RBAC) {
	labels, ok := service.Labels[LabelCronJob]
	if !ok {
		return nil, nil
	}
	mapping, err := labelStructs.CronJobFrom(labels)
	if err != nil {
		log.Fatalf("Error parsing cronjob labels: %s", err)
		return nil, nil
	}

	if _, ok := chart.Values[service.Name]; !ok {
		chart.Values[service.Name] = NewValue(service, false)
	}
	if chart.Values[service.Name].(*Value).CronJob == nil {
		chart.Values[service.Name].(*Value).CronJob = &CronJobValue{}
	}
	chart.Values[service.Name].(*Value).CronJob.Schedule = mapping.Schedule
	chart.Values[service.Name].(*Value).CronJob.ImagePullPolicy = "IfNotPresent"
	chart.Values[service.Name].(*Value).CronJob.Environment = map[string]any{}

	image, tag := mapping.Image, ""
	if image == "" { // if image is not set, use the image from the service
		image = service.Image
	}

	if strings.Contains(image, ":") {
		image = strings.Split(service.Image, ":")[0]
		tag = strings.Split(service.Image, ":")[1]
	}

	chart.Values[service.Name].(*Value).CronJob.Repository = &RepositoryValue{
		Image: image,
		Tag:   tag,
	}

	cronjob := &CronJob{
		CronJob: &batchv1.CronJob{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: "batch/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        utils.TplName(service.Name, appName),
				Labels:      GetLabels(service.Name, appName),
				Annotations: Annotations,
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "{{ .Values." + service.Name + ".cronjob.schedule }}",
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "cronjob",
										Image: "{{ .Values." + service.Name + ".cronjob.repository.image }}:{{ default .Values." + service.Name + ".cronjob.repository.tag \"latest\" }}",
										Command: []string{
											"sh",
											"-c",
											mapping.Command,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		service: &service,
	}

	var rbac *RBAC
	if mapping.Rbac {
		rbac = NewRBAC(service, appName)
		// add the service account to the cronjob
		cronjob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = utils.TplName(service.Name, appName)
	}

	return cronjob, rbac
}

// Filename returns the filename of the cronjob.
//
// Implements the Yaml interface.
func (c *CronJob) Filename() string {
	return c.service.Name + ".cronjob.yaml"
}

// Yaml returns the yaml representation of the cronjob.
//
// Implements the Yaml interface.
func (c *CronJob) Yaml() ([]byte, error) {
	return yaml.Marshal(c)
}
