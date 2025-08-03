package generator

import (
	"fmt"
	"github.com/katenary/katenary/internal/generator/labels"
	"os"
	"strings"
	"testing"

	yaml3 "gopkg.in/yaml.v3"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const webTemplateOutput = `templates/web/deployment.yaml`

func TestGenerate(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
`
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", webTemplateOutput)

	// dt := DeploymentTest{}
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if *dt.Spec.Replicas != 1 {
		t.Errorf("Expected replicas to be 1, got %d", dt.Spec.Replicas)
		t.Errorf("Output: %s", output)
	}

	if dt.Spec.Template.Spec.Containers[0].Image != "nginx:1.29" {
		t.Errorf("Expected image to be nginx:1.29, got %s", dt.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestGenerateOneDeploymentWithSamePod(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80

    fpm:
        image: php:fpm
        ports:
        - 9000:9000
        labels:
            katenary.v3/same-pod: web
`

	outDir := "./chart"
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", webTemplateOutput)
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if len(dt.Spec.Template.Spec.Containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(dt.Spec.Template.Spec.Containers))
	}
	// endsure that the fpm service is not created

	var err error
	_, err = helmTemplate(ConvertOptions{
		OutputDir: outDir,
	}, "-s", "templates/fpm/deployment.yaml")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// ensure that the web service is created and has got 2 ports
	output, err = helmTemplate(ConvertOptions{
		OutputDir: outDir,
	}, "-s", "templates/web/service.yaml")
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	service := corev1.Service{}
	if err := yaml.Unmarshal([]byte(output), &service); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if len(service.Spec.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(service.Spec.Ports))
	}
}

func TestDependsOn(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80
        depends_on:
        - database

    database:
        image: mariadb:10.5
        ports:
        - 3306:3306
`
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", webTemplateOutput)
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if len(dt.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(dt.Spec.Template.Spec.Containers))
	}
	// find an init container
	if len(dt.Spec.Template.Spec.InitContainers) != 1 {
		t.Errorf("Expected 1 init container, got %d", len(dt.Spec.Template.Spec.InitContainers))
	}
}

func TestHelmDependencies(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80

    mariadb:
        image: mariadb:10.5
        ports:
        - 3306:3306
        labels:
            %s/dependencies: |
                - name: mariadb
                  repository: oci://registry-1.docker.io/bitnamicharts
                  version: 18.x.X

    `
	composeFile = fmt.Sprintf(composeFile, labels.Prefix())
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", webTemplateOutput)
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	// ensure that there is no mariasb deployment
	_, err := helmTemplate(ConvertOptions{
		OutputDir: "./chart",
	}, "-s", "templates/mariadb/deployment.yaml")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// check that Chart.yaml has the dependency
	chart := HelmChart{}
	chartFile := "./chart/Chart.yaml"
	if _, err := os.Stat(chartFile); os.IsNotExist(err) {
		t.Errorf("Chart.yaml does not exist")
	}
	chartContent, err := os.ReadFile(chartFile)
	if err != nil {
		t.Errorf("Error reading Chart.yaml: %s", err)
	}
	if err := yaml.Unmarshal(chartContent, &chart); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if len(chart.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(chart.Dependencies))
	}
}

func TestLivenessProbesFromHealthCheck(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80
        healthcheck:
            test: ["CMD", "curl", "-f", "http://localhost"]
            interval: 5s
            timeout: 3s
            retries: 3
        `
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", webTemplateOutput)
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if dt.Spec.Template.Spec.Containers[0].LivenessProbe == nil {
		t.Errorf("Expected liveness probe to be set")
	}
}

func TestProbesFromLabels(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80
        labels:
            %s/health-check: |
                livenessProbe:
                    httpGet:
                        path: /healthz
                        port: 80
                readinessProbe:
                    httpGet:
                        path: /ready
                        port: 80
    `
	composeFile = fmt.Sprintf(composeFile, labels.Prefix())
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", webTemplateOutput)
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if dt.Spec.Template.Spec.Containers[0].LivenessProbe == nil {
		t.Errorf("Expected liveness probe to be set")
	}
	if dt.Spec.Template.Spec.Containers[0].ReadinessProbe == nil {
		t.Errorf("Expected readiness probe to be set")
	}
	t.Logf("LivenessProbe: %+v", dt.Spec.Template.Spec.Containers[0].LivenessProbe)

	// ensure that the liveness probe is set to /healthz
	if dt.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path != "/healthz" {
		t.Errorf("Expected liveness probe path to be /healthz, got %s", dt.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path)
	}

	// ensure that the readiness probe is set to /ready
	if dt.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path != "/ready" {
		t.Errorf("Expected readiness probe path to be /ready, got %s", dt.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path)
	}
}

