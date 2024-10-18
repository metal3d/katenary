package generator

import (
	"os"
	"testing"

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
