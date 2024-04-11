package generator

import (
	"log"
	"os"
	"testing"

	"sigs.k8s.io/yaml"
)

func setup(content string) string {
	// write the _compose_file in temporary directory
	tmpDir, err := os.MkdirTemp("", "katenary")
	if err != nil {
		panic(err)
	}
	os.WriteFile(tmpDir+"/compose.yml", []byte(content), 0o644)
	return tmpDir
}

func teardown(tmpDir string) {
	// remove the temporary directory
	log.Println("Removing temporary directory: ", tmpDir)
	if err := os.RemoveAll(tmpDir); err != nil {
		panic(err)
	}
}

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

	output := _compile_test(t)

	dt := DeploymentTest{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf("Failed to unmarshal the output: %s", err)
	}

	if dt.Spec.Replicas != 1 {
		t.Errorf("Expected replicas to be 1, got %d", dt.Spec.Replicas)
		t.Errorf("Output: %s", output)
	}

	if dt.Spec.Template.Spec.Containers[0].Image != "nginx:1.29" {
		t.Errorf("Expected image to be nginx:1.29, got %s", dt.Spec.Template.Spec.Containers[0].Image)
	}
}
