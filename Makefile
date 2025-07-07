CUR_SHA=$(shell git log -n1 --pretty='%h')
CUR_BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --exact-match --tags $(CUR_SHA) 2>/dev/null || echo $(CUR_BRANCH)-$(CUR_SHA))

# get the container (podman is preferred, but docker is also supported)
# TODO: prpose nerdctl
CTN:=$(shell which podman 2>&1 1>/dev/null && echo "podman" || echo "docker")
PREFIX=~/.local

GOVERSION=1.24
GO=container
OUT=katenary

MODE=default
RELEASE=""
# if release mode
ifeq ($(MODE),release)
	VERSION:=release-$(VERSION)
endif

BLD_CMD=go build -ldflags="-X 'katenary/generator.Version=$(VERSION)'" -o $(OUT)  ./cmd/katenary
GOOS=linux
GOARCH=amd64
CGO_ENABLED=0

# GPG signer
SIGNER=metal3d@gmail.com

# upx compression
UPX_OPTS =
UPX ?= upx $(UPX_OPTS)

BUILD_IMAGE=docker.io/golang:$(GOVERSION)
# SHELL=/bin/bash

# List of source files
SOURCES=$(wildcard ./*.go ./*/*.go ./*/*/*.go)
# List of binaries to build and sign
BINARIES=dist/katenary-linux-amd64 dist/katenary-linux-arm64 dist/katenary.exe dist/katenary-darwin-amd64 dist/katenary-freebsd-amd64 dist/katenary-freebsd-arm64
BINARIES += dist/katenary-windows-setup.exe
# installer
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
.PHONY: help dist-clean build install tests test doc nsis

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


## BUILD

build: pull katenary

pull:
ifneq ($(GO),local)
	@echo -e "\033[1;32mPulling $(BUILD_IMAGE) docker image\033[0m"
	@$(CTN) pull $(BUILD_IMAGE)
endif

katenary: $(SOURCES) Makefile go.mod go.sum
ifeq ($(GO),local)
	@echo "=> Build on host using go"
	$(BLD_CMD)
else ifeq ($(CTN),podman)
	@echo "=> Build in container using" $(CTN)
	@podman run -e CGO_ENABLED=$(CGO_ENABLED) -e GOOS=$(GOOS) -e GOARCH=$(GOARCH) \
		--rm -v $(PWD):/go/src/katenary:z -w /go/src/katenary --userns keep-id  $(BUILD_IMAGE) $(BLD_CMD)
else
	@echo "=> Build in container using" $(CTN)
	@docker run -e CGO_ENABLED=$(CGO_ENABLED) -e GOOS=$(GOOS) -e GOARCH=$(GOARCH) \
		--rm -v $(PWD):/go/src/katenary:z -w /go/src/katenary --user $(shell id -u):$(shell id -g) -e HOME=/tmp  $(BUILD_IMAGE) $(BLD_CMD)
endif


# Make dist, build executables for all platforms, sign them, and compress them with upx if possible.
# Also generate the windows installer.
dist: prepare $(BINARIES) upx gpg-sign check-sign

prepare: pull
	mkdir -p dist

dist/katenary-linux-amd64:
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for linux-amd64...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=amd64 OUT=$@
	strip $@

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
	strip $@

dist/katenary-freebsd-arm64:
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for freebsd-arm64...\033[0m"
	$(MAKE) katenary GOOS=freebsd GOARCH=arm64 OUT=$@

dist/katenary-windows-setup.exe: nsis/EnVar.dll dist/katenary.exe
	makensis -DAPP_VERSION=$(VERSION) nsis/katenary.nsi
	mv nsis/katenary-windows-setup.exe dist/katenary-windows-setup.exe

nsis/EnVar.dll:
	curl https://nsis.sourceforge.io/mediawiki/images/7/7f/EnVar_plugin.zip -o nsis/EnVar_plugin.zip
	cd nsis
	unzip -o EnVar_plugin.zip Plugins/x86-unicode/EnVar.dll
	mv Plugins/x86-unicode/EnVar.dll EnVar.dll
	rm -rf EnVar_plugin.zip Plugins

upx:
	$(UPX) dist/katenary-linux-amd64
	$(UPX) dist/katenary-linux-arm64
	#$(UPX) dist/katenary.exe
	$(UPX) dist/katenary-darwin-amd64 --force-macos

## GPG signing

gpg-sign:
	rm -f dist/*.asc
	$(MAKE) $(ASC_BINARIES)

check-sign:
	@echo "=> Checking signatures..."
	@for f in $(ASC_BINARIES); do \
		if gpg --verify $$f &>/dev/null; then \
			echo "Signature for $$f is valid"; \
		else \
			echo "Signature for $$f is invalid"; \
			exit 1; \
		fi; \
	done

dist/%.asc: dist/%
	gpg --armor --detach-sign  --default-key $(SIGNER) $< &>/dev/null || exit 1


## installation and uninstallation

install: build
	install -Dm755 katenary $(PREFIX)/bin/katenary

uninstall:
	rm -f $(PREFIX)/bin/katenary


serve-doc: __label_doc
	@cd doc && \
		[ -d venv ] || python -m venv venv; \
		source venv/bin/activate && \
		echo "==> Installing requirements in the virtual env..."
		pip install -qq -r requirements.txt && \
		echo "==> Serving doc with mkdocs..." && \
		mkdocs serve

## Documentation generation

doc:
	@echo "=> Generating documentation..."
	# generate the labels doc and code doc
	$(MAKE) __label_doc

install-gomarkdoc:
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

__label_doc:
	@command -v gomarkdoc || (echo "==> We need to install gomarkdoc..." && \
		$(MAKE) install-gomarkdoc)
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


## TESTS, security analysis, and code quality

# Scan the source code. 
# - we don't need detection of text/template as it's not a web application, and
# - we don't need sha1 detection as it is not used for cryptographic purposes.
# Note: metrics are actually not sent to anyone - it's a thing that is removed from the code in the future.
sast:
	opengrep \
		--config auto \
		--exclude-rule go.lang.security.audit.xss.import-text-template.import-text-template \
		--exclude-rule go.lang.security.audit.crypto.use_of_weak_crypto.use-of-sha1  \
		--metrics=on  \
		.

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


## Miscellaneous

dist-clean:
	rm -rf dist
	rm -f katenary
