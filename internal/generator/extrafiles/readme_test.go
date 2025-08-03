package extrafiles

import (
	"regexp"
	"testing"
)

func TestReadMeFile_Basic(t *testing.T) {
	values := map[string]any{
		"replicas": 2,
		"image": map[string]any{
			"repository": "nginx",
			"tag":        "latest",
		},
	}

	result := ReadMeFile("testchart", "A test chart", values)
	t.Logf("Generated README content:\n%s", result)
	paramerRegExp := regexp.MustCompile(`\|\s+` + "`" + `(.*?)` + "`" + `\s+\|\s+` + "`" + `(.*?)` + "`" + `\s+\|`)
	matches := paramerRegExp.FindAllStringSubmatch(result, -1)
	if len(matches) != 3 {
		t.Errorf("Expected 5 lines in the table for headers and parameters, got %d", len(matches))
	}
	if matches[0][1] != "image.repository" || matches[0][2] != "nginx" {
		t.Errorf("Expected third line to be image.repository, got %s", matches[1])
	}
	if matches[1][1] != "image.tag" || matches[1][2] != "latest" {
		t.Errorf("Expected fourth line to be image.tag, got %s", matches[2])
	}
	if matches[2][1] != "replicas" || matches[2][2] != "2" {
		t.Errorf("Expected second line to be replicas, got %s", matches[0])
	}
}
