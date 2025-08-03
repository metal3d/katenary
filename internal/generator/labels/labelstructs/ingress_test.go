package labelstructs

import "testing"

func TestIngressLabel(t *testing.T) {
	ts := "\nhostname: example.com\npath: /\nenabled: true\nport: 8888"
	tc, err := IngressFrom(ts)
	if err != nil {
		t.Errorf("Error parsing IngressLabel: %v", err)
	}
	if tc.Hostname != "example.com" {
		t.Errorf("Expected IngressLabel to contain 'example.com', got %s", tc.Hostname)
	}
	if tc.Path == nil || *tc.Path != "/" {
		t.Errorf("Expected IngressLabel to contain '/', got %v", tc.Path)
	}
	if tc.Enabled != true {
		t.Errorf("Expected IngressLabel to be enabled, got %v", tc.Enabled)
	}
	if tc.Port == nil || *tc.Port != 8888 {
		t.Errorf("Expected IngressLabel to have port 8888, got %d", tc.Port)
	}
}

func TestIngressLabelNoPort(t *testing.T) {
	ts := "\nhostname: example.com\npath: /\nenabled: true"
	_, err := IngressFrom(ts)
	if err == nil {
		t.Errorf("Expected error when parsing IngressLabel without port, got nil")
	}
}
