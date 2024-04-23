package generator

import (
	"fmt"
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
            %singress: |-
                host: my.test.tld
                port: 80
`
	composeFile = fmt.Sprintf(composeFile, KATENARY_PREFIX)
	tmpDir := setup(composeFile)
	defer teardown(tmpDir)

	currentDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(currentDir)

	output := _compile_test(t, "-s", "templates/web/ingress.yaml", "--set", "web.ingress.enabled=true")
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
