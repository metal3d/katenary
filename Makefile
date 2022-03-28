CUR_SHA=$(shell git log -n1 --pretty='%h')
CUR_BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --exact-match --tags $(CUR_SHA) 2>/dev/null || echo $(CUR_BRANCH)-$(CUR_SHA))
CTN:=$(shell which podman 2>&1 1>/dev/null && echo "podman" || echo "docker")
PREFIX=~/.local

GO=container
BLD_CMD=go build -o katenary  -ldflags="-X 'main.Version=$(VERSION)'" ./cmd/*.go
GOOS=linux
GOARCH=amd64

BUILD_IMAGE=docker.io/golang:1.17

.PHONY: help clean build

.ONESHELL:
help:
	@cat <<EOF
	=== HELP ===
	To avoid you to install Go, the build is made by podman or docker.
	
	You can use:
	$$ make install
	This will build and install katenary inside the PREFIX(/bin) value (default is $(PREFIX))
	
	To change the PREFIX to somewhere where only root or sudo users can save the binary, it is recommended to build before install:
	$$ make build
	$$ sudo make install PREFIX=/usr/local
	
	Katenary is statically built (in Go), so there is no library to install.
	
	To build for others OS:
	$$ make build GOOS=linux GOARCH=amd64
	This will build the binary for linux amd64.
	
	$$ make build GOOS=linux GOARCH=arm
	This will build the binary for linux arm.
	
	$$ make build GOOS=windows GOARCH=amd64
	This will build the binary for windows amd64.
	
	$$ make build GOOS=darwin GOARCH=amd64
	This will build the binary for darwin amd64.
	
	Or you can build all versions:
	$$ make build-all
	EOF



build: pull katenary

build-all: pull dist dist/katenary-linux-amd64 dist/katenary-linux-arm64 dist/katenary.exe dist/katenary-darwin

pull:
ifneq ($(GO),local)
	@echo -e "\033[1;32mPulling $(BUILD_IMAGE) docker image\033[0m"
	@$(CTN) pull $(BUILD_IMAGE)
endif

dist:
	mkdir -p dist

dist/katenary-linux-amd64:
	@echo -e "\033[1;32mBuilding katenary for linux-amd64...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=amd64
	strip katenary
	mv katenary dist/katenary-linux-amd64


dist/katenary-linux-arm64:
	@echo -e "\033[1;32mBuilding katenary for linux-arm...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=arm64
	strip katenary
	mv katenary dist/katenary-linux-arm64

dist/katenary.exe:
	@echo -e "\033[1;32mBuilding katenary for windows...\033[0m"
	$(MAKE) katenary GOOS=windows GOARCH=amd64
	strip katenary
	mv katenary dist/katenary-windows.exe

dist/katenary-darwin:
	@echo -e "\033[1;32mBuilding katenary for darwin...\033[0m"
	$(MAKE) katenary GOOS=darwin GOARCH=amd64
	strip katenary
	mv katenary dist/katenary-darwin

katenary: $(wildcard */*.go Makefile go.mod go.sum)
	@echo -e "\033[1;33mBuilding katenary $(VERSION)...\033[0m"
ifeq ($(GO),local)
	@echo "=> Build in host using go"
	@echo
else
	@echo "=> Build in container using" $(CTN)
	@echo
endif
	echo $(BLD_CMD)
ifeq ($(GO),local)
	$(BLD_CMD)
else ifeq ($(CTN),podman)
	@podman run -e CGO_ENABLED=0 -e GOOS=$(GOOS) -e GOARCH=$(GOARCH) --rm -v $(PWD):/go/src/katenary:z -w /go/src/katenary --userns keep-id -it docker.io/golang $(BLD_CMD)
else
	@docker run -e CGO_ENABLED=0 -e GOOS=$(GOOS) -e GOARCH=$(GOARCH) --rm -v $(PWD):/go/src/katenary:z -w /go/src/katenary --user $(shell id -u):$(shell id -g) -e HOME=/tmp -it docker.io/golang $(BLD_CMD)
endif
	


install: build
	cp katenary $(PREFIX)/bin/katenary

uninstall:
	rm -f $(PREFIX)/bin/katenary

clean:
	rm -f katenary
	rm -rf dist

test:
	@echo -e "\033[1;33mTesting katenary $(VERSION)...\033[0m"
	go test -v ./...
