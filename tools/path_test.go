package tools

import (
	"katenary/compose"
	"testing"
)

func Test_PathToName(t *testing.T) {
	path := compose.GetCurrentDir() + "/envéfoo.file"
	name := PathToName(path)
	if name != "env-foo-file" {
		t.Error("Expected env-foo-file, got ", name)
	}
}
