package generator

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	// we build on "devel" branch
	v := GetVersion()
	// by default, the version comes from build info and it's a development version
	if !strings.Contains(v, "(devel)") {
		t.Errorf("Expected version to be set, got %s", v)
	}

	// now, imagine we are on a release branch
	Version = "1.0.0"
	v = GetVersion()
	if !strings.Contains(v, "1.0.0") {
		t.Errorf("Expected version to be set, got %s", v)
	}
	// now, imagine we are on v1.0.0
	Version = "v1.0.0"
	v = GetVersion()
	if !strings.Contains(v, "v1.0.0") {
		t.Errorf("Expected version to be set, got %s", v)
	}
	// we can also compile a release branch
	Version = "release-1.0.0"
	v = GetVersion()
	if !strings.Contains(v, "release-1.0.0") {
		t.Errorf("Expected version to be set, got %s", v)
	}
}
