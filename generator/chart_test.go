package generator

import (
	"fmt"
	"katenary/generator/labels"
	"os"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestValuesFrom(t *testing.T) {
	composeFile := `
services:
  aa:
    image: nginx:latest
    environment:
      AA_USER: foo
  bb:
    image: nginx:latest
    labels:
      %[1]s/values-from: |-
        BB_USER: aa.USER
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/aa/configmap.yaml")
	configMap := v1.ConfigMap{}
	if err := yaml.Unmarshal([]byte(output), &configMap); err != nil {
		t.Errorf(unmarshalError, err)
	}
	data := configMap.Data
	if v, ok := data["AA_USER"]; !ok || v != "foo" {
		t.Errorf("Expected AA_USER to be foo, got %s", v)
	}
}

func TestValuesFromCopy(t *testing.T) {
	composeFile := `
services:
  aa:
    image: nginx:latest
    environment:
      AA_USER: foo
  bb:
    image: nginx:latest
    labels:
      %[1]s/values-from: |-
        BB_USER: aa.AA_USER
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/bb/deployment.yaml")
	dep := appsv1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dep); err != nil {
		t.Errorf(unmarshalError, err)
	}
	containers := dep.Spec.Template.Spec.Containers
	environment := containers[0].Env[0]

	envFrom := environment.ValueFrom.ConfigMapKeyRef
	if envFrom.Key != "AA_USER" {
		t.Errorf("Expected AA_USER, got %s", envFrom.Key)
	}
	if !strings.Contains(envFrom.Name, "aa") {
		t.Errorf("Expected aa, got %s", envFrom.Name)
	}
}

func TestValuesFromSecret(t *testing.T) {
	composeFile := `
services:
  aa:
    image: nginx:latest
    environment:
      AA_USER: foo
    labels:
      %[1]s/secrets: |-
        - AA_USER
  bb:
    image: nginx:latest
    labels:
      %[1]s/values-from: |-
        BB_USER: aa.AA_USER
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/bb/deployment.yaml")
	dep := appsv1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dep); err != nil {
		t.Errorf(unmarshalError, err)
	}
	containers := dep.Spec.Template.Spec.Containers
	environment := containers[0].Env[0]

	envFrom := environment.ValueFrom.SecretKeyRef
	if envFrom.Key != "AA_USER" {
		t.Errorf("Expected AA_USER, got %s", envFrom.Key)
	}
	if !strings.Contains(envFrom.Name, "aa") {
		t.Errorf("Expected aa, got %s", envFrom.Name)
	}
}

func TestEnvFrom(t *testing.T) {
	composeFile := `
services:
  web:
    image: nginx:1.29
    environment:
      Foo: bar
      BAZ: qux
  db:
    image: postgres
    labels:
      %[1]s/env-from: |-
        - web
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/db/deployment.yaml")
	dep := appsv1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dep); err != nil {
		t.Errorf(unmarshalError, err)
	}
	envFrom := dep.Spec.Template.Spec.Containers[0].EnvFrom
	if len(envFrom) != 1 {
		t.Fatalf("Expected 1 envFrom, got %d", len(envFrom))
	}
}
