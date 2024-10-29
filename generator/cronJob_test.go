package generator

import (
	"os"
	"strings"
	"testing"

	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestBasicCronJob(t *testing.T) {
	composeFile := `
services:
    cron:
        image: fedora
        labels:
            katenary.v3/cronjob: |
                image: alpine
                command: echo hello
                schedule: "*/1 * * * *"
                rbac: false
`
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/cron/cronjob.yaml")
	cronJob := batchv1.CronJob{}
	if err := yaml.Unmarshal([]byte(output), &cronJob); err != nil {
		t.Errorf(unmarshalError, err)
	}
	if cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image != "alpine:latest" {
		t.Errorf("Expected image to be alpine, got %s", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	}
	combinedCommand := strings.Join(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Command, " ")
	if combinedCommand != "sh -c echo hello" {
		t.Errorf("Expected command to be sh -c echo hello, got %s", combinedCommand)
	}
	if cronJob.Spec.Schedule != "*/1 * * * *" {
		t.Errorf("Expected schedule to be */1 * * * *, got %s", cronJob.Spec.Schedule)
	}

	// ensure that there are a deployment for the fedora Container
	var err error
	output, err = helmTemplate(ConvertOptions{
		OutputDir: "./chart",
	}, "-s", "templates/cron/deployment.yaml")
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	deployment := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &deployment); err != nil {
		t.Errorf(unmarshalError, err)
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != "fedora:latest" {
		t.Errorf("Expected image to be fedora, got %s", deployment.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestCronJobbWithRBAC(t *testing.T) {
	composeFile := `
services:
    cron:
        image: fedora
        labels:
            katenary.v3/cronjob: |
                image: alpine
                command: echo hello
                schedule: "*/1 * * * *"
                rbac: true
`

	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/cron/cronjob.yaml")
	cronJob := batchv1.CronJob{}
	if err := yaml.Unmarshal([]byte(output), &cronJob); err != nil {
		t.Errorf(unmarshalError, err)
	}
	if cronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName == "" {
		t.Errorf("Expected ServiceAccountName to be set")
	}

	// find the service account file
	output, err := helmTemplate(ConvertOptions{
		OutputDir: "./chart",
	}, "-s", "templates/cron/serviceaccount.yaml")
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	serviceAccount := corev1.ServiceAccount{}

	if err := yaml.Unmarshal([]byte(output), &serviceAccount); err != nil {
		t.Errorf(unmarshalError, err)
	}
	if serviceAccount.Name == "" {
		t.Errorf("Expected ServiceAccountName to be set")
	}

	// ensure that the serviceAccount is equal to the cronJob
	if serviceAccount.Name != cronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName {
		t.Errorf("Expected ServiceAccountName to be %s, got %s", cronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName, serviceAccount.Name)
	}
}
