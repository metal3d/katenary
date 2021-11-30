package compose

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
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

	return p
}

func (p *Parser) Parse(appname string) {
	Appname = appname
}
