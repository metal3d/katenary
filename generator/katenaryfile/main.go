package katenaryfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"katenary/generator/labels"
	"katenary/generator/labels/labelstructs"
	"katenary/utils"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/compose-spec/compose-go/types"
	"github.com/invopop/jsonschema"
	"gopkg.in/yaml.v3"
)

var allowedKatenaryYamlFileNames = []string{"katenary.yaml", "katenary.yml"}

// StringOrMap is a struct that can be either a string or a map of strings.
// It's a helper struct to unmarshal the katenary.yaml file and produce the schema
type StringOrMap any

// Service is a struct that contains the service configuration for katenary
type Service struct {
	MainApp         *bool                          `yaml:"main-app,omitempty" jsonschema:"title=Is this service the main application"`
	Values          []StringOrMap                  `yaml:"values,omitempty" jsonschema:"description=Environment variables to be set in values.yaml with or without a description"`
	Secrets         *labelstructs.Secrets          `yaml:"secrets,omitempty" jsonschema:"title=Secrets,description=Environment variables to be set as secrets"`
	Ports           *labelstructs.Ports            `yaml:"ports,omitempty" jsonschema:"title=Ports,description=Ports to be exposed in services"`
	Ingress         *labelstructs.Ingress          `yaml:"ingress,omitempty" jsonschema:"title=Ingress,description=Ingress configuration"`
	HealthCheck     *labelstructs.HealthCheck      `yaml:"health-check,omitempty" jsonschema:"title=Health Check,description=Health check configuration that respects the kubernetes api"`
	SamePod         *string                        `yaml:"same-pod,omitempty" jsonschema:"title=Same Pod,description=Service that should be in the same pod"`
	Description     *string                        `yaml:"description,omitempty" jsonschema:"title=Description,description=Description of the service that will be injected in the values.yaml file"`
	Ignore          *bool                          `yaml:"ignore,omitempty" jsonschema:"title=Ignore,description=Ignore the service in the conversion"`
	Dependencies    []labelstructs.Dependency      `yaml:"dependencies,omitempty" jsonschema:"title=Dependencies,description=Services that should be injected in the Chart.yaml file"`
	ConfigMapFiles  *labelstructs.ConfigMapFiles   `yaml:"configmap-files,omitempty" jsonschema:"title=ConfigMap Files,description=Files that should be injected as ConfigMap"`
	MapEnv          *labelstructs.MapEnv           `yaml:"map-env,omitempty" jsonschema:"title=Map Env,description=Map environment variables to another value"`
	CronJob         *labelstructs.CronJob          `yaml:"cron-job,omitempty" jsonschema:"title=Cron Job,description=Cron Job configuration"`
	EnvFrom         *labelstructs.EnvFrom          `yaml:"env-from,omitempty" jsonschema:"title=Env From,description=Inject environment variables from another service"`
	ExchangeVolumes []*labelstructs.ExchangeVolume `yaml:"exchange-volumes,omitempty" jsonschema:"title=Exchange Volumes,description=Exchange volumes between services"`
	ValuesFrom      *labelstructs.ValueFrom        `yaml:"values-from,omitempty" jsonschema:"title=Values From,description=Inject values from another service (secret or configmap environment variables)"`
}

// OverrideWithConfig overrides the project with the katenary.yaml file. It
// will set the labels of the services with the values from the katenary.yaml file.
// It work in memory, so it will not modify the original project.
func OverrideWithConfig(project *types.Project) {
	var yamlFile string
	var err error
	for _, yamlFile = range allowedKatenaryYamlFileNames {
		_, err = os.Stat(yamlFile)
		if err == nil {
			break
		}
	}
	if err != nil {
		// no katenary file found
		return
	}
	fmt.Println(utils.IconInfo, "Using katenary file", yamlFile)

	services := make(map[string]Service)
	fp, err := os.Open(yamlFile)
	if err != nil {
		return
	}
	if err := yaml.NewDecoder(fp).Decode(&services); err != nil {
		log.Fatal(err)
		return
	}
	for i, p := range project.Services {
		name := p.Name
		if project.Services[i].Labels == nil {
			project.Services[i].Labels = make(map[string]string)
		}
		mustGetLabelContent := func(o any, s *types.ServiceConfig, labelName string) {
			err := getLabelContent(o, s, labelName)
			if err != nil {
				log.Fatal(err)
			}
		}

		if s, ok := services[name]; ok {
			mustGetLabelContent(s.MainApp, &project.Services[i], labels.LabelMainApp)
			mustGetLabelContent(s.Values, &project.Services[i], labels.LabelValues)
			mustGetLabelContent(s.Secrets, &project.Services[i], labels.LabelSecrets)
			mustGetLabelContent(s.Ports, &project.Services[i], labels.LabelPorts)
			mustGetLabelContent(s.Ingress, &project.Services[i], labels.LabelIngress)
			mustGetLabelContent(s.HealthCheck, &project.Services[i], labels.LabelHealthCheck)
			mustGetLabelContent(s.SamePod, &project.Services[i], labels.LabelSamePod)
			mustGetLabelContent(s.Description, &project.Services[i], labels.LabelDescription)
			mustGetLabelContent(s.Ignore, &project.Services[i], labels.LabelIgnore)
			mustGetLabelContent(s.Dependencies, &project.Services[i], labels.LabelDependencies)
			mustGetLabelContent(s.ConfigMapFiles, &project.Services[i], labels.LabelConfigMapFiles)
			mustGetLabelContent(s.MapEnv, &project.Services[i], labels.LabelMapEnv)
			mustGetLabelContent(s.CronJob, &project.Services[i], labels.LabelCronJob)
			mustGetLabelContent(s.EnvFrom, &project.Services[i], labels.LabelEnvFrom)
			mustGetLabelContent(s.ExchangeVolumes, &project.Services[i], labels.LabelExchangeVolume)
			mustGetLabelContent(s.ValuesFrom, &project.Services[i], labels.LabelValuesFrom)
		}
	}
	fmt.Println(utils.IconInfo, "Katenary file loaded successfully, the services are now configured.")
}

func getLabelContent(o any, service *types.ServiceConfig, labelName string) error {
	if reflect.ValueOf(o).IsZero() {
		return nil
	}

	c, err := yaml.Marshal(o)
	if err != nil {
		log.Println(err)
		return err
	}
	val := strings.TrimSpace(string(c))
	if labelName == labels.LabelIngress {
		// special case, values must be set from some defaults
		ing, err := labelstructs.IngressFrom(val)
		if err != nil {
			log.Fatal(err)
			return err
		}
		c, err := yaml.Marshal(ing)
		if err != nil {
			return err
		}
		val = strings.TrimSpace(string(c))
	}

	service.Labels[labelName] = val
	return nil
}

// GenerateSchema generates the schema for the katenary.yaml file.
func GenerateSchema() string {
	s := jsonschema.Reflect(map[string]Service{})

	// redefine the IntOrString type from k8s
	s.Definitions["IntOrString"] = &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{Type: "integer"},
			{Type: "string"},
		},
	}

	// same for the StringOrMap type, that can be either a string or a map of string:string
	s.Definitions["StringOrMap"] = &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{Type: "string"},
			{Type: "object", AdditionalProperties: &jsonschema.Schema{Type: "string"}},
		},
	}

	c, _ := s.MarshalJSON()
	// indent the json
	var out bytes.Buffer
	err := json.Indent(&out, c, "", "  ")
	if err != nil {
		return err.Error()
	}

	return out.String()
}
