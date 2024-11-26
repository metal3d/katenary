CUR_SHA=$(shell git log -n1 --pretty='%h')
CUR_BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --exact-match --tags $(CUR_SHA) 2>/dev/null || echo $(CUR_BRANCH)-$(CUR_SHA))
CTN:=$(shell which podman 2>&1 1>/dev/null && echo "podman" || echo "docker")
PREFIX=~/.local

GOVERSION=1.23
GO=container
OUT=katenary
RELEASE=""
BLD_CMD=go build -ldflags="-X 'katenary/generator.Version=$(RELEASE)$(VERSION)'" -o $(OUT)  ./cmd/katenary
GOOS=linux
GOARCH=amd64
SIGNER=metal3d@gmail.com

BUILD_IMAGE=docker.io/golang:$(GOVERSION)-alpine
# SHELL=/bin/bash

# List of source files
SOURCES=$(wildcard ./*.go ./*/*.go ./*/*/*.go)
# List of binaries to build and sign
BINARIES=dist/katenary-linux-amd64 dist/katenary-linux-arm64 dist/katenary.exe dist/katenary-darwin-amd64 dist/katenary-freebsd-amd64 dist/katenary-freebsd-arm64
# List of signatures to build
ASC_BINARIES=$(patsubst %,%.asc,$(BINARIES))

# defaults
BROWSER=$(shell command -v epiphany || echo xdg-open)
SHELL := bash
# strict mode
.SHELLFLAGS := -eu -o pipefail -c
# One session per target
.ONESHELL:
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
.PHONY: help clean build install tests test

all: build


help:
	@cat <<EOF | fold -s -w 80
	=== HELP ===
	To avoid you to install Go, the build is made by podman or docker.
	
	Installinf (you can use local Go by setting GO=local)):
	# use podman or docker to build
	$$ make install
	# or use local Go
	$$ make install GO=local
	This will build and install katenary inside the PREFIX(/bin) value (default is $(PREFIX))
	
	
	To change the PREFIX to somewhere where only root or sudo users can save the binary, it is recommended to build before install, one more time you can use local Go by setting GO=local:
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


## Standard build
build: pull katenary

pull:
ifneq ($(GO),local)
	@echo -e "\033[1;32mPulling $(BUILD_IMAGE) docker image\033[0m"
	@$(CTN) pull $(BUILD_IMAGE)
endif

katenary: $(SOURCES) Makefile go.mod go.sum
ifeq ($(GO),local)
	@echo "=> Build on host using go"
else
	@echo "=> Build in container using" $(CTN)
endif
	echo $(BLD_CMD)
ifeq ($(GO),local)
	$(BLD_CMD)
else ifeq ($(CTN),podman)
	@podman run -e CGO_ENABLED=0 -e GOOS=$(GOOS) -e GOARCH=$(GOARCH) \
		--rm -v $(PWD):/go/src/katenary:z -w /go/src/katenary --userns keep-id -it $(BUILD_IMAGE) $(BLD_CMD)
else
	@docker run -e CGO_ENABLED=0 -e GOOS=$(GOOS) -e GOARCH=$(GOARCH) \
		--rm -v $(PWD):/go/src/katenary:z -w /go/src/katenary --user $(shell id -u):$(shell id -g) -e HOME=/tmp -it $(BUILD_IMAGE) $(BLD_CMD)
endif
	echo "=> Stripping if possible"
	strip $(OUT) 2>/dev/null || echo "=> No strip available"


## Release build
dist: prepare $(BINARIES) $(ASC_BINARIES)

prepare: pull
	mkdir -p dist

dist/katenary-linux-amd64:
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for linux-amd64...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=amd64 OUT=$@

dist/katenary-linux-arm64:
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for linux-arm...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=arm64 OUT=$@

