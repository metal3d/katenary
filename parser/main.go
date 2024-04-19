// Parser package is a wrapper around compose-go to parse compose files.
package parser

import (
	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
)

func init() {
	// prepend compose.katenary.yaml to the list of default override file names
	cli.DefaultOverrideFileNames = append([]string{
		"compose.katenary.yml",
		"compose.katenary.yaml",
	}, cli.DefaultOverrideFileNames...)
	cli.DefaultOverrideFileNames = append(cli.DefaultOverrideFileNames,
		[]string{
			"podman-compose.katenary.yml",
			"podman-compose.katenary.yaml",
			"podman-compose.yml",
			"podman-compose.yaml",
		}...)
}

// Parse compose files and return a project. The project is parsed with dotenv, osenv and profiles.
func Parse(profiles []string, dockerComposeFile ...string) (*types.Project, error) {
	if len(dockerComposeFile) == 0 {
		cli.DefaultOverrideFileNames = append(cli.DefaultOverrideFileNames, dockerComposeFile...)
	}

	options, err := cli.NewProjectOptions(nil,
		cli.WithProfiles(profiles),
		cli.WithDefaultConfigPath,
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithNormalization(true),
		cli.WithInterpolation(true),
		cli.WithResolvedPaths(false),
	)
	if err != nil {
		return nil, err
	}
	return cli.ProjectFromOptions(options)
}
