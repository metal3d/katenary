package tools

import (
	"katenary/compose"
	"regexp"
	"strings"
)

// replaceChars replaces some chars in a string.
const replaceChars = `[^a-zA-Z0-9_]+`

// GetRelPath return the relative path from the root of the project.
func GetRelPath(path string) string {
	return strings.Replace(path, compose.GetCurrentDir(), ".", 1)
}

// PathToName transform a path to a yaml name.
func PathToName(path string) string {
	path = strings.TrimPrefix(GetRelPath(path), "./")
	path = regexp.MustCompile(replaceChars).ReplaceAllString(path, "-")
	if len(path) > 0 && path[0] == '-' {
		path = path[1:]
	}
	return path
}
