package generator

import (
	"fmt"
	"katenary/generator/labels"
	"os"
	"testing"

	"github.com/compose-spec/compose-go/types"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestEnvInConfigMap(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        environment:
        - FOO=bar
        - BAR=baz
`
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web/configmap.yaml")
	configMap := v1.ConfigMap{}
	if err := yaml.Unmarshal([]byte(output), &configMap); err != nil {
		t.Errorf(unmarshalError, err)
	}
	data := configMap.Data
	if len(data) != 2 {
		t.Errorf("Expected 2 data, got %d", len(data))
	}
	if data["FOO"] != "bar" {
		t.Errorf("Expected FOO to be bar, got %s", data["FOO"])
	}
	if data["BAR"] != "baz" {
		t.Errorf("Expected BAR to be baz, got %s", data["BAR"])
	}
}

func TestMapEnv(t *testing.T) {
	composeFile := `
services:
  web:
    image: nginx:1.29
    environment:
      FOO: bar
    labels:
      %[1]s/map-env: |-
        FOO: 'baz'
`

	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web/configmap.yaml")
	configMap := v1.ConfigMap{}
	if err := yaml.Unmarshal([]byte(output), &configMap); err != nil {
		t.Errorf(unmarshalError, err)
	}
	data := configMap.Data
	if v, ok := data["FOO"]; !ok || v != "baz" {
		t.Errorf("Expected FOO to be baz, got %s", v)
	}
}

func TestAppendBadFile(t *testing.T) {
	cm := NewConfigMap(types.ServiceConfig{}, "app", true)
	err := cm.AppendFile("foo")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestAppendBadDir(t *testing.T) {
	cm := NewConfigMap(types.ServiceConfig{}, "app", true)
	err := cm.AppendDir("foo")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
