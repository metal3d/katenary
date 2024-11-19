package generator

import (
	"katenary/generator/labels"
	"regexp"
)

var (
	// find all labels starting by __replace_ and ending with ":"
	// and get the value between the quotes
	// ?s => multiline
	// (?P<inc>.+?) => named capture group to "inc" variable (so we could use $inc in the replace)
	replaceLabelRegexp = regexp.MustCompile(`(?s)__replace_.+?: '(?P<inc>.+?)'`)

	// Standard annotationss
	Annotations = map[string]string{
		labels.LabelName("version"): Version,
	}
)