func TestSetValues(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        environment:
            FOO: bar
            BAZ: qux
        labels:
            %s/values: |
                - FOO
`

	composeFile = fmt.Sprintf(composeFile, labels.Prefix())
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", webTemplateOutput)
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	// readh the values.yaml, we must have FOO in web environment but not BAZ
	valuesFile := "./chart/values.yaml"
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		t.Errorf("values.yaml does not exist")
	}
	valuesContent, err := os.ReadFile(valuesFile)
	if err != nil {
		t.Errorf("Error reading values.yaml: %s", err)
	}
	mapping := struct {
		Web struct {
			Environment map[string]string `yaml:"environment"`
		} `yaml:"web"`
	}{}
	if err := yaml3.Unmarshal(valuesContent, &mapping); err != nil {
		t.Errorf(unmarshalError, err)
	}

	if v, ok := mapping.Web.Environment["FOO"]; !ok {
		t.Errorf("Expected FOO in web environment")
		if v != "bar" {
			t.Errorf("Expected FOO to be bar, got %s", v)
		}
	}
	if v, ok := mapping.Web.Environment["BAZ"]; ok {
		t.Errorf("Expected BAZ not in web environment")
		if v != "qux" {
			t.Errorf("Expected BAZ to be qux, got %s", v)
		}
	}
}

func TestWithUnderscoreInContainerName(t *testing.T) {
	composeFile := `
services:
    web-app:
        image: nginx:1.29
        container_name: web_app_container
        environment:
            FOO: BAR
        labels:
            %s/values: |
                - FOO
`
	composeFile = fmt.Sprintf(composeFile, labels.Prefix())
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web_app/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}
	// find container.name
	containerName := dt.Spec.Template.Spec.Containers[0].Name
	if strings.Contains(containerName, "_") {
		t.Errorf("Expected container name to not contain underscores, got %s", containerName)
	}
}

func TestWithDashes(t *testing.T) {
	composeFile := `
services:
    web-app:
        image: nginx:1.29
        environment:
            FOO: BAR
        labels:
            %s/values: |
                - FOO
`

	composeFile = fmt.Sprintf(composeFile, labels.Prefix())
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web_app/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	valuesFile := "./chart/values.yaml"
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		t.Errorf("values.yaml does not exist")
	}
	valuesContent, err := os.ReadFile(valuesFile)
	if err != nil {
		t.Errorf("Error reading values.yaml: %s", err)
	}
	mapping := struct {
		Web struct {
			Environment map[string]string `yaml:"environment"`
		} `yaml:"web_app"`
	}{}
	if err := yaml3.Unmarshal(valuesContent, &mapping); err != nil {
		t.Errorf(unmarshalError, err)
	}

	// we must have FOO in web_app environment (not web-app)
	// this validates that the service name is converted to a valid k8s name
	if v, ok := mapping.Web.Environment["FOO"]; !ok {
		t.Errorf("Expected FOO in web_app environment")
		if v != "BAR" {
			t.Errorf("Expected FOO to be BAR, got %s", v)
		}
	}
}

func TestDashesWithValueFrom(t *testing.T) {
	composeFile := `
services:
    web-app:
        image: nginx:1.29
        environment:
            FOO: BAR
        labels:
            %[1]s/values: |
                - FOO
    web2:
        image: nginx:1.29
        labels:
            %[1]s/values-from: |
                BAR: web-app.FOO
`

	composeFile = fmt.Sprintf(composeFile, labels.Prefix())
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web2/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}

	valuesFile := "./chart/values.yaml"
	if _, err := os.Stat(valuesFile); os.IsNotExist(err) {
		t.Errorf("values.yaml does not exist")
	}
	valuesContent, err := os.ReadFile(valuesFile)
	if err != nil {
		t.Errorf("Error reading values.yaml: %s", err)
	}
	mapping := struct {
		Web struct {
			Environment map[string]string `yaml:"environment"`
		} `yaml:"web_app"`
	}{}
	if err := yaml3.Unmarshal(valuesContent, &mapping); err != nil {
		t.Errorf(unmarshalError, err)
	}

	// we must have FOO in web_app environment (not web-app)
	// this validates that the service name is converted to a valid k8s name
	if v, ok := mapping.Web.Environment["FOO"]; !ok {
		t.Errorf("Expected FOO in web_app environment")
		if v != "BAR" {
			t.Errorf("Expected FOO to be BAR, got %s", v)
		}
	}

	// ensure that the deployment has the value from the other service
	barenv := dt.Spec.Template.Spec.Containers[0].Env[0]
	if barenv.Value != "" {
		t.Errorf("Expected value to be empty")
	}
	if barenv.ValueFrom == nil {
		t.Errorf("Expected valueFrom to be set")
	}
}

func TestCheckCommand(t *testing.T) {
	composeFile := `
services:
    web-app:
        image: nginx:1.29
        command:
            - sh
            - -c
            - |-
                echo "Hello, World!"
                echo "Done"
`

	// composeFile = fmt.Sprintf(composeFile, labels.Prefix())
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(t, "-s", "templates/web_app/deployment.yaml")
	dt := v1.Deployment{}
	if err := yaml.Unmarshal([]byte(output), &dt); err != nil {
		t.Errorf(unmarshalError, err)
	}
	// find the command in the container
	command := dt.Spec.Template.Spec.Containers[0].Command
	if len(command) != 3 {
		t.Errorf("Expected command to have 3 elements, got %d", len(command))
	}
	if command[0] != "sh" || command[1] != "-c" {
		t.Errorf("Expected command to be 'sh -c', got %s", strings.Join(command, " "))
	}
}
