package main

import (
	"errors"
	"flag"
	"fmt"
	"katenary/compose"
	"katenary/generator"
	"katenary/helm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var composeFiles = []string{"docker-compose.yaml", "docker-compose.yml"}
var ComposeFile = ""
var AppName = "MyApp"
var Version = "master" // set at build time to the git version/tag
var ChartsDir = "chart"

func findComposeFile() {
	for _, file := range composeFiles {
		if _, err := os.Stat(file); err == nil {
			ComposeFile = file
			return
		}
	}
	fmt.Printf("No compose file found in %s\n", composeFiles)
	os.Exit(1)
}

func detectGitVersion() (string, error) {
	defaulVersion := "0.0.1"
	// Check if .git directory exists
	if s, err := os.Stat(".git"); err != nil {
		// .git should be a directory
		return defaulVersion, errors.New("no git repository found")
	} else if !s.IsDir() {
		// .git should be a directory
		return defaulVersion, errors.New(".git is not a directory")
	}

	// check if "git" executable is callable
	if _, err := exec.LookPath("git"); err != nil {
		return defaulVersion, errors.New("git executable not found")
	}

	// get the latest commit hash
	if out, err := exec.Command("git", "log", "-n1", "--pretty=format:%h").Output(); err == nil {
		latestCommit := strings.TrimSpace(string(out))
		// then get the current branch/tag
		out, err := exec.Command("git", "branch", "--show-current").Output()
		if err != nil {
			return defaulVersion, errors.New("git branch --show-current failed")
		} else {
			currentBranch := strings.TrimSpace(string(out))
			// finally, check if the current tag (if exists) correspond to the current commit
			// git describe --exact-match --tags <latestCommit>
			out, err := exec.Command("git", "describe", "--exact-match", "--tags", latestCommit).Output()
			if err == nil {
				return strings.TrimSpace(string(out)), nil
			} else {
				return currentBranch + "-" + latestCommit, nil
			}
		}
	}
	return defaulVersion, errors.New("git log failed")
}

func main() {

	appVersion := "0.0.1"
	helpMessageForAppversion := "The version of the application. " +
		"Default is 0.0.1. If you are using git, it will be the git version. " +
		"Otherwise, it will be the branch name and the commit hash."
	if v, err := detectGitVersion(); err == nil {
		appVersion = v
		helpMessageForAppversion = "The version of the application. " +
			"If not set, the version will be detected from git."
	}

	// flags
	findComposeFile()
	flag.StringVar(&ChartsDir, "chart-dir", ChartsDir, "set the chart directory")
	flag.StringVar(&ComposeFile, "compose", ComposeFile, "set the compose file to parse")
	flag.StringVar(&AppName, "appname", helm.GetProjectName(), "set the helm chart app name")
	flag.StringVar(&appVersion, "appversion", appVersion, helpMessageForAppversion)
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
	if err := os.RemoveAll(dirname); err != nil {
		fmt.Printf("Error removing %s: %s\n", dirname, err)
		os.Exit(1)
	}

	// create the templates directory
	templatesDir := filepath.Join(dirname, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		fmt.Printf("Error creating %s: %s\n", templatesDir, err)
		os.Exit(1)
	}

	// Parse the compose file now
	p := compose.NewParser(ComposeFile)
	p.Parse(AppName)

	// start generator
	generator.Generate(p, Version, AppName, appVersion, ComposeFile, dirname)

}
