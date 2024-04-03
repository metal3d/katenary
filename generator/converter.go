package generator

import (
	"bytes"
	"errors"
	"fmt"
	"katenary/generator/extrafiles"
	"katenary/parser"
	"katenary/utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/types"
	goyaml "gopkg.in/yaml.v3"
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
			fmt.Print(utils.IconWarning, " The chart directory "+config.OutputDir+" already exists, do you want to overwrite it? [y/N] ")
			var answer string
			fmt.Scanln(&answer)
			if strings.ToLower(answer) != "y" {
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
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}

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
		if err := os.MkdirAll(filepath.Join(templateDir, servicename), 0755); err != nil {
			fmt.Println(utils.IconFailure, err)
			os.Exit(1)
		}
		fmt.Println(icon, "Creating", kind, servicename)
		// if the name is a path, create the directory
		if strings.Contains(name, string(filepath.Separator)) {
			name = filepath.Join(templateDir, name)
			err := os.MkdirAll(filepath.Dir(name), 0755)
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

	// calculate the sha1 hash of the services

	buf := bytes.NewBuffer(nil)
	encoder := goyaml.NewEncoder(buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(chart); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	yamlChart := buf.Bytes()
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

	f, err := os.Create(chartPath)
	if err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}
	f.Write(yamlChart)
	f.Close()

	buf.Reset()
	encoder = goyaml.NewEncoder(buf)
	encoder.SetIndent(2)
	if err = encoder.Encode(&chart.Values); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	values := buf.Bytes()
	values = addDescriptions(values, *project)
	values = addDependencyDescription(values, chart.Dependencies)
	values = addCommentsToValues(values)
	values = addStorageClassHelp(values)
	values = addImagePullSecretsHelp(values)
	values = addImagePullPolicyHelp(values)
	values = addVariablesDoc(values, project)
	values = addMainTagAppDoc(values, project)
	values = append([]byte(headerHelp), values...)

	f, err = os.Create(valuesPath)
	if err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}
	f.Write(values)
	f.Close()

	f, err = os.Create(helpersPath)
	if err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}
	f.Write([]byte(chart.Helper))
	f.Close()

	readme := extrafiles.ReadMeFile(chart.Name, chart.Description, chart.Values)
	f, err = os.Create(readmePath)
	if err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}
	f.Write([]byte(readme))
	f.Close()

	notes := extrafiles.NotesFile()
	f, err = os.Create(notesPath)
	if err != nil {
		fmt.Println(utils.IconFailure, err)
		os.Exit(1)
	}
	f.Write([]byte(notes))
	f.Close()

	if config.HelmUpdate {
		if err := helmUpdate(config); err != nil {
			fmt.Println(utils.IconFailure, err)
			os.Exit(1)
		} else if err := helmLint(config); err != nil {
			fmt.Println(utils.IconFailure, err)
			os.Exit(1)
		} else {
			fmt.Println(utils.IconSuccess, "Helm chart created successfully")
		}
	}
}

const ingressClassHelp = `# Default value for ingress.class annotation
# class: "-"
# If the value is "-", controller will not set ingressClassName
# If the value is "", Ingress will be set to an empty string, so
# controller will use the default value for ingressClass
# If the value is specified, controller will set the named class e.g. "nginx"
# More info: https://kubernetes.io/docs/concepts/services-networking/ingress/#the-ingress-resource
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
# More info: https://kubernetes.io/docs/concepts/storage/storage-classes/
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
	modeline := "# vi" + "m: ft=gotmpl.yaml"

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
		if description, ok := service.Labels[LABEL_DESCRIPTION]; ok {
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

func addDependencyDescription(values []byte, dependencies []Dependency) []byte {
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
# More info: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
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
# More info: https://kubernetes.io/docs/concepts/containers/images/#updating-images
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

func addVariablesDoc(values []byte, project *types.Project) []byte {

	lines := strings.Split(string(values), "\n")

	currentService := ""
	for _, service := range project.Services {
		variables := utils.GetValuesFromLabel(service, LABEL_VALUES)
		for i, line := range lines {
			if regexp.MustCompile(`(?m)^` + service.Name + `:`).MatchString(line) {
				currentService = service.Name
			}
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
	}
	return []byte(strings.Join(lines, "\n"))
}

const mainTagAppDoc = `This is the version of the main application.
Leave it to blank to use the Chart "AppVersion" value.`

func addMainTagAppDoc(values []byte, project *types.Project) []byte {
	lines := strings.Split(string(values), "\n")

	for _, service := range project.Services {
		inService := false
		inRegistry := false
		// read the label LabelMainApp
		if v, ok := service.Labels[LABEL_MAIN_APP]; !ok {
			continue
		} else if v == "false" || v == "no" || v == "0" {
			continue
		} else {
			fmt.Printf("%s Adding main tag app doc %s\n", utils.IconConfig, service.Name)
		}

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
	}

	return []byte(strings.Join(lines, "\n"))
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
			if strings.Contains(label, "katenary.") && !strings.Contains(label, KATENARY_PREFIX) {
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
			KATENARY_PREFIX[0:len(KATENARY_PREFIX)-1],
			strings.Join(badServices, "\n"),
		)

		return errors.New(utils.WordWrap(message, 80))

	}
	return nil
}

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
