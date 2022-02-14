package main

import (
	"katenary/cmd"
	"katenary/helm"

	"github.com/spf13/cobra"
)

var Version = "master" // changed at compile time

var longHelp = `Katenary aims to be a tool to convert docker-compose files to Helm Charts. 
It will create deployments, services, volumes, secrets, and ingress resources.
But it will also create initContainers based on depend_on, healthcheck, and other features.
It's not magical, sometimes you'll need to fix the generated charts.
The general way to use it is to call one of these commands:

    katenary convert
    katenary convert -f docker-compose.yml
    katenary convert -f docker-compose.yml -o ./charts

In case of, check the help of each command using:
    katenary <command> --help
or
    "katenary help <command>"
`

func main() {

	// set the version to "cmd" package - yes... it's ugly
	cmd.Version = Version

	// The base command
	rootCmd := &cobra.Command{
		Use:   "katenary",
		Long:  longHelp,
		Short: "Katenary is a tool to convert docker-compose files to Helm Charts",
	}

	// to display the version
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display version",
		Run:   func(c *cobra.Command, args []string) { c.Println(Version) },
	}

	// convert command, need some flags
	convertCmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert docker-compose to helm chart",
		Long: "Convert docker-compose to helm chart. The resulting helm chart will be in the current directory/" +
			cmd.ChartsDir + "/" + cmd.AppName +
			".\nThe appversion will be produced that waty:\n" +
			"- from git version or tag\n" +
			"- if it's not defined, so the version will be get from the --apVersion flag \n" +
			"- if it's not defined, so the 0.0.1 version is used",
		Run: func(c *cobra.Command, args []string) {
			force := c.Flag("force").Changed
			appversion := c.Flag("app-version").Value.String()
			composeFile := c.Flag("compose-file").Value.String()
			appName := c.Flag("app-name").Value.String()
			chartDir := c.Flag("output-dir").Value.String()
			cmd.Convert(composeFile, appversion, appName, chartDir, force)
		},
	}
	convertCmd.Flags().BoolP(
		"force", "f", false, "Force overwrite of existing files")
	convertCmd.Flags().StringP(
		"app-version", "a", cmd.AppVersion, "App version")
	convertCmd.Flags().StringP(
		"compose-file", "c", cmd.ComposeFile, "Docker compose file")
	convertCmd.Flags().StringP(
		"app-name", "n", cmd.AppName, "Application name")
	convertCmd.Flags().StringP(
		"output-dir", "o", cmd.ChartsDir, "Chart directory")

	// show possible labels to set in docker-compose file
	showLabelsCmd := &cobra.Command{
		Use:   "show-labels",
		Short: "Show labels of a resource",
		Run: func(c *cobra.Command, args []string) {
			c.Println(helm.GetLabelsDocumentation())
		},
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(showLabelsCmd)

	rootCmd.Execute()

}
