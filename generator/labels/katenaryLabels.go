package labels

import (
	"bytes"
	_ "embed"
	"fmt"
	"katenary/utils"
	"log"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"

	"sigs.k8s.io/yaml"
)

const KatenaryLabelPrefix = "katenary.v3"

// Known labels.
const (
	LabelMainApp        Label = KatenaryLabelPrefix + "/main-app"
	LabelValues         Label = KatenaryLabelPrefix + "/values"
	LabelSecrets        Label = KatenaryLabelPrefix + "/secrets"
	LabelPorts          Label = KatenaryLabelPrefix + "/ports"
	LabelIngress        Label = KatenaryLabelPrefix + "/ingress"
	LabelMapEnv         Label = KatenaryLabelPrefix + "/map-env"
	LabelHealthCheck    Label = KatenaryLabelPrefix + "/health-check"
	LabelSamePod        Label = KatenaryLabelPrefix + "/same-pod"
	LabelDescription    Label = KatenaryLabelPrefix + "/description"
	LabelIgnore         Label = KatenaryLabelPrefix + "/ignore"
	LabelDependencies   Label = KatenaryLabelPrefix + "/dependencies"
	LabelConfigMapFiles Label = KatenaryLabelPrefix + "/configmap-files"
	LabelCronJob        Label = KatenaryLabelPrefix + "/cronjob"
	LabelEnvFrom        Label = KatenaryLabelPrefix + "/env-from"
	LabelExchangeVolume Label = KatenaryLabelPrefix + "/exchange-volumes"
	LabelValueFrom      Label = KatenaryLabelPrefix + "/values-from"
)

var (
	// Set the documentation of labels here
	//
	//go:embed katenaryLabelsDoc.yaml
	labelFullHelpYAML []byte

	// parsed yaml
	labelFullHelp map[string]Help

	//go:embed help-template.tpl
	helpTemplatePlain string

	//go:embed help-template.md.tpl
	helpTemplateMarkdown string
)

// Label is a katenary label to find in compose files.
type Label = string

func LabelName(name string) Label {
	return Label(KatenaryLabelPrefix + "/" + name)
}

// Help is the documentation of a label.
type Help struct {
	Short   string `yaml:"short"`
	Long    string `yaml:"long"`
	Example string `yaml:"example"`
	Type    string `yaml:"type"`
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

// GetLabelHelpFor returns the help for a specific label.
func GetLabelHelpFor(labelname string, asMarkdown bool) string {
	help, ok := labelFullHelp[labelname]
	if !ok {
		return "No help available for " + labelname + "."
	}

	help.Long = strings.TrimPrefix(help.Long, "\n")
	help.Example = strings.TrimPrefix(help.Example, "\n")
	help.Short = strings.TrimPrefix(help.Short, "\n")

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

	// get help template
	var helpTemplate string
	switch asMarkdown {
	case true:
		helpTemplate = helpTemplateMarkdown
	case false:
		helpTemplate = helpTemplatePlain
	}

	var buf bytes.Buffer
	var err error
	err = template.Must(template.New("shorthelp").Parse(help.Long)).Execute(&buf, struct {
		KatenaryPrefix string
	}{
		KatenaryPrefix: KatenaryLabelPrefix,
	})
	if err != nil {
		log.Fatalf("Error executing template: %v", err)
	}
	help.Long = buf.String()
	buf.Reset()

	err = template.Must(template.New("example").Parse(help.Example)).Execute(&buf, struct {
		KatenaryPrefix string
	}{
		KatenaryPrefix: KatenaryLabelPrefix,
	})
	if err != nil {
		log.Fatalf("Error executing template: %v", err)
	}
	help.Example = buf.String()
	buf.Reset()

	err = template.Must(template.New("complete").Parse(helpTemplate)).Execute(&buf, struct {
		Name           string
		Help           Help
		KatenaryPrefix string
	}{
		Name:           labelname,
		Help:           help,
		KatenaryPrefix: KatenaryLabelPrefix,
	})
	if err != nil {
		log.Fatalf("Error executing template: %v", err)
	}

	return buf.String()
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
		maxNameLength = max(maxNameLength, len(name)+3+len(KatenaryLabelPrefix))
		maxDescriptionLength = max(maxDescriptionLength, len(help.Short))
		maxTypeLength = max(maxTypeLength, len(help.Type)+3)
	}

	fmt.Fprintf(&builder, "%s\n", generateTableHeader(maxNameLength, maxDescriptionLength, maxTypeLength))
	fmt.Fprintf(&builder, "%s\n", generateTableHeaderSeparator(maxNameLength, maxDescriptionLength, maxTypeLength))

	for _, name := range names {
		help := labelFullHelp[name]
		fmt.Fprintf(&builder, "| %-*s | %-*s | %-*s |\n",
			maxNameLength, "`"+LabelName(name)+"`", // enclose in backticks
			maxDescriptionLength, help.Short,
			maxTypeLength, "`"+help.Type+"`",
		)
	}

	return builder.String()
}

func generatePlainHelp(names []string) string {
	var builder strings.Builder
	for _, name := range names {
		help := labelFullHelp[name]
		fmt.Fprintf(&builder, "%s:\t%s\t%s\n", LabelName(name), help.Type, help.Short)
	}

	// use tabwriter to align the help text
	buf := new(strings.Builder)
	w := tabwriter.NewWriter(buf, 0, 8, 0, '\t', tabwriter.AlignRight)
	fmt.Fprintln(w, builder.String())
	w.Flush()

	head := "To get more information about a label, use `katenary help-label <name_without_prefix>\ne.g. katenary help-label dependencies\n\n"
	return head + buf.String()
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

func Prefix() string {
	return KatenaryLabelPrefix
}
