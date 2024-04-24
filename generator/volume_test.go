package generator

import (
	"fmt"
	"os"
	"testing"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestGenerateWithBoundVolume(t *testing.T) {
	_compose_file := `
services:
    web:
        image: nginx:1.29
        volumes:
        - data:/var/www
volumes:
    data:
`
	tmpDir := setup(_compose_file)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := _compile_test(t, "-s", "templates/web/deployment.yaml")

	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if dt.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name != "data" {
		t.Errorf("Expected volume name to be data: %v", dt)
	}
}

func TestWithStaticFiles(t *testing.T) {
	_compose_file := `
services:
    web:
        image: nginx:1.29
        volumes:
        - ./static:/var/www
        labels:
            %s/configmap-files: |-
                - ./static
`
	_compose_file = fmt.Sprintf(_compose_file, katenaryLabelPrefix)
	tmpDir := setup(_compose_file)
	defer teardown(tmpDir)

	// create a static directory with an index.html file
	staticDir := tmpDir + "/static"
	os.Mkdir(staticDir, 0o755)
	indexFile, err := os.Create(staticDir + "/index.html")
	if err != nil {
		t.Errorf("Failed to create index.html: %s", err)
	}
	indexFile.WriteString("<html><body><h1>Hello, World!</h1></body></html>")
	indexFile.Close()

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := _compile_test(t, "-s", "templates/web/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}
	// get the volume mount path
	volumeMountPath := dt.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath
	if volumeMountPath != "/var/www" {
		t.Errorf("Expected volume mount path to be /var/www, got %s", volumeMountPath)
	}

	// read the configMap
	output, err = helmTemplate(ConvertOptions{
		OutputDir: tmpDir + "/chart",
	}, "-s", "templates/web/statics/static/configmap.yaml")
	if err != nil {
		t.Errorf("Failed to run helm template: %s", err)
	}
	configMap := corev1.ConfigMap{}
	if err := yaml.Unmarshal([]byte(output), &configMap); err != nil {
		t.Errorf(unmarshalError, err)
	}
	data := configMap.Data
	if len(data) != 1 {
		t.Errorf("Expected 1 data, got %d", len(data))
	}
	if data["index.html"] != "<html><body><h1>Hello, World!</h1></body></html>" {
		t.Errorf("Expected index.html to be <html><body><h1>Hello, World!</h1></body></html>, got %s", data["index.html"])
	}
}
