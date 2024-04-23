package generator

import (
	"os"
	"testing"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestBasicService(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80
        - 443:443
    `
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := _compile_test(t, "-s", "templates/web/service.yaml")
	service := v1.Service{}
	if err := yaml.Unmarshal([]byte(output), &service); err != nil {
		t.Errorf("Failed to unmarshal the output: %s", err)
	}

	if len(service.Spec.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(service.Spec.Ports))
	}

	foundPort := 0
	for _, port := range service.Spec.Ports {
		if port.Port == 80 && port.TargetPort.StrVal == "http" {
			foundPort++
		}
		if port.Port == 443 && port.TargetPort.StrVal == "https" {
			foundPort++
		}
	}
	if foundPort != 2 {
		t.Errorf("Expected 2 ports, got %d", foundPort)
	}
}
