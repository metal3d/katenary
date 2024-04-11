package generator

import (
	"os/exec"
	"testing"

	"katenary/parser"
)

type DeploymentTest struct {
	Spec struct {
		Replicas int `yaml:"replicas"`
		Template struct {
			Spec struct {
				Containers []struct {
					Image string `yaml:"image"`
				} `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

func _compile_test(t *testing.T) string {
	_, err := parser.Parse(nil, "compose.yml")
	if err != nil {
		t.Errorf("Failed to parse the project: %s", err)
	}

	force := false
	outputDir := "./chart"
	profiles := make([]string, 0)
	helmdepUpdate := false
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
	if output, err = helmTemplate(convertOptions); err != nil {
		t.Errorf("Failed to template the generated chart")
		t.Errorf("Output: %s", output)
	}
	return output
}

func helmTemplate(options ConvertOptions) (string, error) {
	cmd := exec.Command("helm", "template", options.OutputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
