package compose

import (
	"log"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
)

const (
	ICON_EXCLAMATION = "❕"
)

// Parser is a docker-compose parser.
type Parser struct {
	Data      *types.Project
	temporary *string
}

var Appname = ""

// NewParser create a Parser and parse the file given in filename. If filename is empty, we try to parse the content[0] argument that should be a valid YAML content.
func NewParser(filename string, content ...string) *Parser {

	p := &Parser{}

	if len(content) > 0 {
		//write it in a temporary file
		tmp, err := os.MkdirTemp(os.TempDir(), "katenary-")
		if err != nil {
			log.Fatal(err)
		}
		tmpfile, err := os.Create(filepath.Join(tmp, "tmp.yml"))
		if err != nil {
			log.Fatal(err)
		}
		tmpfile.WriteString(content[0])
		tmpfile.Close()
		filename = tmpfile.Name()
		p.temporary = &tmp
		cli.DefaultFileNames = []string{filename}
	}
	// if filename is not in cli Default files, add it
	if len(filename) > 0 {
		found := false
		for _, f := range cli.DefaultFileNames {
			if f == filename {
				found = true
				break
			}
		}
		// add the file at first position
		if !found {
			cli.DefaultFileNames = append([]string{filename}, cli.DefaultFileNames...)
		}
	}

	return p
}

// Parse using compose-go parser, adapt a bit the Project and set Appname.
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
		log.Fatal("Failed to create project", err)
	}

	Appname = proj.Name
	p.Data = proj
	if p.temporary != nil {
		defer os.RemoveAll(*p.temporary)
	}
}
