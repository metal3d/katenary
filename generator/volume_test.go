package generator

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"katenary/generator/labels"
	"log"
	"os"
	"path/filepath"
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

func TestBinaryMount(t *testing.T) {
	composeFile := `
services:
  web:
    image: nginx
    volumes:
      - ./images/foo.png:/var/www/foo
    labels:
      %[1]s/configmap-files: |-
        - ./images/foo.png
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	log.Println(tmpDir)
	defer teardown(tmpDir)

	os.Mkdir(filepath.Join(tmpDir, "images"), 0o755)

	// create a png image
	pngFile := tmpDir + "/images/foo.png"
	w, h := 100, 100
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	red := color.RGBA{255, 0, 0, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, red)
		}
	}

	blue := color.RGBA{0, 0, 255, 255}
	for y := 30; y < 70; y++ {
		for x := 30; x < 70; x++ {
			img.Set(x, y, blue)
		}
	}
	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	f, err := os.Create(pngFile)
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, img)
	f.Close()
	output := internalCompileTest(t, "-s", "templates/web/deployment.yaml")
	d := v1.Deployment{}
	yaml.Unmarshal([]byte(output), &d)
	volumes := d.Spec.Template.Spec.Volumes
	if len(volumes) != 1 {
		t.Errorf("Expected 1 volume, got %d", len(volumes))
	}

	cm := corev1.ConfigMap{}
	cmContent, err := helmTemplate(ConvertOptions{
		OutputDir: "chart",
	}, "-s", "templates/web/statics/images/configmap.yaml")
	yaml.Unmarshal([]byte(cmContent), &cm)
	if im, ok := cm.BinaryData["foo.png"]; !ok {
		t.Errorf("Expected foo.png to be in the configmap")
	} else {
		if len(im) == 0 {
			t.Errorf("Expected image to be non-empty")
		}
	}
}

func TestGloballyBinaryMount(t *testing.T) {
	composeFile := `
services:
  web:
    image: nginx
    volumes:
      - ./images:/var/www/foo
    labels:
      %[1]s/configmap-files: |-
        - ./images
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	log.Println(tmpDir)
	defer teardown(tmpDir)

	os.Mkdir(filepath.Join(tmpDir, "images"), 0o755)

	// create a png image
	pngFile := tmpDir + "/images/foo.png"
	w, h := 100, 100
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	red := color.RGBA{255, 0, 0, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, red)
		}
	}

	blue := color.RGBA{0, 0, 255, 255}
	for y := 30; y < 70; y++ {
		for x := 30; x < 70; x++ {
			img.Set(x, y, blue)
		}
	}
	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	f, err := os.Create(pngFile)
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, img)
	f.Close()
	output := internalCompileTest(t, "-s", "templates/web/deployment.yaml")
	d := v1.Deployment{}
	yaml.Unmarshal([]byte(output), &d)
	volumes := d.Spec.Template.Spec.Volumes
	if len(volumes) != 1 {
		t.Errorf("Expected 1 volume, got %d", len(volumes))
	}

	cm := corev1.ConfigMap{}
	cmContent, err := helmTemplate(ConvertOptions{
		OutputDir: "chart",
	}, "-s", "templates/web/statics/images/configmap.yaml")
	yaml.Unmarshal([]byte(cmContent), &cm)
	if im, ok := cm.BinaryData["foo.png"]; !ok {
		t.Errorf("Expected foo.png to be in the configmap")
	} else {
		if len(im) == 0 {
			t.Errorf("Expected image to be non-empty")
		}
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

func TestExchangeVolume(t *testing.T) {
	composeFile := `
services:
  app1:
    image: nginx:1.29
    labels:
      %[1]s/exchange-volumes: |-
        - name: data
          mountPath: /var/www
  app2:
    image: foo:bar
    labels:
      %[1]s/same-pod: app1
      %[1]s/exchange-volumes: |-
        - name: data
          mountPath: /opt
          init: cp -r /var/www /opt
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)
	output := internalCompileTest(t, "-s", "templates/app1/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}
	// the deployment should have a volume named "data"
	volumes := dt.Spec.Template.Spec.Volumes
	found := false
	for v := range volumes {
		if volumes[v].Name == "exchange-data" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected volume name to be data: %v", volumes)
	}
	mounted := 0
	// we should have a volume mount for both containers
	containers := dt.Spec.Template.Spec.Containers
	for c := range containers {
		for _, vm := range containers[c].VolumeMounts {
			if vm.Name == "exchange-data" {
				mounted++
			}
		}
	}
	if mounted != 2 {
		t.Errorf("Expected 2 mounted volumes, got %d", mounted)
	}
}
