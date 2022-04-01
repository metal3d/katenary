package compose

import (
	"katenary/logger"
	"testing"
)

const DOCKER_COMPOSE_YML1 = `
version: "3"

services:
    # first service, very basic
    web:
        image: nginx
        ports:
            - "80:80"
        environment:
            FOO: bar
            BAZ: qux
        networks:
            - frontend


    database:
        image: postgres
        networks:
            - frontend
        environment:
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: mysecretpassword
          POSTGRES_DB: mydb
        labels:
            katenary.io/ports: "5432"

    commander1:
        image: foo
        command: ["/bin/sh", "-c", "echo 'hello world'"]

    commander2:
        image: foo
        command: echo "hello world"

    hc1:
        image: foo
        healthcheck:
            test: ["CMD-SHELL", "echo 'hello world1'"]

    hc2:
        image: foo
        healthcheck:
            test: echo "hello world2"

    hc3:
        image: foo
        healthcheck:
            test: ["CMD", "echo 'hello world3'"]


`

func init() {
	logger.NOLOG = true
}

func TestParser(t *testing.T) {
	p := NewParser("", DOCKER_COMPOSE_YML1)
	p.Parse("test")

	// check if the "web" and "database" service is parsed correctly
	// by checking if the "ports" and "environment"
	for name, service := range p.Data.Services {
		if name == "web" {
			if len(service.Ports) != 1 {
				t.Errorf("Expected 1 port, got %d", len(service.Ports))
			}
			if service.Ports[0] != "80:80" {
				t.Errorf("Expected port 80:80, got %s", service.Ports[0])
			}
			if len(service.Environment) != 2 {
				t.Errorf("Expected 2 environment variables, got %d", len(service.Environment))
			}
			if service.Environment["FOO"] != "bar" {
				t.Errorf("Expected FOO=bar, got %s", service.Environment["FOO"])
			}
			if service.Environment["BAZ"] != "qux" {
				t.Errorf("Expected BAZ=qux, got %s", service.Environment["BAZ"])
			}
		}
		// same for the "database" service
		if name == "database" {
			if len(service.Ports) != 1 {
				t.Errorf("Expected 1 port, got %d", len(service.Ports))
			}
			if service.Ports[0] != "5432" {
				t.Errorf("Expected port 5432, got %s", service.Ports[0])
			}
			if len(service.Environment) != 3 {
				t.Errorf("Expected 3 environment variables, got %d", len(service.Environment))
			}
			if service.Environment["POSTGRES_USER"] != "postgres" {
				t.Errorf("Expected POSTGRES_USER=postgres, got %s", service.Environment["POSTGRES_USER"])
			}
			if service.Environment["POSTGRES_PASSWORD"] != "mysecretpassword" {
				t.Errorf("Expected POSTGRES_PASSWORD=mysecretpassword, got %s", service.Environment["POSTGRES_PASSWORD"])
			}
			if service.Environment["POSTGRES_DB"] != "mydb" {
				t.Errorf("Expected POSTGRES_DB=mydb, got %s", service.Environment["POSTGRES_DB"])
			}
			// check labels
			if len(service.Labels) != 1 {
				t.Errorf("Expected 1 label, got %d", len(service.Labels))
			}
			// is label katenary.io/ports correct?
			if service.Labels["katenary.io/ports"] != "5432" {
				t.Errorf("Expected katenary.io/ports=5432, got %s", service.Labels["katenary.io/ports"])
			}
		}
	}
}

func TestParseCommand(t *testing.T) {
	p := NewParser("", DOCKER_COMPOSE_YML1)
	p.Parse("test")

	for name, s := range p.Data.Services {
		if name == "commander1" {
			t.Log(s.Command)
			if len(s.Command) != 3 {
				t.Errorf("Expected 3 command, got %d", len(s.Command))
			}
			if s.Command[0] != "/bin/sh" {
				t.Errorf("Expected /bin/sh, got %s", s.Command[0])
			}
			if s.Command[1] != "-c" {
				t.Errorf("Expected -c, got %s", s.Command[1])
			}
			if s.Command[2] != "echo 'hello world'" {
				t.Errorf("Expected echo 'hello world', got %s", s.Command[2])
			}
		}
		if name == "commander2" {
			t.Log(s.Command)
			if len(s.Command) != 2 {
				t.Errorf("Expected 1 command, got %d", len(s.Command))
			}
			if s.Command[0] != "echo" {
				t.Errorf("Expected echo, got %s", s.Command[0])
			}
			if s.Command[1] != "hello world" {
				t.Errorf("Expected hello world, got %s", s.Command[1])
			}
		}
	}
}

func TestHealthChecks(t *testing.T) {
	p := NewParser("", DOCKER_COMPOSE_YML1)
	p.Parse("test")

	for name, s := range p.Data.Services {
		if name != "hc1" && name != "hc2" && name != "hc3" {
			continue
		}

		if name == "hc1" {
			if len(s.HealthCheck.Test) != 2 {
				t.Errorf("Expected 2 healthcheck tests, got %d", len(s.HealthCheck.Test))
			}
			if s.HealthCheck.Test[0] != "CMD-SHELL" {
				t.Errorf("Expected CMD-SHELL, got %s", s.HealthCheck.Test[0])
			}
			if s.HealthCheck.Test[1] != "echo 'hello world1'" {
				t.Errorf("Expected echo 'hello world1', got %s", s.HealthCheck.Test[1])
			}
		}
		if name == "hc2" {
			if len(s.HealthCheck.Test) != 2 {
				t.Errorf("Expected 2 healthcheck tests, got %d", len(s.HealthCheck.Test))
			}
			if s.HealthCheck.Test[0] != "echo" {
				t.Errorf("Expected echo, got %s", s.HealthCheck.Test[1])
			}
			if s.HealthCheck.Test[1] != "hello world2" {
				t.Errorf("Expected echo 'hello world2', got %s", s.HealthCheck.Test[1])
			}
		}
		if name == "hc3" {
			if len(s.HealthCheck.Test) != 2 {
				t.Errorf("Expected 2 healthcheck tests, got %d", len(s.HealthCheck.Test))
			}
			if s.HealthCheck.Test[0] != "CMD" {
				t.Errorf("Expected CMD, got %s", s.HealthCheck.Test[0])
			}
			if s.HealthCheck.Test[1] != "echo 'hello world3'" {
				t.Errorf("Expected echo 'hello world3', got %s", s.HealthCheck.Test[1])
			}
		}
	}
}
