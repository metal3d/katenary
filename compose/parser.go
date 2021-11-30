package compose

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Parser struct {
	Data *Compose
}

var Appname = ""

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
