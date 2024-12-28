package generator

import (
	"context"
	"fmt"
	"katenary/generator/labels"
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/cli"
)

func TestSplitPorts(t *testing.T) {
	composeFileContent := `
services:
  foo:
    image: nginx:latest
    labels:
      %[1]s/ports: 80,443
`
	composeFileContent = fmt.Sprintf(composeFileContent, labels.KatenaryLabelPrefix)
	tmpDir, err := os.MkdirTemp("", "katenary-test-override")
	if err != nil {
		t.Fatal(err.Error())
	}
	composeFile := filepath.Join(tmpDir, "compose.yaml")

	os.MkdirAll(tmpDir, 0755)
	if err := os.WriteFile(composeFile, []byte(composeFileContent), 0644); err != nil {
		t.Log(err)
	}
	defer os.RemoveAll(tmpDir)

	// chand dir to this directory
	os.Chdir(tmpDir)
	options, _ := cli.NewProjectOptions(nil,
		cli.WithWorkingDirectory(tmpDir),
		cli.WithDefaultConfigPath,
	)
	project, err := cli.ProjectFromOptions(context.TODO(), options)
	if err != nil {
		t.Fatal(err)
	}
	service := project.Services["foo"]
	if err := fixPorts(&service); err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	project.Services["foo"] = service
	found := 0
	t.Log(service.Ports, project.Services["foo"].Ports)
	for _, p := range project.Services["foo"].Ports {
		switch p.Target {
		case 80, 443:
			found++
		}
	}
	if found != 2 {
		t.Errorf("Expected 2 ports, got %d", found)
	}
}

func TestSplitPortsWithDefault(t *testing.T) {
	composeFileContent := `
services:
  foo:
    image: nginx:latest
    ports:
      - 8080
    labels:
      %[1]s/ports: 80,443
`
	composeFileContent = fmt.Sprintf(composeFileContent, labels.KatenaryLabelPrefix)
	tmpDir, err := os.MkdirTemp("", "katenary-test-override")
	if err != nil {
		t.Fatal(err)
	}
	composeFile := filepath.Join(tmpDir, "compose.yaml")

	os.MkdirAll(tmpDir, 0755)
	if err := os.WriteFile(composeFile, []byte(composeFileContent), 0644); err != nil {
		t.Log(err)
	}
	defer os.RemoveAll(tmpDir)

	// chand dir to this directory
	os.Chdir(tmpDir)
	options, _ := cli.NewProjectOptions(nil,
		cli.WithWorkingDirectory(tmpDir),
		cli.WithDefaultConfigPath,
	)
	project, err := cli.ProjectFromOptions(context.TODO(), options)
	if err != nil {
		t.Fatal(err)
	}
	service := project.Services["foo"]
	if err := fixPorts(&service); err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	found := 0
	for _, p := range service.Ports {
		switch p.Target {
		case 80, 443, 8080:
			found++
		}
	}
	if found != 3 {
		t.Errorf("Expected 3 ports, got %d", found)
	}
}
