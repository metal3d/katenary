package generator

import "katenary/generator/labelStructs"

// ChartTemplate is a template of a chart. It contains the content of the template and the name of the service.
// This is used internally to generate the templates.
//
// TODO: maybe we can set it private.
type ChartTemplate struct {
	Content     []byte
	Servicename string
}

// HelmChart is a Helm Chart representation. It contains all the
// tempaltes, values, versions, helpers...
type HelmChart struct {
	Name         string                    `yaml:"name"`
	ApiVersion   string                    `yaml:"apiVersion"`
	Version      string                    `yaml:"version"`
	AppVersion   string                    `yaml:"appVersion"`
	Description  string                    `yaml:"description"`
	Dependencies []labelStructs.Dependency `yaml:"dependencies,omitempty"`
	Templates    map[string]*ChartTemplate `yaml:"-"` // do not export to yaml
	Helper       string                    `yaml:"-"` // do not export to yaml
	Values       map[string]any            `yaml:"-"` // do not export to yaml
	VolumeMounts map[string]any            `yaml:"-"` // do not export to yaml
	composeHash  *string                   `yaml:"-"` // do not export to yaml
}

// NewChart creates a new empty chart with the given name.
func NewChart(name string) *HelmChart {
	return &HelmChart{
		Name:        name,
		Templates:   make(map[string]*ChartTemplate, 0),
		Description: "A Helm chart for " + name,
		ApiVersion:  "v2",
		Version:     "",
		AppVersion:  "", // set to 0.1.0 by default if no "main-app" label is found
		Values: map[string]any{
			"pullSecrets": []string{},
		},
	}
}

// ConvertOptions are the options to convert a compose project to a helm chart.
type ConvertOptions struct {
	Force        bool     // Force the chart directory deletion if it already exists.
	OutputDir    string   // The output directory of the chart.
	Profiles     []string // Profile to use for the conversion.
	HelmUpdate   bool     // If true, the "helm dep update" command will be run after the chart generation.
	AppVersion   *string  // Set the chart "appVersion" field. If nil, the version will be set to 0.1.0.
	ChartVersion string   // Set the chart "version" field.
}
