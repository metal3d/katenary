package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"katenary/generator/labelStructs"
	"katenary/utils"
)

// ConvertOptions are the options to convert a compose project to a helm chart.
type ConvertOptions struct {
	AppVersion   *string
	OutputDir    string
	ChartVersion string
	Profiles     []string
	Force        bool
	HelmUpdate   bool
}

// ChartTemplate is a template of a chart. It contains the content of the template and the name of the service.
// This is used internally to generate the templates.
type ChartTemplate struct {
	Servicename string
	Content     []byte
}

// HelmChart is a Helm Chart representation. It contains all the
// tempaltes, values, versions, helpers...
type HelmChart struct {
	Templates    map[string]*ChartTemplate `yaml:"-"`
	Values       map[string]any            `yaml:"-"`
	VolumeMounts map[string]any            `yaml:"-"`
	composeHash  *string                   `yaml:"-"`
	Name         string                    `yaml:"name"`
	ApiVersion   string                    `yaml:"apiVersion"`
	Version      string                    `yaml:"version"`
	AppVersion   string                    `yaml:"appVersion"`
	Description  string                    `yaml:"description"`
	Helper       string                    `yaml:"-"`
	Dependencies []labelStructs.Dependency `yaml:"dependencies,omitempty"`
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

// SaveTemplates the templates of the chart to the given directory.
func (chart *HelmChart) SaveTemplates(templateDir string) {
	for name, template := range chart.Templates {
		t := template.Content
		t = removeNewlinesInsideBrackets(t)
		t = removeUnwantedLines(t)
		t = addModeline(t)

		kind := utils.GetKind(name)
		var icon utils.Icon
		switch kind {
		case "deployment":
			icon = utils.IconPackage
		case "service":
			icon = utils.IconPlug
		case "ingress":
			icon = utils.IconWorld
		case "volumeclaim":
			icon = utils.IconCabinet
		case "configmap":
			icon = utils.IconConfig
		case "secret":
			icon = utils.IconSecret
		default:
			icon = utils.IconInfo
		}

		servicename := template.Servicename
		if err := os.MkdirAll(filepath.Join(templateDir, servicename), 0o755); err != nil {
			fmt.Println(utils.IconFailure, err)
			os.Exit(1)
		}
		fmt.Println(icon, "Creating", kind, servicename)
		// if the name is a path, create the directory
		if strings.Contains(name, string(filepath.Separator)) {
			name = filepath.Join(templateDir, name)
			err := os.MkdirAll(filepath.Dir(name), 0o755)
			if err != nil {
				fmt.Println(utils.IconFailure, err)
				os.Exit(1)
			}
		} else {
			// remove the serivce name from the template name
			name = strings.Replace(name, servicename+".", "", 1)
			name = filepath.Join(templateDir, servicename, name)
		}
		f, err := os.Create(name)
		if err != nil {
			fmt.Println(utils.IconFailure, err)
			os.Exit(1)
		}

		f.Write(t)
		f.Close()
	}
}
