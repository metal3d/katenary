package compose

import "testing"

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

`

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
