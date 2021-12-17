package main

import (
	"flag"
	"fmt"
	"katenary/compose"
	"katenary/generator"
	"katenary/helm"
	"os"
	"path/filepath"
	"strings"
)

var ComposeFile = "docker-compose.yaml"
var AppName = "MyApp"
var AppVersion = "0.0.1"
var Version = "master" // set at build time to the git version/tag
var ChartsDir = "chart"

func main() {
	flag.StringVar(&ChartsDir, "chart-dir", ChartsDir, "set the chart directory")
	flag.StringVar(&ComposeFile, "compose", ComposeFile, "set the compose file to parse")
	flag.StringVar(&AppName, "appname", helm.GetProjectName(), "set the helm chart app name")
	flag.StringVar(&AppVersion, "appversion", AppVersion, "set the chart appVersion")
	version := flag.Bool("version", false, "show version and exit")
	force := flag.Bool("force", false, "force the removal of the chart-dir")
	flag.Parse()

	// Only display the version
	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}

	dirname := filepath.Join(ChartsDir, AppName)
	if _, err := os.Stat(dirname); err == nil && !*force {
		response := ""
		for response != "y" && response != "n" {
			response = "n"
			fmt.Printf(""+
				"The %s directory already exists, it will be \x1b[31;1mremoved\x1b[0m!\n"+
				"Do you really want to continue? [y/N]: ", dirname)
			fmt.Scanf("%s", &response)
			response = strings.ToLower(response)
		}
		if response == "n" {
			fmt.Println("Cancelled")
			os.Exit(0)
		}
	}

	// cleanup and create the chart directory (until "templates")
	os.RemoveAll(dirname)
	templatesDir := filepath.Join(dirname, "templates")
	os.MkdirAll(templatesDir, 0755)

	// Parse the compose file now
	p := compose.NewParser(ComposeFile)
	p.Parse(AppName)
	generator.Generate(p, Version, AppName, AppVersion, ComposeFile, dirname)

}
