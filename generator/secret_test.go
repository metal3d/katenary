package generator

import (
	"fmt"
	"os"
	"testing"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestCreateSecretFromEnvironment(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        environment:
        - FOO=bar
        - BAR=baz
        labels:
            %s/secrets: |-
                - BAR
`
	composeFile = fmt.Sprintf(composeFile, katenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web/secret.yaml")
	secret := v1.Secret{}
	if err := yaml.Unmarshal([]byte(output), &secret); err != nil {
		t.Errorf(unmarshalError, err)
	}
	data := secret.Data
	if len(data) != 1 {
		t.Errorf("Expected 1 data, got %d", len(data))
	}
	// v1.Secret.Data is decoded, no problem
	if string(data["BAR"]) != "baz" {
		t.Errorf("Expected BAR to be baz, got %s", data["BAR"])
	}
}