dist/katenary.exe:
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for windows...\033[0m"
	$(MAKE) katenary GOOS=windows GOARCH=amd64 OUT=$@

dist/katenary-darwin-amd64:
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for darwin...\033[0m"
	$(MAKE) katenary GOOS=darwin GOARCH=amd64 OUT=$@

dist/katenary-freebsd-amd64:
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for freebsd...\033[0m"
	$(MAKE) katenary GOOS=freebsd GOARCH=amd64 OUT=$@

dist/katenary-freebsd-arm64:
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for freebsd-arm64...\033[0m"
	$(MAKE) katenary GOOS=freebsd GOARCH=arm64 OUT=$@
	
gpg-sign:
	rm -f dist/*.asc
	$(MAKE) $(ASC_BINARIES)

dist/%.asc: dist/%
	gpg --armor --detach-sign  --default-key $(SIGNER) $< &>/dev/null || exit 1

install: build
	install -Dm755 katenary $(PREFIX)/bin/katenary

uninstall:
	rm -f $(PREFIX)/bin/katenary

clean:
	rm -rf katenary dist/* release.id 


serve-doc: __label_doc
	@cd doc && \
		[ -d venv ] || python -m venv venv; \
		source venv/bin/activate && \
		echo "==> Installing requirements in the virtual env..."
		pip install -qq -r requirements.txt && \
		echo "==> Serving doc with mkdocs..." && \
		mkdocs serve

tests: test
test:
	@echo -e "\033[1;33mTesting katenary $(VERSION)...\033[0m"
	go test -coverprofile=cover.out ./...
	$(MAKE) cover

cover:
	go tool cover -func=cover.out | grep "total:"
	go tool cover -html=cover.out -o cover.html
	if [ "$(BROWSER)" = "xdg-open" ]; then
		xdg-open cover.html
	else
		$(BROWSER) -i --new-window cover.html
	fi

push-release: build-all
	@rm -f release.id
	# read personal access token from .git-credentials
	TOKEN=$(shell cat .credentials)
	# create a new release based on current tag and get the release id
	@curl -sSL -X POST \
		-H "Accept: application/vnd.github.v3+json" \
		-H "Authorization: token $$TOKEN" \
		-d "{\"tag_name\": \"$(VERSION)\", \"target_commitish\": \"\", \"name\": \"$(VERSION)\", \"draft\": true, \"prerelease\": true}" \
		https://api.github.com/repos/metal3d/katenary/releases | jq -r '.id' > release.id
	@echo "Release id: $$(cat release.id) created"
	@echo "Uploading assets..."
	# push all dist binary as assets to the release
	@for i in $$(find dist -type f -name "katenary*"); do
		curl -sSL -H "Authorization: token $$TOKEN" \
			-H "Accept: application/vnd.github.v3+json" \
			-H "Content-Type: application/octet-stream" \
			--data-binary @$$i \
			https://uploads.github.com/repos/metal3d/katenary/releases/$$(cat release.id)/assets?name=$$(basename $$i)
	done
	@rm -f release.id


__label_doc:
	@command -v gomarkdoc || (echo "==> We need to install gomarkdoc..." && \
		go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest)
	@echo "=> Generating labels doc..."
	# short label doc
	go run ./cmd/katenary help-labels -m | \
		sed -i '
			/START_LABEL_DOC/,/STOP_LABEL_DOC/{/<!--/!d};
			/START_LABEL_DOC/,/STOP_LABEL_DOC/r/dev/stdin
		' doc/docs/labels.md
	# detailed label doc
	go run ./cmd/katenary help-labels -am | sed 's/^##/###/' | \
		sed -i '
			/START_DETAILED_DOC/,/STOP_DETAILED_DOC/{/<!--/!d}; 
			/START_DETAILED_DOC/,/STOP_DETAILED_DOC/r/dev/stdin
		' doc/docs/labels.md
	
	echo "=> Generating Code documentation..."
	PACKAGES=$$(for f in $$(find . -name "*.go" -type f); do dirname $$f; done | sort -u)
	for pack in $$PACKAGES; do
		echo "-> Generating doc for $$pack"
		gomarkdoc --repository.default-branch $(shell git branch --show-current) -o doc/docs/packages/$$pack.md $$pack
		sed -i  '/^## Index/,/^##/ { /## Index/d; /^##/! d }' doc/docs/packages/$$pack.md
	done
