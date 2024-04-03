// Parser package is a wrapper around compose-go to parse compose files.
package parser

import (
	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
)

// Parse compose files and return a project. The project is parsed with dotenv, osenv and profiles.
func Parse(profiles []string, dockerComposeFile ...string) (*types.Project, error) {

	cli.DefaultOverrideFileNames = append(cli.DefaultOverrideFileNames, "compose.katenary.yaml")

	options, err := cli.NewProjectOptions(nil,
		cli.WithProfiles(profiles),
		cli.WithDefaultConfigPath,
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithNormalization(true),
		cli.WithInterpolation(true),
		cli.WithResolvedPaths(false),

		//cli.WithResolvedPaths(true),
	)
	if err != nil {
		return nil, err
	}
	return cli.ProjectFromOptions(options)
}
