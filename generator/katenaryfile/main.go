package katenaryfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"katenary/generator/labels"
	"katenary/generator/labels/labelStructs"
	"katenary/utils"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/invopop/jsonschema"
	"gopkg.in/yaml.v3"
)

var allowedKatenaryYamlFileNames = []string{"katenary.yaml", "katenary.yml"}

// StringOrMap is a struct that can be either a string or a map of strings.
// It's a helper struct to unmarshal the katenary.yaml file and produce the schema
type StringOrMap any

// Service is a struct that contains the service configuration for katenary
type Service struct {
	MainApp         *bool                          `json:"main-app,omitempty" jsonschema:"title=Is this service the main application"`
	Values          []StringOrMap                  `json:"values,omitempty" jsonschema:"description=Environment variables to be set in values.yaml with or without a description"`
	Secrets         *labelStructs.Secrets          `json:"secrets,omitempty" jsonschema:"title=Secrets,description=Environment variables to be set as secrets"`
	Ports           *labelStructs.Ports            `json:"ports,omitempty" jsonschema:"title=Ports,description=Ports to be exposed in services"`
	Ingress         *labelStructs.Ingress          `json:"ingress,omitempty" jsonschema:"title=Ingress,description=Ingress configuration"`
	HealthCheck     *labelStructs.HealthCheck      `json:"health-check,omitempty" jsonschema:"title=Health Check,description=Health check configuration that respects the kubernetes api"`
	SamePod         *string                        `json:"same-pod,omitempty" jsonschema:"title=Same Pod,description=Service that should be in the same pod"`
	Description     *string                        `json:"description,omitempty" jsonschema:"title=Description,description=Description of the service that will be injected in the values.yaml file"`
	Ignore          *bool                          `json:"ignore,omitempty" jsonschema:"title=Ignore,description=Ignore the service in the conversion"`
	Dependencies    []labelStructs.Dependency      `json:"dependencies,omitempty" jsonschema:"title=Dependencies,description=Services that should be injected in the Chart.yaml file"`
	ConfigMapFile   *labelStructs.ConfigMapFile    `json:"configmap-files,omitempty" jsonschema:"title=ConfigMap Files,description=Files that should be injected as ConfigMap"`
	MapEnv          *labelStructs.MapEnv           `json:"map-env,omitempty" jsonschema:"title=Map Env,description=Map environment variables to another value"`
	CronJob         *labelStructs.CronJob          `json:"cron-job,omitempty" jsonschema:"title=Cron Job,description=Cron Job configuration"`
	EnvFrom         *labelStructs.EnvFrom          `json:"env-from,omitempty" jsonschema:"title=Env From,description=Inject environment variables from another service"`
	ExchangeVolumes []*labelStructs.ExchangeVolume `json:"exchange-volumes,omitempty" jsonschema:"title=Exchange Volumes,description=Exchange volumes between services"`
	ValuesFrom      *labelStructs.ValueFrom        `json:"values-from,omitempty" jsonschema:"title=Values From,description=Inject values from another service (secret or configmap environment variables)"`
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
		if p.Labels == nil {
			p.Labels = make(map[string]string)
			project.Services[i] = p
		}

		if s, ok := services[name]; ok {
			service := project.Services[i]
			getLabelContent(s.MainApp, &service, labels.LabelMainApp)
			getLabelContent(s.Values, &service, labels.LabelValues)
			getLabelContent(s.Secrets, &service, labels.LabelSecrets)
			getLabelContent(s.Ports, &service, labels.LabelPorts)
			getLabelContent(s.Ingress, &service, labels.LabelIngress)
			getLabelContent(s.HealthCheck, &service, labels.LabelHealthCheck)
			getLabelContent(s.SamePod, &service, labels.LabelSamePod)
			getLabelContent(s.Description, &service, labels.LabelDescription)
			getLabelContent(s.Ignore, &service, labels.LabelIgnore)
			getLabelContent(s.Dependencies, &service, labels.LabelDependencies)
			getLabelContent(s.ConfigMapFile, &service, labels.LabelConfigMapFiles)
			getLabelContent(s.MapEnv, &service, labels.LabelMapEnv)
			getLabelContent(s.CronJob, &service, labels.LabelCronJob)
			getLabelContent(s.EnvFrom, &service, labels.LabelEnvFrom)
			getLabelContent(s.ExchangeVolumes, &service, labels.LabelExchangeVolume)
			getLabelContent(s.ValuesFrom, &service, labels.LabelValueFrom)
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
		ing, err := labelStructs.IngressFrom(val)
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

	if service.Labels == nil {
		service.Labels = make(map[string]string)
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

	return string(out.Bytes())
}
