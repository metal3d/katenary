package compose

import (
	"log"
	"os"
	"strings"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

const (
	ICON_EXCLAMATION = "❕"
)

// Parser is a docker-compose parser.
type Parser struct {
	Data *types.Project
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

	p := &Parser{}

	return p
}

func (p *Parser) Parse(appname string) {

	// Reminder:
	// - set Appname
	// - loas services

	options, err := cli.NewProjectOptions(nil,
		cli.WithDefaultConfigPath,
		cli.WithNormalization(true),
		cli.WithInterpolation(true),
		cli.WithResolvedPaths(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	proj, err := cli.ProjectFromOptions(options)
	if err != nil {
		log.Fatal(err)
	}

	Appname = proj.Name

	p.Data = proj

}
