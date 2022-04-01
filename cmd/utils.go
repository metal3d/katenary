package main

import (
	"errors"
	"fmt"
	"katenary/compose"
	"katenary/generator"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	composeFiles = []string{"docker-compose.yaml", "docker-compose.yml"}
	ComposeFile  = ""
	AppName      = "MyApp"
	ChartsDir    = "chart"
	AppVersion   = "0.0.1"
)

func init() {
	FindComposeFile()
	SetAppName()
	SetAppVersion()
}

func FindComposeFile() bool {
	for _, file := range composeFiles {
		if _, err := os.Stat(file); err == nil {
			ComposeFile = file
			return true
		}
	}
	return false
}

// SetAppName sets the application name from the current directory name.
func SetAppName() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	AppName = filepath.Base(wd)

	if AppName == "" {
		AppName = "MyApp"
	}
}

// SetAppVersion set the AppVersion variable to the git version/tag
func SetAppVersion() {
	AppVersion, _ = detectGitVersion()
}

// Try to detect the git version/tag.
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

func Convert(composeFile, appVersion, appName, chartDir string, force bool) {
	if len(composeFile) == 0 {
		fmt.Println("No compose file given")
		return
	}
	_, err := os.Stat(ComposeFile)
	if err != nil {
		fmt.Println("No compose file found")
		os.Exit(1)
	}

	dirname := filepath.Join(chartDir, appName)
	if _, err := os.Stat(dirname); err == nil && !force {
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
	p := compose.NewParser(composeFile)
	p.Parse(appName)

	// start generator
	generator.Generate(p, Version, appName, appVersion, ComposeFile, dirname)

}
