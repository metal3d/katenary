package generator

import (
	"fmt"
	"katenary/generator/labels"
	"os"
	"testing"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const htmlContent = "<html><body><h1>Hello, World!</h1></body></html>"

func TestGenerateWithBoundVolume(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        volumes:
        - data:/var/www
volumes:
    data:
`
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web/deployment.yaml")

	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if dt.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name != "data" {
		t.Errorf("Expected volume name to be data: %v", dt)
	}
}

func TestWithStaticFiles(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        volumes:
        - ./static:/var/www
        labels:
            %s/configmap-files: |-
                - ./static
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	// create a static directory with an index.html file
	staticDir := tmpDir + "/static"
	os.Mkdir(staticDir, 0o755)
	indexFile, err := os.Create(staticDir + "/index.html")
	if err != nil {
		t.Errorf("Failed to create index.html: %s", err)
	}
	indexFile.WriteString(htmlContent)
	indexFile.Close()

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web/deployment.yaml")
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
	if data["index.html"] != htmlContent {
		t.Errorf("Expected index.html to be "+htmlContent+", got %s", data["index.html"])
	}
}

func TestWithFileMapping(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        volumes:
        - ./static/index.html:/var/www/index.html
        labels:
            %s/configmap-files: |-
                - ./static/index.html
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	// create a static directory with an index.html file
	staticDir := tmpDir + "/static"
	os.Mkdir(staticDir, 0o755)
	indexFile, err := os.Create(staticDir + "/index.html")
	if err != nil {
		t.Errorf("Failed to create index.html: %s", err)
	}
	indexFile.WriteString(htmlContent)
	indexFile.Close()

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	// get the volume mount path
	volumeMountPath := dt.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath
	if volumeMountPath != "/var/www/index.html" {
		t.Errorf("Expected volume mount path to be /var/www/index.html, got %s", volumeMountPath)
	}
	// but this time, we need a subpath
	subPath := dt.Spec.Template.Spec.Containers[0].VolumeMounts[0].SubPath
	if subPath != "index.html" {
		t.Errorf("Expected subpath to be index.html, got %s", subPath)
	}
}

func TestBindFrom(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        volumes:
        - data:/var/www

    fpm:
        image: php:fpm
        volumes:
        - data:/var/www
        labels:
            %[1]s/ports: |
                - 9000
            %[1]s/same-pod: web

volumes:
    data:
`

	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}
	// both containers should have the same volume mount
	if dt.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name != "data" {
		t.Errorf("Expected volume name to be data: %v", dt)
	}
	if dt.Spec.Template.Spec.Containers[1].VolumeMounts[0].Name != "data" {
		t.Errorf("Expected volume name to be data: %v", dt)
	}
}
