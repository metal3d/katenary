package generator

import (
	"io/ioutil"
	"katenary/compose"
	"katenary/helm"
	"katenary/logger"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/cli"
)

const DOCKER_COMPOSE_YML = `version: '3'
services:
    # first service, very simple
    http:
        image: nginx
        ports:
            - "80:80"

    # second service, with environment variables
    http2:
        image: nginx
        environment:
            SOME_ENV_VAR: some_value
            ANOTHER_ENV_VAR: another_value

    # third service with ingress label
    web:
        image: nginx
        ports:
            - "80:80"
        labels:
            katenary.io/ingress: 80

    web2:
        image: nginx
        command: ["/bin/sh", "-c", "while true; do echo hello; sleep 1; done"]

    # fourth service is a php service depending on database
    php:
        image: php:7.2-apache
        depends_on:
            - database
        environment:
            SOME_ENV_VAR: some_value
            ANOTHER_ENV_VAR: another_value
            DB_HOST: database
        labels:
            katenary.io/env-to-service: DB_HOST

    database:
        image: mysql:5.7
        environment:
            MYSQL_ROOT_PASSWORD: root
            MYSQL_DATABASE: database
            MYSQL_USER: user
            MYSQL_PASSWORD: password
        volumes:
            - data:/var/lib/mysql
        labels:
            katenary.io/ports: 3306


    # try to deploy 2 services but one is in the same pod than the other
    http3:
        image: nginx

    http4:
        image: nginx
        labels:
            katenary.io/same-pod: http3

    # unmapped volumes
    novol:
        image: nginx
        volumes:
            - /tmp/data
        labels:
            katenary.io/ports: 80

    # use = sign for environment variables
    eqenv:
        image: nginx
        environment:
          - SOME_ENV_VAR=some_value
          - ANOTHER_ENV_VAR=another_value

volumes:
    data:
        driver: local
`

var defaultCliFiles = cli.DefaultFileNames

func init() {
	logger.NOLOG = false
}

func setUp(t *testing.T) (string, *compose.Parser) {
	cli.DefaultFileNames = defaultCliFiles
	p := compose.NewParser("", DOCKER_COMPOSE_YML)
	p.Parse("testapp")

	// create a temporary directory
	tmp, err := os.MkdirTemp(os.TempDir(), "katenary-test")
	if err != nil {
		t.Fatal(err)
	}

	Generate(p, "test-0", "testapp", "1.2.3", DOCKER_COMPOSE_YML, tmp)

	return tmp, p
}

// Check if the web2 service has got a command.
func TestCommand(t *testing.T) {
	tmp, p := setUp(t)
	defer os.RemoveAll(tmp)

	for _, service := range p.Data.Services {
		name := service.Name
		if name == "web2" {
			// Ensure that the command is correctly set
			// The command should be a string array
			path := filepath.Join(tmp, "templates", name+".deployment.yaml")
			path = filepath.Join(tmp, "templates", name+".deployment.yaml")
			fp, _ := os.Open(path)
			defer fp.Close()
			lines, _ := ioutil.ReadAll(fp)
			next := false
			commands := make([]string, 0)
			for _, line := range strings.Split(string(lines), "\n") {
				if strings.Contains(line, "command") {
					next = true
					continue
				}
				if next {
					commands = append(commands, line)
				}
			}
			//log.Printf("%#v\n", commands)
			ok := 0
			for _, command := range commands {
				if strings.Contains(command, "- /bin/sh") {
					ok++
				}
				if strings.Contains(command, "- -c") {
					ok++
				}
				if strings.Contains(command, "while true; do") {
					ok++
				}
			}
			if ok != 3 {
				t.Error("Command is not correctly set")
			}
		}
	}
}

// Check if environment is correctly set.
func TestEnvs(t *testing.T) {
	tmp, p := setUp(t)
	defer os.RemoveAll(tmp)

	for _, service := range p.Data.Services {
		name := service.Name

		if name == "php" {
			// the "DB_HOST" environment variable inside the template must be set to '{{ .Release.Name }}-database'
			path := filepath.Join(tmp, "templates", name+".deployment.yaml")
			// read the file and find the DB_HOST variable
			matched := false
			fp, _ := os.Open(path)
			defer fp.Close()
			lines, _ := ioutil.ReadAll(fp)
			next := false
			for _, line := range strings.Split(string(lines), "\n") {
				if strings.Contains(line, "DB_HOST") {
					next = true
					continue
				}
				if next {
					matched = true
					if !strings.Contains(line, helm.RELEASE_NAME+"-database") {
						t.Error("DB_HOST variable should be set to " + helm.RELEASE_NAME + "-database")
					}
					break
				}
			}
			if !matched {
				t.Error("DB_HOST variable not found in ", path)
			}
		}
	}
}

