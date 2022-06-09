package main

import (
	"fmt"
	"katenary/generator/writers"
	"katenary/helm"
	"katenary/update"
	"log"
	"strconv"

	"github.com/spf13/cobra"
)

var Version = "master" // changed at compile time

var longHelp = `Katenary aims to be a tool to convert docker-compose files to Helm Charts. 
It will create deployments, services, volumes, secrets, and ingress resources.
But it will also create initContainers based on depend_on, healthcheck, and other features.
It's not magical, sometimes you'll need to fix the generated charts.
The general way to use it is to call one of these commands:

    katenary convert
    katenary convert -c docker-compose.yml
    katenary convert -c docker-compose.yml -o ./charts

In case of, check the help of each command using:
    katenary <command> --help
or
    "katenary help <command>"
`

func init() {
	// apply the version to the "update" package
	update.Version = Version
}

func main() {

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
	var composeFiles *[]string
	convertCmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert docker-compose to helm chart",
		Long: "Convert docker-compose to helm chart. The resulting helm chart will be in the current directory/" +
			ChartsDir + "/" + AppName +
			".\nThe appversion will be generated that way:\n" +
			"- if it's in a git project, it takes git version or tag\n" +
			"- if it's not defined, so the version will be get from the --app-version flag \n" +
			"- if it's not defined, so the 0.0.1 version is used",
		Run: func(c *cobra.Command, args []string) {
			force := c.Flag("force").Changed
			appversion := c.Flag("app-version").Value.String()
			appName := c.Flag("app-name").Value.String()
			chartVersion := c.Flag("chart-version").Value.String()
			chartDir := c.Flag("output-dir").Value.String()
			indentation, err := strconv.Atoi(c.Flag("indent-size").Value.String())
			if err != nil {
				writers.IndentSize = indentation
			}
			log.Println(composeFiles)
			Convert(*composeFiles, appversion, appName, chartDir, chartVersion, force)
		},
	}
	composeFiles = convertCmd.Flags().StringArrayP(
		"compose-file", "c", []string{ComposeFile}, "compose file to convert, can be use several times to override previous file. Order is important!")
	convertCmd.Flags().BoolP(
		"force", "f", false, "force overwrite of existing output files")
	convertCmd.Flags().StringP(
		"app-version", "a", AppVersion, "app version")
	convertCmd.Flags().StringP(
		"chart-version", "v", ChartVersion, "chart version")
	convertCmd.Flags().StringP(
		"app-name", "n", AppName, "application name")
	convertCmd.Flags().StringP(
		"output-dir", "o", ChartsDir, "chart directory")
	convertCmd.Flags().IntP(
		"indent-size", "i", 2, "set the indent size of the YAML files")

	// show possible labels to set in docker-compose file
	showLabelsCmd := &cobra.Command{
		Use:   "show-labels",
		Short: "Show labels of a resource",
		Run: func(c *cobra.Command, args []string) {
			c.Println(helm.GetLabelsDocumentation())
		},
	}

	// Update the binary to the latest version
	updateCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade katenary to the latest version if available",
		Run: func(c *cobra.Command, args []string) {
			version, assets, err := update.CheckLatestVersion()
			if err != nil {
				c.Println(err)
				return
			}
			c.Println("Updating to version: " + version)
			err = update.DownloadLatestVersion(assets)
			if err != nil {
				c.Println(err)
				return
			}
			c.Println("Update completed")
		},
	}

	rootCmd.AddCommand(
		versionCmd,
		convertCmd,
		showLabelsCmd,
		updateCmd,
	)

	// in parallel, check if the current katenary version is the latest
	ch := make(chan string)
	go func() {
		version, _, err := update.CheckLatestVersion()
		if err != nil {
			ch <- ""
			return
		}
		if Version != version {
			ch <- fmt.Sprintf("\x1b[33mNew version available: " +
				version +
				" - to auto upgrade katenary, you can execute: katenary upgrade\x1b[0m\n")
		}
	}()

	// Execute the command
	finalize := make(chan error)
	go func() {
		finalize <- rootCmd.Execute()
	}()

	// Wait for both goroutines to finish
	if err := <-finalize; err != nil {
		fmt.Println(err)
	}
	fmt.Print(<-ch)
}
