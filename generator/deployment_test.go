package generator

import (
	"os"
	"testing"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestGenerate(t *testing.T) {
	compose_file := `
services:
    web:
        image: nginx:1.29
`
	tmpDir := setup(compose_file)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := _compile_test(t, "-s", "templates/web/deployment.yaml")

	// dt := DeploymentTest{}
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if *dt.Spec.Replicas != 1 {
		t.Errorf("Expected replicas to be 1, got %d", dt.Spec.Replicas)
		t.Errorf("Output: %s", output)
	}

	if dt.Spec.Template.Spec.Containers[0].Image != "nginx:1.29" {
		t.Errorf("Expected image to be nginx:1.29, got %s", dt.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestGenerateOneDeploymentWithSamePod(t *testing.T) {
	compose_file := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80

    fpm:
        image: php:fpm
        ports:
        - 9000:9000
        labels:
            katenary.v3/same-pod: web
`

	tmpDir := setup(compose_file)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := _compile_test(t, "-s", "templates/web/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if len(dt.Spec.Template.Spec.Containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(dt.Spec.Template.Spec.Containers))
	}
	// endsure that the fpm service is not created

	var err error
	output, err = helmTemplate(ConvertOptions{
		OutputDir: "./chart",
	}, "-s", "templates/fpm/deployment.yaml")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// ensure that the web service is created and has got 2 ports
	output, err = helmTemplate(ConvertOptions{
		OutputDir: "./chart",
	}, "-s", "templates/web/service.yaml")
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	service := corev1.Service{}
	if err := yaml.Unmarshal([]byte(output), &service); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if len(service.Spec.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(service.Spec.Ports))
	}
}

func TestDependsOn(t *testing.T) {
	compose_file := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80
        depends_on:
        - database

    database:
        image: mariadb:10.5
        ports:
        - 3306:3306
`
	tmpDir := setup(compose_file)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := _compile_test(t, "-s", "templates/web/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if len(dt.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(dt.Spec.Template.Spec.Containers))
	}
	// find an init container
	if len(dt.Spec.Template.Spec.InitContainers) != 1 {
		t.Errorf("Expected 1 init container, got %d", len(dt.Spec.Template.Spec.InitContainers))
	}
}