// Check if the same pod is not deployed twice.
func TestSamePod(t *testing.T) {
	tmp, p := setUp(t)
	defer os.RemoveAll(tmp)

	for _, service := range p.Data.Services {
		name := service.Name
		path := filepath.Join(tmp, "templates", name+".deployment.yaml")

		if _, found := service.Labels[helm.LABEL_SAMEPOD]; found {
			// fail if the service has a deployment
			if _, err := os.Stat(path); err == nil {
				t.Error("Service ", name, " should not have a deployment")
			}
			continue
		}

		// others should have a deployment file
		t.Log("Checking ", name, " deployment file")
		_, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// Check if the ports are correctly set.
func TestPorts(t *testing.T) {
	tmp, p := setUp(t)
	defer os.RemoveAll(tmp)

	for _, service := range p.Data.Services {
		name := service.Name
		path := ""

		// if the service has a port found in helm.LABEL_PORT or ports, so the service file should exist
		hasPort := false
		if _, found := service.Labels[helm.LABEL_PORT]; found {
			hasPort = true
		}
		if service.Ports != nil {
			hasPort = true
		}
		if hasPort {
			path = filepath.Join(tmp, "templates", name+".service.yaml")
			t.Log("Checking ", name, " service file")
			_, err := os.Stat(path)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

// Check if the volumes are correctly set.
func TestPVC(t *testing.T) {
	tmp, p := setUp(t)
	defer os.RemoveAll(tmp)

	for _, service := range p.Data.Services {
		name := service.Name
		path := filepath.Join(tmp, "templates", name+"-data.pvc.yaml")

		// the "database" service should have a pvc file in templates (name-data.pvc.yaml)
		if name == "database" {
			path = filepath.Join(tmp, "templates", name+"-data.pvc.yaml")
			t.Log("Checking ", name, " pvc file")
			_, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

//Check if web service has got a ingress.
func TestIngress(t *testing.T) {
	tmp, p := setUp(t)
	defer os.RemoveAll(tmp)

	for _, service := range p.Data.Services {
		name := service.Name
		path := filepath.Join(tmp, "templates", name+".ingress.yaml")

		// the "web" service should have a ingress file in templates (name.ingress.yaml)
		if name == "web" {
			path = filepath.Join(tmp, "templates", name+".ingress.yaml")
			t.Log("Checking ", name, " ingress file")
			_, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

// Check unmapped volumes
func TestUnmappedVolumes(t *testing.T) {
	tmp, p := setUp(t)
	defer os.RemoveAll(tmp)

	for _, service := range p.Data.Services {
		name := service.Name
		if name == "novol" {
			path := filepath.Join(tmp, "templates", name+".deployment.yaml")
			fp, _ := os.Open(path)
			defer fp.Close()
			lines, _ := ioutil.ReadAll(fp)
			for _, line := range strings.Split(string(lines), "\n") {
				if strings.Contains(line, "novol-data") {
					t.Error("novol service should not have a volume")
				}
			}
		}
	}
}

// Check if service using equal sign for environment works
func TestEqualSignOnEnv(t *testing.T) {
	tmp, p := setUp(t)
	defer os.RemoveAll(tmp)

	// if the name is eqenv, the service should habe environment
	for _, service := range p.Data.Services {
		name := service.Name
		if name == "eqenv" {
			path := filepath.Join(tmp, "templates", name+".deployment.yaml")
			fp, _ := os.Open(path)
			defer fp.Close()
			lines, _ := ioutil.ReadAll(fp)
			match := 0
			for _, line := range strings.Split(string(lines), "\n") {
				// we must find the line with the environment variable name
				if strings.Contains(line, "SOME_ENV_VAR") {
					// we must find the line with the environment variable value
					match++
				}
				if strings.Contains(line, "ANOTHER_ENV_VAR") {
					// we must find the line with the environment variable value
					match++
				}
			}
			if match != 2 {
				t.Error("eqenv service should have 2 environment variables")
			}
		}
	}
}
