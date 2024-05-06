package generator

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/types"

	"katenary/generator/extrafiles"
	"katenary/generator/labelStructs"
	"katenary/parser"
	"katenary/utils"
)

const headerHelp = `# This file is autogenerated by katenary
#
# DO NOT EDIT IT BY HAND UNLESS YOU KNOW WHAT YOU ARE DOING
#
# If you want to change the content of this file, you should edit the
# compose file and run katenary again.
# If you need to override some values, you can do it in a override file
# and use the -f flag to specify it when running the helm command.


`

// Convert a compose (docker, podman...) project to a helm chart.
// It calls Generate() to generate the chart and then write it to the disk.
func Convert(config ConvertOptions, dockerComposeFile ...string) {
	var (
		templateDir = filepath.Join(config.OutputDir, "templates")
		helpersPath = filepath.Join(config.OutputDir, "templates", "_helpers.tpl")
		chartPath   = filepath.Join(config.OutputDir, "Chart.yaml")
		valuesPath  = filepath.Join(config.OutputDir, "values.yaml")
		readmePath  = filepath.Join(config.OutputDir, "README.md")
		notesPath   = filepath.Join(templateDir, "NOTES.txt")
	)

	// the current working directory is the directory
	currentDir, _ := os.Getwd()
	// go to the root of the project
	if err := os.Chdir(filepath.Dir(dockerComposeFile[0])); err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}
	defer os.Chdir(currentDir) // after the generation, go back to the original directory

	// repove the directory part of the docker-compose files
	for i, f := range dockerComposeFile {
		dockerComposeFile[i] = filepath.Base(f)
	}

	// parse the compose files
	project, err := parser.Parse(config.Profiles, dockerComposeFile...)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// check older version of labels
	if err := checkOldLabels(project); err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}

	if !config.Force {
		// check if the chart directory exists
		// if yes, prevent the user from overwriting it and ask for confirmation
		if _, err := os.Stat(config.OutputDir); err == nil {
			overwrite := utils.Confirm(
				"The chart directory "+config.OutputDir+" already exists, do you want to overwrite it?",
				utils.IconWarning,
			)
			if !overwrite {
				fmt.Println("Aborting")
				os.Exit(126) // 126 is the exit code for "Command invoked cannot execute"
			}
		}
		fmt.Println() // clean line
	}

	// Build the objects !
	chart, err := Generate(project)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// if the app version is set from the command line, use it
	if config.AppVersion != nil {
		chart.AppVersion = *config.AppVersion
	}
	chart.Version = config.ChartVersion

	// remove the chart directory if it exists
	os.RemoveAll(config.OutputDir)

	// create the chart directory
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}

	// write the templates to the disk
	chart.SaveTemplates(templateDir)

	// write the Chart.yaml file
	buildCharYamlFile(chart, project, chartPath)

	// build and write the values.yaml file
	buildValues(chart, project, valuesPath)

	// write the _helpers.tpl to the disk
	writeContent(helpersPath, []byte(chart.Helper))

	// write the readme to the disk
	readme := extrafiles.ReadMeFile(chart.Name, chart.Description, chart.Values)
	writeContent(readmePath, []byte(readme))

	// get the list of services to write in the notes
	buildNotesFile(project, notesPath)

	// call helm update if needed
	callHelmUpdate(config)
}

const ingressClassHelp = `# Default value for ingress.class annotation
# class: "-"
# If the value is "-", controller will not set ingressClassName
# If the value is "", Ingress will be set to an empty string, so
# controller will use the default value for ingressClass
# If the value is specified, controller will set the named class e.g. "nginx"
`

