package compose

import (
	"fmt"
	"katenary/helm"
	"log"
	"os"
	"strings"

	"github.com/google/shlex"
	"gopkg.in/yaml.v3"
)

const (
	ICON_EXCLAMATION = "❕"
)

// Parser is a docker-compose parser.
type Parser struct {
	Data *Compose
}

var Appname = ""

// NewParser create a Parser and parse the file given in filename. If filename is empty, we try to parse the content[0] argument that should be a valid YAML content.
func NewParser(filename string, content ...string) *Parser {

	c := NewCompose()
	if filename != "" {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		dec := yaml.NewDecoder(f)
		err = dec.Decode(c)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		dec := yaml.NewDecoder(strings.NewReader(content[0]))
		err := dec.Decode(c)
		if err != nil {
			log.Fatal(err)
		}
	}

	p := &Parser{Data: c}

	return p
}

func (p *Parser) Parse(appname string) {
	Appname = appname

	services := make(map[string][]string)
	// get the service list, to be sure that everything is ok

	// fix ugly types
	for _, s := range p.Data.Services {
		parseEnv(s)
		parseCommand(s)
	}

	c := p.Data
	for name, s := range c.Services {
		if portlabel, ok := s.Labels[helm.LABEL_PORT]; ok {
			services := strings.Split(portlabel, ",")
			for _, serviceport := range services {
				portexists := false
				for _, found := range s.Ports {
					if found == serviceport {
						portexists = true
					}
				}
				if !portexists {
					s.Ports = append(s.Ports, serviceport)
				}
			}
		}
		if len(s.Ports) > 0 {
			services[name] = s.Ports
		}
	}

	// check if dependencies are resolved
	missing := []string{}
	for name, s := range c.Services {
		for _, dep := range s.DependsOn {
			if _, ok := services[dep]; !ok {
				missing = append(missing, fmt.Sprintf(
					"The service \"%s\" hasn't got "+
						"declared port for dependency from \"%s\" - please "+
						"append a %s label or a \"ports\" section in the docker-compose file",
					dep, name, helm.LABEL_PORT),
				)
			}
		}
	}

	if len(missing) > 0 {
		log.Fatal(strings.Join(missing, "\n"))
	}

	// check if all "image" properties are set
	missing = []string{}
	for name, s := range c.Services {
		if s.Image == "" {
			missing = append(missing, fmt.Sprintf(
				"The service \"%s\" hasn't got "+
					"an image property - please "+
					"append an image property in the docker-compose file",
				name,
			))
		}
	}
	if len(missing) > 0 {
		log.Fatal(strings.Join(missing, "\n"))
	}

	// check the build element
	for name, s := range c.Services {
		if s.RawBuild == nil {
			continue
		}

		fmt.Println(ICON_EXCLAMATION +
			" \x1b[33myou will need to build and push your image named \"" + s.Image + "\"" +
			" for the \"" + name + "\" service \x1b[0m")

	}

}

// manage environment variables, if the type is map[string]string so we can use it, else we need to split "=" sign
// and apply this in env variable
func parseEnv(s *Service) {
	env := make(map[string]string)
	if s.RawEnvironment == nil {
		return
	}
	switch s.RawEnvironment.(type) {
	case map[string]string:
		env = s.RawEnvironment.(map[string]string)
	case map[string]interface{}:
		for k, v := range s.RawEnvironment.(map[string]interface{}) {
			env[k] = v.(string)
		}
	case []interface{}:
		for _, v := range s.RawEnvironment.([]interface{}) {
			// Splot the value of the env variable with "="
			parts := strings.Split(v.(string), "=")
			env[parts[0]] = parts[1]
		}
	case string:
		parts := strings.Split(s.RawEnvironment.(string), "=")
		env[parts[0]] = parts[1]
	default:
		log.Printf("%+v, %T", s.RawEnvironment, s.RawEnvironment)
		log.Printf("%+v", s)
		log.Fatal("Environment type not supported")
	}
	s.Environment = env
}

func parseCommand(s *Service) {

	if s.RawCommand == nil {
		return
	}

	// following the command type, it can be a "slice" or a simple sting, so we need to check it
	switch s.RawCommand.(type) {
	case string:
		// use shlex to parse the command
		command, err := shlex.Split(s.RawCommand.(string))
		if err != nil {
			log.Fatal(err)
		}
		s.Command = command
	case []string:
		s.Command = s.RawCommand.([]string)
	case []interface{}:
		for _, v := range s.RawCommand.([]interface{}) {
			s.Command = append(s.Command, v.(string))
		}
	default:
		log.Printf("%+v %T", s.RawCommand, s.RawCommand)
		log.Fatal("Command type not supported")
	}
}
