package katenaryfile

import (
	"context"
	"katenary/generator/labels"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/cli"
)

func TestBuildSchema(t *testing.T) {
	sh := GenerateSchema()
	if len(sh) == 0 {
		t.Errorf("Expected schema to be defined")
	}
}

func TestOverrideProjectWithKatenaryFile(t *testing.T) {
	composeContent := `
services:
  webapp:
    image: nginx:latest
`

	katenaryfileContent := `
webapp:
  ports:
    - 80
`

	// create /tmp/katenary-test-override directory, save the compose.yaml file
	tmpDir, err := os.MkdirTemp("", "katenary-test-override")
	if err != nil {
		t.Fatal(err.Error())
	}
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	katenaryFile := filepath.Join(tmpDir, "katenary.yaml")

	os.MkdirAll(tmpDir, 0755)
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Log(err)
	}
	if err := os.WriteFile(katenaryFile, []byte(katenaryfileContent), 0644); err != nil {
		t.Log(err)
	}
	defer os.RemoveAll(tmpDir)

	c, _ := os.ReadFile(composeFile)
	log.Println(string(c))

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

	OverrideWithConfig(project)
	w := project.Services["webapp"].Labels
	if v, ok := w[labels.LabelPorts]; !ok {
		t.Fatal("Expected ports to be defined", v)
	}
}

func TestOverrideProjectWithIngress(t *testing.T) {
	composeContent := `
services:
  webapp:
    image: nginx:latest
`

	katenaryfileContent := `
webapp:
  ports:
    - 80
  ingress:
    port: 80
`

	// create /tmp/katenary-test-override directory, save the compose.yaml file
	tmpDir, err := os.MkdirTemp("", "katenary-test-override")
	if err != nil {
		t.Fatal(err.Error())
	}
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	katenaryFile := filepath.Join(tmpDir, "katenary.yaml")

	os.MkdirAll(tmpDir, 0755)
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Log(err)
	}
	if err := os.WriteFile(katenaryFile, []byte(katenaryfileContent), 0644); err != nil {
		t.Log(err)
	}
	defer os.RemoveAll(tmpDir)

	c, _ := os.ReadFile(composeFile)
	log.Println(string(c))

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

	OverrideWithConfig(project)
	log.Println(project.Services["webapp"].Labels)
	w := project.Services["webapp"].Labels
	if v, ok := w[labels.LabelPorts]; !ok {
		t.Fatal("Expected ports to be defined", v)
	}
	if v, ok := w[labels.LabelIngress]; !ok {
		t.Fatal("Expected ingress to be defined", v)
	}
}
