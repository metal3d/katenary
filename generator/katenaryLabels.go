package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"

	"katenary/utils"

	"sigs.k8s.io/yaml"
)

var (
	// Set the documentation of labels here
	//
	//go:embed katenaryLabelsDoc.yaml
	labelFullHelpYAML []byte

	// parsed yaml
	labelFullHelp map[string]Help
)

// Label is a katenary label to find in compose files.
type Label = string

// Help is the documentation of a label.
type Help struct {
	Short   string `yaml:"short"`
	Long    string `yaml:"long"`
	Example string `yaml:"example"`
	Type    string `yaml:"type"`
}

const KATENARY_PREFIX = "katenary.v3/"

// Known labels.
const (
	LABEL_MAIN_APP     Label = KATENARY_PREFIX + "main-app"
	LABEL_VALUES       Label = KATENARY_PREFIX + "values"
	LABEL_SECRETS      Label = KATENARY_PREFIX + "secrets"
	LABEL_PORTS        Label = KATENARY_PREFIX + "ports"
	LABEL_INGRESS      Label = KATENARY_PREFIX + "ingress"
	LABEL_MAP_ENV      Label = KATENARY_PREFIX + "map-env"
	LABEL_HEALTHCHECK  Label = KATENARY_PREFIX + "health-check"
	LABEL_SAME_POD     Label = KATENARY_PREFIX + "same-pod"
	LABEL_DESCRIPTION  Label = KATENARY_PREFIX + "description"
	LABEL_IGNORE       Label = KATENARY_PREFIX + "ignore"
	LABEL_DEPENDENCIES Label = KATENARY_PREFIX + "dependencies"
	LABEL_CM_FILES     Label = KATENARY_PREFIX + "configmap-files"
	LABEL_CRONJOB      Label = KATENARY_PREFIX + "cronjob"
	LABEL_ENV_FROM     Label = KATENARY_PREFIX + "env-from"
)

func init() {
	if err := yaml.Unmarshal(labelFullHelpYAML, &labelFullHelp); err != nil {
		panic(err)
	}
}

// Generate the help for the labels.
func GetLabelHelp(asMarkdown bool) string {
	names := GetLabelNames() // sorted
	if !asMarkdown {
		return generatePlainHelp(names)
	}
	return generateMarkdownHelp(names)
}

func generatePlainHelp(names []string) string {
	var builder strings.Builder
	for _, name := range names {
		help := labelFullHelp[name]
		fmt.Fprintf(&builder, "%s%s:\t%s\t%s\n", KATENARY_PREFIX, name, help.Type, help.Short)
	}

	// use tabwriter to align the help text
	buf := new(strings.Builder)
	w := tabwriter.NewWriter(buf, 0, 8, 0, '\t', tabwriter.AlignRight)
	fmt.Fprintln(w, builder.String())
	w.Flush()

	head := "To get more information about a label, use `katenary help-label <name_without_prefix>\ne.g. katenary help-label dependencies\n\n"
	return head + buf.String()
}

func generateMarkdownHelp(names []string) string {
	var builder strings.Builder
	var maxNameLength, maxDescriptionLength, maxTypeLength int

	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}
	for _, name := range names {
		help := labelFullHelp[name]
		maxNameLength = max(maxNameLength, len(name)+2+len(KATENARY_PREFIX))
		maxDescriptionLength = max(maxDescriptionLength, len(help.Short))
		maxTypeLength = max(maxTypeLength, len(help.Type))
	}

	fmt.Fprintf(&builder, "%s\n", generateTableHeader(maxNameLength, maxDescriptionLength, maxTypeLength))
	fmt.Fprintf(&builder, "%s\n", generateTableHeaderSeparator(maxNameLength, maxDescriptionLength, maxTypeLength))

	for _, name := range names {
		help := labelFullHelp[name]
		fmt.Fprintf(&builder, "| %-*s | %-*s | %-*s |\n",
			maxNameLength, "`"+KATENARY_PREFIX+name+"`", // enclose in backticks
			maxDescriptionLength, help.Short,
			maxTypeLength, help.Type,
		)
	}

	return builder.String()
}

func generateTableHeader(maxNameLength, maxDescriptionLength, maxTypeLength int) string {
	return fmt.Sprintf(
		"| %-*s | %-*s | %-*s |",
		maxNameLength, "Label name",
		maxDescriptionLength, "Description",
		maxTypeLength, "Type",
	)
}

func generateTableHeaderSeparator(maxNameLength, maxDescriptionLength, maxTypeLength int) string {
	return fmt.Sprintf(
		"| %s | %s | %s |",
		strings.Repeat("-", maxNameLength),
		strings.Repeat("-", maxDescriptionLength),
		strings.Repeat("-", maxTypeLength),
	)
}

// GetLabelHelpFor returns the help for a specific label.
func GetLabelHelpFor(labelname string, asMarkdown bool) string {
	help, ok := labelFullHelp[labelname]
	if !ok {
		return "No help available for " + labelname + "."
	}

	help.Long = strings.TrimPrefix(help.Long, "\n")
	help.Example = strings.TrimPrefix(help.Example, "\n")
	help.Short = strings.TrimPrefix(help.Short, "\n")

	// get help template
	helpTemplate := getHelpTemplate(asMarkdown)

	if asMarkdown {
		// enclose templates in backticks
		help.Long = regexp.MustCompile(`\{\{(.*?)\}\}`).ReplaceAllString(help.Long, "`{{$1}}`")
		help.Long = strings.ReplaceAll(help.Long, "__APP__", "`__APP__`")
	} else {
		help.Long = strings.ReplaceAll(help.Long, " \n", "\n")
		help.Long = strings.ReplaceAll(help.Long, "`", "")
		help.Long = strings.ReplaceAll(help.Long, "<code>", "")
		help.Long = strings.ReplaceAll(help.Long, "</code>", "")
		help.Long = utils.WordWrap(help.Long, 80)
	}

	var buf bytes.Buffer
	template.Must(template.New("shorthelp").Parse(help.Long)).Execute(&buf, struct {
		KATENARY_PREFIX string
	}{
		KATENARY_PREFIX: KATENARY_PREFIX,
	})
	help.Long = buf.String()
	buf.Reset()

	template.Must(template.New("example").Parse(help.Example)).Execute(&buf, struct {
		KATENARY_PREFIX string
	}{
		KATENARY_PREFIX: KATENARY_PREFIX,
	})
	help.Example = buf.String()
	buf.Reset()

	template.Must(template.New("complete").Parse(helpTemplate)).Execute(&buf, struct {
		Name            string
		Help            Help
		KATENARY_PREFIX string
	}{
		Name:            labelname,
		Help:            help,
		KATENARY_PREFIX: KATENARY_PREFIX,
	})

	return buf.String()
}

// GetLabelNames returns a sorted list of all katenary label names.
func GetLabelNames() []string {
	var names []string
	for name := range labelFullHelp {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func getHelpTemplate(asMarkdown bool) string {
	if asMarkdown {
		return `## {{ .KATENARY_PREFIX }}{{ .Name }}

{{ .Help.Short }}

**Type**: ` + "`" + `{{ .Help.Type }}` + "`" + `

{{ .Help.Long }}

**Example:**` + "\n\n```yaml\n" + `{{ .Help.Example }}` + "\n```\n"
	}

	return `{{ .KATENARY_PREFIX }}{{ .Name }}: {{ .Help.Short }}
Type: {{ .Help.Type }}

{{ .Help.Long }}

Example:
{{ .Help.Example }}
`
}
