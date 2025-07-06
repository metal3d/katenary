package labelstructs

import "testing"

func TestPortsFromLabel(t *testing.T) {
	data := "- 8080\n- 9090\n"
	expected := Ports{8080, 9090}

	ports, err := PortsFrom(data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(ports) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(ports))
	}

	for i, port := range ports {
		if port != expected[i] {
			t.Errorf("expected port %d at index %d, got %d", expected[i], i, port)
		}
	}
}
