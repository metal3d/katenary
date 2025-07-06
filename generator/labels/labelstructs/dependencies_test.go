package labelstructs

import "testing"

func TestDependenciesLabel(t *testing.T) {
	ts := "- name: mongodb"
	tc, _ := DependenciesFrom(ts)
	if len(tc) != 1 {
		t.Errorf("Expected DependenciesLabel to have 1 item, got %d", len(tc))
	}
	if tc[0].Name != "mongodb" {
		t.Errorf("Expected DependenciesLabel to contain 'mongodb', got %s", tc[0].Name)
	}
}