func addCommentsToValues(values []byte) []byte {
	lines := strings.Split(string(values), "\n")
	for i, line := range lines {
		if strings.Contains(line, "ingress:") {
			spaces := utils.CountStartingSpaces(line)
			spacesString := strings.Repeat(" ", spaces)
			// indent ingressClassHelper comment
			ingressClassHelp := strings.ReplaceAll(ingressClassHelp, "\n", "\n"+spacesString)
			ingressClassHelp = strings.TrimRight(ingressClassHelp, " ")
			ingressClassHelp = spacesString + ingressClassHelp
			lines[i] = ingressClassHelp + line
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

const storageClassHelp = `# Storage class to use for PVCs
# storageClass: "-" means use default
# storageClass: "" means do not specify
# storageClass: "foo" means use that storageClass
`

// addStorageClassHelp adds a comment to the values.yaml file to explain how to
// use the storageClass option.
func addStorageClassHelp(values []byte) []byte {
	lines := strings.Split(string(values), "\n")
	for i, line := range lines {
		if strings.Contains(line, "storageClass:") {
			spaces := utils.CountStartingSpaces(line)
			spacesString := strings.Repeat(" ", spaces)
			// indent ingressClassHelper comment
			storageClassHelp := strings.ReplaceAll(storageClassHelp, "\n", "\n"+spacesString)
			storageClassHelp = strings.TrimRight(storageClassHelp, " ")
			storageClassHelp = spacesString + storageClassHelp
			lines[i] = storageClassHelp + line
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// addModeline adds a modeline to the values.yaml file to make sure that vim
// will use the correct syntax highlighting.
func addModeline(values []byte) []byte {
	modeline := "# vi" + "m: ft=helm.gotmpl.yaml"

	// if the values ends by `{{- end }}` we need to add the modeline before
	lines := strings.Split(string(values), "\n")

	if lines[len(lines)-1] == "{{- end }}" || lines[len(lines)-1] == "{{- end -}}" {
		lines = lines[:len(lines)-1]
		lines = append(lines, modeline, "{{- end }}")
		return []byte(strings.Join(lines, "\n"))
	}

	return append(values, []byte(modeline)...)
}

// addDescriptions adds the description from the label to the values.yaml file on top
// of the service definition.
func addDescriptions(values []byte, project types.Project) []byte {
	for _, service := range project.Services {
		if description, ok := service.Labels[LabelDescription]; ok {
			// set it as comment
			description = "\n# " + strings.ReplaceAll(description, "\n", "\n# ")

			values = regexp.MustCompile(
				`(?m)^`+service.Name+`:$`,
			).ReplaceAll(values, []byte(description+"\n"+service.Name+":"))
		} else {
			// set it as comment
			description = "\n# " + service.Name + " configuration"

			values = regexp.MustCompile(
				`(?m)^`+service.Name+`:$`,
			).ReplaceAll(
				values,
				[]byte(description+"\n"+service.Name+":"),
			)
		}
	}
	return values
}

func addDependencyDescription(values []byte, dependencies []labelStructs.Dependency) []byte {
	for _, d := range dependencies {
		name := d.Name
		if d.Alias != "" {
			name = d.Alias
		}

		values = regexp.MustCompile(
			`(?m)^`+name+`:$`,
		).ReplaceAll(
			values,
			[]byte("\n# "+d.Name+" helm dependency configuration\n"+name+":"),
		)
	}
	return values
}

const imagePullSecretHelp = `
# imagePullSecrets allows you to specify a name of an image pull secret.
# You must provide a list of object with the name field set to the name of the
# e.g.
# pullSecrets:
# - name: regcred
# You are, for now, repsonsible for creating the secret.
`

func addImagePullSecretsHelp(values []byte) []byte {
	// add imagePullSecrets help
	lines := strings.Split(string(values), "\n")

	for i, line := range lines {
		if strings.Contains(line, "pullSecrets:") {
			spaces := utils.CountStartingSpaces(line)
			spacesString := strings.Repeat(" ", spaces)
			// indent imagePullSecretHelp comment
			imagePullSecretHelp := strings.ReplaceAll(imagePullSecretHelp, "\n", "\n"+spacesString)
			imagePullSecretHelp = strings.TrimRight(imagePullSecretHelp, " ")
			imagePullSecretHelp = spacesString + imagePullSecretHelp
			lines[i] = imagePullSecretHelp + line
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func addChartDoc(values []byte, project *types.Project) []byte {
	chartDoc := fmt.Sprintf(`# This is the main values.yaml file for the %s chart.
# More information can be found in the chart's README.md file.
#
`, project.Name)

	lines := strings.Split(string(values), "\n")
	for i, line := range lines {
		if regexp.MustCompile(`(?m)^name:`).MatchString(line) {
			doc := "\n# Name of the chart (required), basically the name of the project.\n"
			lines[i] = doc + line
		} else if regexp.MustCompile(`(?m)^version:`).MatchString(line) {
			doc := "\n# Version of the chart (required)\n"
			lines[i] = doc + line
		} else if strings.Contains(line, "appVersion:") {
			spaces := utils.CountStartingSpaces(line)
			doc := fmt.Sprintf(
				"\n%s# Version of the application (required).\n%s# This should be the main application version.\n",
				strings.Repeat(" ", spaces),
				strings.Repeat(" ", spaces),
			)
			lines[i] = doc + line
		} else if strings.Contains(line, "dependencies:") {
			spaces := utils.CountStartingSpaces(line)
			doc := fmt.Sprintf("\n"+
				"%s# Dependencies are external charts that this chart will depend on.\n"+
				"%s# More information can be found in the chart's README.md file.\n",
				strings.Repeat(" ", spaces),
				strings.Repeat(" ", spaces),
			)
			lines[i] = doc + line
		}
	}
	return []byte(chartDoc + strings.Join(lines, "\n"))
}

const imagePullPolicyHelp = `# imagePullPolicy allows you to specify a policy to cache or always pull an image.
# You must provide a string value with one of the following values:
# - Always       -> will always pull the image
# - Never        -> will never pull the image, the image should be present on the node
# - IfNotPresent -> will pull the image only if it is not present on the node
`

func addImagePullPolicyHelp(values []byte) []byte {
	// add imagePullPolicy help
	lines := strings.Split(string(values), "\n")
	for i, line := range lines {
		if strings.Contains(line, "imagePullPolicy:") {
			spaces := utils.CountStartingSpaces(line)
			spacesString := strings.Repeat(" ", spaces)
			// indent imagePullPolicyHelp comment
			imagePullPolicyHelp := strings.ReplaceAll(imagePullPolicyHelp, "\n", "\n"+spacesString)
			imagePullPolicyHelp = strings.TrimRight(imagePullPolicyHelp, " ")
			imagePullPolicyHelp = spacesString + imagePullPolicyHelp
			lines[i] = imagePullPolicyHelp + line
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

const resourceHelp = `# Resources allows you to specify the resource requests and limits for a service.
# Resources are used to specify the amount of CPU and memory that 
# a container needs.
#
# e.g.
# resources:
#   requests:
#     memory: "64Mi"
#     cpu: "250m"
#   limits:
#     memory: "128Mi"
#     cpu: "500m"
`

func addResourceHelp(values []byte) []byte {
	lines := strings.Split(string(values), "\n")
	for i, line := range lines {
		if strings.Contains(line, "resources:") {
			spaces := utils.CountStartingSpaces(line)
			spacesString := strings.Repeat(" ", spaces)
			// indent resourceHelp comment
			resourceHelp := strings.ReplaceAll(resourceHelp, "\n", "\n"+spacesString)
			resourceHelp = strings.TrimRight(resourceHelp, " ")
			resourceHelp = spacesString + resourceHelp
			lines[i] = resourceHelp + line
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func addVariablesDoc(values []byte, project *types.Project) []byte {
	lines := strings.Split(string(values), "\n")

	for _, service := range project.Services {
		lines = addDocToVariable(service, lines)
	}
	return []byte(strings.Join(lines, "\n"))
}

func addDocToVariable(service types.ServiceConfig, lines []string) []string {
	currentService := ""
	variables := utils.GetValuesFromLabel(service, LabelValues)
	for i, line := range lines {
		// if the line is a service, it is a name followed by a colon
		if regexp.MustCompile(`(?m)^` + service.Name + `:`).MatchString(line) {
			currentService = service.Name
		}
		// for each variable in the service, add the description
		for varname, variable := range variables {
			if variable == nil {
				continue
			}
			spaces := utils.CountStartingSpaces(line)
			if regexp.MustCompile(`(?m)\s*`+varname+`:`).MatchString(line) && currentService == service.Name {

				// add # to the beginning of the Description
				doc := strings.ReplaceAll("\n"+variable.Description, "\n", "\n"+strings.Repeat(" ", spaces)+"# ")
				doc = strings.TrimRight(doc, " ")
				doc += "\n" + line

				lines[i] = doc
			}
		}
	}
	return lines
}

const mainTagAppDoc = `This is the version of the main application.
Leave it to blank to use the Chart "AppVersion" value.`

func addMainTagAppDoc(values []byte, project *types.Project) []byte {
	lines := strings.Split(string(values), "\n")

	for _, service := range project.Services {
		// read the label LabelMainApp
		if v, ok := service.Labels[LabelMainApp]; !ok {
			continue
		} else if v == "false" || v == "no" || v == "0" {
			continue
		} else {
			fmt.Printf("%s Adding main tag app doc %s\n", utils.IconConfig, service.Name)
		}

		lines = addMainAppDoc(lines, service)
	}

	return []byte(strings.Join(lines, "\n"))
}

func addMainAppDoc(lines []string, service types.ServiceConfig) []string {
	inService := false
	inRegistry := false
	for i, line := range lines {
		if regexp.MustCompile(`^` + service.Name + `:`).MatchString(line) {
			inService = true
		}
		if inService && regexp.MustCompile(`^\s*repository:.*`).MatchString(line) {
			inRegistry = true
		}
		if inService && inRegistry {
			if regexp.MustCompile(`^\s*tag: .*`).MatchString(line) {
				spaces := utils.CountStartingSpaces(line)
				doc := strings.ReplaceAll(mainTagAppDoc, "\n", "\n"+strings.Repeat(" ", spaces)+"# ")
				doc = strings.Repeat(" ", spaces) + "# " + doc

				lines[i] = doc + "\n" + line + "\n"
				break
			}
		}
	}
	return lines
}

func removeNewlinesInsideBrackets(values []byte) []byte {
	re, err := regexp.Compile(`(?s)\{\{(.*?)\}\}`)
	if err != nil {
		log.Fatal(err)
	}
	return re.ReplaceAllFunc(values, func(b []byte) []byte {
		// get the first match
		matches := re.FindSubmatch(b)
		replacement := bytes.ReplaceAll(matches[1], []byte("\n"), []byte(" "))
		// remove repeated spaces
		replacement = regexp.MustCompile(`\s+`).ReplaceAll(replacement, []byte(" "))
		// remove newlines inside brackets
		return bytes.ReplaceAll(b, matches[1], replacement)
	})
}

var unwantedLines = []string{
	"creationTimestamp:",
	"status:",
}

func removeUnwantedLines(values []byte) []byte {
	lines := strings.Split(string(values), "\n")
	output := []string{}
	for _, line := range lines {
		next := false
		for _, unwanted := range unwantedLines {
			if strings.Contains(line, unwanted) {
				next = true
			}
		}
		if !next {
			output = append(output, line)
		}
	}
	return []byte(strings.Join(output, "\n"))
}

// check if the project makes use of older labels (kanetary.[^v3])
func checkOldLabels(project *types.Project) error {
	badServices := make([]string, 0)
	for _, service := range project.Services {
		for label := range service.Labels {
			if strings.Contains(label, "katenary.") && !strings.Contains(label, katenaryLabelPrefix) {
				badServices = append(badServices, fmt.Sprintf("- %s: %s", service.Name, label))
			}
		}
	}
	if len(badServices) > 0 {
		message := fmt.Sprintf(` Old labels detected in project "%s".

  The current version of katenary uses labels with the prefix "%s" which are not compatible with previous versions.
  Your project is not compatible with this version.
  
  Please upgrade your labels to follow the current version
  
  Services to upgrade:
%s`,
			project.Name,
			katenaryLabelPrefix[0:len(katenaryLabelPrefix)-1],
			strings.Join(badServices, "\n"),
		)

		return errors.New(utils.WordWrap(message, 80))

	}
	return nil
}

// helmUpdate runs "helm dependency update" on the output directory.
func helmUpdate(config ConvertOptions) error {
	// lookup for "helm" binary
	fmt.Println(utils.IconInfo, "Updating helm dependencies...")
	helm, err := exec.LookPath("helm")
	if err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}
	// run "helm dependency update"
	cmd := exec.Command(helm, "dependency", "update", config.OutputDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// helmLint runs "helm lint" on the output directory.
func helmLint(config ConvertOptions) error {
	fmt.Println(utils.IconInfo, "Linting...")
	helm, err := exec.LookPath("helm")
	if err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}
	cmd := exec.Command(helm, "lint", config.OutputDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// keyRegExp checks if the line starts by a #
var keyRegExp = regexp.MustCompile(`^\s*[^#]+:.*`)

// addYAMLSelectorPath adds a selector path to the yaml file for each key
// as comment. E.g. foo.ingress.host
func addYAMLSelectorPath(values []byte) []byte {
	lines := strings.Split(string(values), "\n")
	currentKey := ""
	currentLevel := 0
	toReturn := []string{}
	for _, line := range lines {
		// if the line is a not a key, continue
		if !keyRegExp.MatchString(line) {
			toReturn = append(toReturn, line)
			continue
		}
		// get the key
		key := strings.TrimSpace(strings.Split(line, ":")[0])

		// get the spaces
		spaces := utils.CountStartingSpaces(line)

		if spaces/2 > currentLevel {
			currentLevel++
		} else if spaces/2 < currentLevel {
			currentLevel--
		}
		currentKey = strings.Join(strings.Split(currentKey, ".")[:spaces/2], ".")

		if currentLevel == 0 {
			currentKey = key
			toReturn = append(toReturn, line)
			continue
		}
		// if the key is not empty, add the selector path
		if currentKey != "" {
			currentKey += "."
		}
		currentKey += key
		// add the selector path as comment
		toReturn = append(
			toReturn,
			strings.Repeat(" ", spaces)+"# key: "+currentKey+"\n"+line,
		)
	}
	return []byte(strings.Join(toReturn, "\n"))
}

func writeContent(path string, content []byte) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}
	defer f.Close()
	f.Write(content)
}

func buildValues(chart *HelmChart, project *types.Project, valuesPath string) {
	values, err := utils.EncodeBasicYaml(&chart.Values)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	values = addDescriptions(values, *project)
	values = addDependencyDescription(values, chart.Dependencies)
	values = addCommentsToValues(values)
	values = addStorageClassHelp(values)
	values = addImagePullSecretsHelp(values)
	values = addImagePullPolicyHelp(values)
	values = addVariablesDoc(values, project)
	values = addMainTagAppDoc(values, project)
	values = addResourceHelp(values)
	values = addYAMLSelectorPath(values)
	values = append([]byte(headerHelp), values...)

	// add vim modeline
	values = append(values, []byte("\n# vim: ft=yaml\n")...)

	// write the values to the disk
	writeContent(valuesPath, values)
}

func buildNotesFile(project *types.Project, notesPath string) {
	// get the list of services to write in the notes
	services := make([]string, 0)
	for _, service := range project.Services {
		services = append(services, service.Name)
	}
	// write the notes to the disk
	notes := extrafiles.NotesFile(services)
	writeContent(notesPath, []byte(notes))
}

func buildCharYamlFile(chart *HelmChart, project *types.Project, chartPath string) {
	// calculate the sha1 hash of the services
	yamlChart, err := utils.EncodeBasicYaml(chart)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// concat chart adding a comment with hash of services on top
	yamlChart = append([]byte(fmt.Sprintf("# compose hash (sha1): %s\n", *chart.composeHash)), yamlChart...)
	// add the list of compose files
	files := []string{}
	for _, file := range project.ComposeFiles {
		base := filepath.Base(file)
		files = append(files, base)
	}
	yamlChart = append([]byte(fmt.Sprintf("# compose files: %s\n", strings.Join(files, ", "))), yamlChart...)
	// add generated date
	yamlChart = append([]byte(fmt.Sprintf("# generated at: %s\n", time.Now().Format(time.RFC3339))), yamlChart...)

	// document Chart.yaml file
	yamlChart = addChartDoc(yamlChart, project)

	writeContent(chartPath, yamlChart)
}

func callHelmUpdate(config ConvertOptions) {
	executeAndHandleError := func(fn func(ConvertOptions) error, config ConvertOptions, message string) {
		if err := fn(config); err != nil {
			fmt.Println(utils.IconFailure, err)
			os.Exit(1)
		}
		fmt.Println(utils.IconSuccess, message)
	}
	if config.HelmUpdate {
		executeAndHandleError(helmUpdate, config, "Helm dependencies updated")
		executeAndHandleError(helmLint, config, "Helm chart linted")
		fmt.Println(utils.IconSuccess, "Helm chart created successfully")
	}
}
