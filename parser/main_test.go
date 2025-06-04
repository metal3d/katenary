package parser

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

const composeFile = `
services:
  app:
    image: nginx:latest
`

func setupTest() (string, error) {
	// write the composeFile to a temporary file
	tmpDir, err := os.MkdirTemp("", "katenary-test-parse")
	if err != nil {
		return "", err
	}
	writeFile := filepath.Join(tmpDir, "compose.yaml")
	writeErr := os.WriteFile(writeFile, []byte(composeFile), 0644)
	return writeFile, writeErr
}

func tearDownTest(tmpDir string) {
	if tmpDir != "" {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Fatalf("Failed to remove temporary directory %s: %s", tmpDir, err.Error())
		}
	}
}

func TestParse(t *testing.T) {
	file, err := setupTest()
	dirname := filepath.Dir(file)
	currentDir, _ := os.Getwd()
	if err := os.Chdir(dirname); err != nil {
		t.Fatalf("Failed to change directory to %s: %s", dirname, err.Error())
	}
	defer func() {
		tearDownTest(dirname)
		if err := os.Chdir(currentDir); err != nil {
			t.Fatalf("Failed to change back to original directory %s: %s", currentDir, err.Error())
		}
	}()

	if err != nil {
		t.Fatalf("Failed to setup test: %s", err.Error())
	}

	Project, err := Parse(nil, nil)
	if err != nil {
		t.Fatalf("Failed to parse compose file: %s", err.Error())
	}
	if Project == nil {
		t.Fatal("Expected project to be not nil")
	}
}
