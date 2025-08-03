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

	output := internalCompileTest(t, "-s", "templates/web/service.yaml")
	service := v1.Service{}
	if err := yaml.Unmarshal([]byte(output), &service); err != nil {
		t.Errorf(unmarshalError, err)
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

func TestWithSeveralUnknownPorts(t *testing.T) {
	composeFile := `
services:
  multi:
    image: nginx
    ports:
      - 12443
      - 12480
    labels:
      katenary.v3/ingress: |-
        port: 12443
`
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/multi/service.yaml")
	service := v1.Service{}
	if err := yaml.Unmarshal([]byte(output), &service); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if len(service.Spec.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(service.Spec.Ports))
	}
	// ensure that both port names are different
	if service.Spec.Ports[0].Name == service.Spec.Ports[1].Name {
		t.Errorf("Expected different port names, got %s and %s", service.Spec.Ports[0].Name, service.Spec.Ports[1].Name)
	}
}
