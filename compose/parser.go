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

// NewParser create a Parser and parse the file given in filename.
func NewParser(filename string) *Parser {

	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	c := NewCompose()
	dec := yaml.NewDecoder(f)
	dec.Decode(c)

	p := &Parser{Data: c}

	services := make(map[string][]string)
	// get the service list, to be sure that everything is ok

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

	// check the build element
	for name, s := range c.Services {
		if s.RawBuild == nil {
			continue
		}

		fmt.Println(ICON_EXCLAMATION +
			" \x1b[33myou will need to build and push your image named \"" + s.Image + "\"" +
			" for the \"" + name + "\" service \x1b[0m")

	}

	return p
}

func (p *Parser) Parse(appname string) {
	Appname = appname
}
