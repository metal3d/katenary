package compose

import (
	"io/ioutil"
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

var (
	Appname        = ""
	CURRENT_DIR, _ = os.Getwd()
)

// NewParser create a Parser and parse the file given in filename. If filename is empty, we try to parse the content[0] argument that should be a valid YAML content.
func NewParser(filename []string, content ...string) *Parser {

	p := &Parser{}

	if len(content) > 0 { // mainly for the tests...
		dir := filepath.Dir(filename[0])
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatal(err)
		}
		p.temporary = &dir
		ioutil.WriteFile(filename[0], []byte(content[0]), 0644)
		cli.DefaultFileNames = filename
	}
	// if filename is not in cli Default files, add it
	if len(filename) > 0 {
		found := false
		for _, defaultFileName := range cli.DefaultFileNames {
			for _, givenFileName := range filename {
				if defaultFileName == givenFileName {
					found = true
					break
				}
			}
		}
		// add the file at first position
		if !found {
			cli.DefaultFileNames = append([]string{filename[0]}, cli.DefaultFileNames...)
		}
		if len(filename) > 1 {
			cli.DefaultOverrideFileNames = append(filename[1:], cli.DefaultOverrideFileNames...)
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
	CURRENT_DIR = p.Data.WorkingDir
}

func GetCurrentDir() string {
	return CURRENT_DIR
}
