package generator

import (
	"os"
	"testing"

	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/yaml"
)

func TestGenerate(t *testing.T) {
	_compose_file := `
services:
    web:
        image: nginx:1.29
`
	tmpDir := setup(_compose_file)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := _compile_test(t, "-s", "templates/web/deployment.yaml")

	// dt := DeploymentTest{}
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf("Failed to unmarshal the output: %s", err)
	}

	if *dt.Spec.Replicas != 1 {
		t.Errorf("Expected replicas to be 1, got %d", dt.Spec.Replicas)
		t.Errorf("Output: %s", output)
	}

	if dt.Spec.Template.Spec.Containers[0].Image != "nginx:1.29" {
		t.Errorf("Expected image to be nginx:1.29, got %s", dt.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestGenerateWithBoundVolume(t *testing.T) {
	_compose_file := `
services:
    web:
        image: nginx:1.29
        volumes:
        - data:/var/www
volumes:
    data:
`
	tmpDir := setup(_compose_file)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := _compile_test(t, "-s", "templates/web/deployment.yaml")

	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf("Failed to unmarshal the output: %s", err)
	}

	if dt.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name != "data" {
		t.Errorf("Expected volume name to be data: %v", dt)
	}
}
