package generator

import (
	"katenary/parser"
	"log"
	"os"
	"os/exec"
	"testing"
)

const unmarshalError = "Failed to unmarshal the output: %s"

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

func internalCompileTest(t *testing.T, options ...string) string {
	_, err := parser.Parse(nil, nil, "compose.yml")
	if err != nil {
		t.Fatalf("Failed to parse the project: %s", err)
	}

	force := false
	outputDir := "./chart"
	profiles := make([]string, 0)
	helmdepUpdate := true
	var appVersion *string
	chartVersion := "0.1.0"
	convertOptions := ConvertOptions{
		Force:        force,
		OutputDir:    outputDir,
		Profiles:     profiles,
		HelmUpdate:   helmdepUpdate,
		AppVersion:   appVersion,
		ChartVersion: chartVersion,
	}
	Convert(convertOptions, "compose.yml")

	// launch helm lint to check the generated chart
	if helmLint(convertOptions) != nil {
		t.Errorf("Failed to lint the generated chart")
	}
	// try with helm template
	var output string
	if output, err = helmTemplate(convertOptions, options...); err != nil {
		t.Errorf("Failed to template the generated chart")
		t.Fatalf("Output %s", output)
	}
	return output
}

func helmTemplate(options ConvertOptions, arguments ...string) (string, error) {
	args := []string{"template", options.OutputDir}
	args = append(args, arguments...)

	cmd := exec.Command("helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
