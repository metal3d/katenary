package generator

import (
	"fmt"
	"github.com/katenary/katenary/internal/generator/labels"
	"os"
	"testing"

	v1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/yaml"
)

func TestSimpleIngress(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80
        - 443:443
        labels:
            %s/ingress: |-
                hostname: my.test.tld
                port: 80
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(
		t,
		"-s", "templates/web/ingress.yaml",
		"--set", "web.ingress.enabled=true",
	)
	ingress := v1.Ingress{}
	if err := yaml.Unmarshal([]byte(output), &ingress); err != nil {
		t.Errorf(unmarshalError, err)
	}
	if len(ingress.Spec.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(ingress.Spec.Rules))
	}
	if ingress.Spec.Rules[0].Host != "my.test.tld" {
		t.Errorf("Expected host to be my.test.tld, got %s", ingress.Spec.Rules[0].Host)
	}
}

func TestTLS(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80
        - 443:443
        labels:
            %s/ingress: |-
                hostname: my.test.tld
                port: 80
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(
		t,
		"-s", "templates/web/ingress.yaml",
		"--set", "web.ingress.enabled=true",
	)
	ingress := v1.Ingress{}
	if err := yaml.Unmarshal([]byte(output), &ingress); err != nil {
		t.Errorf(unmarshalError, err)
	}
	// find the tls section
	tls := ingress.Spec.TLS
	if len(tls) != 1 {
		t.Errorf("Expected 1 tls section, got %d", len(tls))
	}
}

func TestTLSName(t *testing.T) {
	composeFile := `
services:
    web:
        image: nginx:1.29
        ports:
        - 80:80
        - 443:443
        labels:
            %s/ingress: |-
                hostname: my.test.tld
                port: 80
`
	composeFile = fmt.Sprintf(composeFile, labels.KatenaryLabelPrefix)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := internalCompileTest(
		t,
		"-s",
		"templates/web/ingress.yaml",
		"--set", "web.ingress.enabled=true",
		"--set", "web.ingress.tls.secretName=mysecret",
	)
	ingress := v1.Ingress{}
	if err := yaml.Unmarshal([]byte(output), &ingress); err != nil {
		t.Errorf(unmarshalError, err)
	}
	// find the tls section
	tls := ingress.Spec.TLS
	if len(tls) != 1 {
		t.Errorf("Expected 1 tls section, got %d", len(tls))
	}
	if tls[0].SecretName != "mysecret" {
		t.Errorf("Expected secretName to be mysecret, got %s", tls[0].SecretName)
	}
}
