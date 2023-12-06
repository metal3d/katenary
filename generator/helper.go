package generator

import (
	_ "embed"
	"strings"
)

// helmHelper is a template for the _helpers.tpl file in the chart templates directory.
//
//go:embed helmHelper.tpl
var helmHelper string

// Helper returns the _helpers.tpl file for a chart.
func Helper(name string) string {
	helmHelper := strings.ReplaceAll(helmHelper, "__APP__", name)
	helmHelper = strings.ReplaceAll(helmHelper, "__PREFIX__", KATENARY_PREFIX)
	helmHelper = strings.ReplaceAll(helmHelper, "__VERSION__", "0.1.0")
	return helmHelper
}
