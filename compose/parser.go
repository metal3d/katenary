package compose

import (
	"fmt"
	"katenary/helm"
	"log"
	"os"
	"strings"

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
