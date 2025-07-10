# Strict mode
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.ONESHELL:
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
.PHONY: help dist-clean dist package build install test doc nsis

# Get a version string from git
CUR_SHA=$(shell git log -n1 --pretty='%h')
CUR_BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --exact-match --tags $(CUR_SHA) 2>/dev/null || echo $(CUR_BRANCH)-$(CUR_SHA))# use by golang flags

# Go build command and environment variables for target OS and architecture
GOVERSION=1.24
GO=container # container, local
OUTPUT=katenary
GOOS=linux
GOARCH=amd64
CGO_ENABLED=0
PREFIX=~/.local

# Get the container (podman is preferred, but docker is also supported)
# TODO: prpose nerdctl
CTN:=$(shell which podman 2>&1 1>/dev/null && echo "podman" || echo "docker")


# Packaging OCI image, to build rpm, deb, pacman, tar packages
PKG_OCI_IMAGE=packaging:fedora
PKG_OCI_OPTS:=--rm -it \
	-w $$PKG_OCI_WDIR \
	-v ./:/opt/katenary:z \
	--userns keep-id:uid=999,gid=999 \
	$(PKG_OCI_IMAGE)

# Set the version and package version, following build mode (default, release)
MODE=default
# If release mode
ifeq ($(MODE),release)
	PKG_VERSION:=$(VERSION)
	VERSION:=$(VERSION)
else
	PKG_VERSION=$(VERSION)# used for package name
endif

BLD_CMD=go build -ldflags="-X 'katenary/generator.Version=$(VERSION)'" -o $(OUTPUT)  ./cmd/katenary


# UPX compression
UPX_OPTS =
UPX ?= upx $(UPX_OPTS)

BUILD_IMAGE=docker.io/golang:$(GOVERSION)

# List of source files
SOURCES=$(shell find -name "*.go" -or -name "*.tpl" -type f | grep -v -P "^./example|^./vendor")
# List of binaries to build and sign
BINARIES=dist/katenary-linux-amd64 dist/katenary-linux-arm64 dist/katenary.exe dist/katenary-darwin-amd64 dist/katenary-freebsd-amd64 dist/katenary-freebsd-arm64
BINARIES += dist/katenary-windows-setup.exe

## GPG
# List of signatures to build
ASC_BINARIES=$(patsubst %,%.asc,$(BINARIES))
# GPG signer
SIGNER=metal3d@gmail.com

# Browser command to see coverage report after tests
BROWSER=$(shell command -v epiphany || echo xdg-open)

check-version:
	@echo "=> Checking version..."
	@echo "Mode: $(MODE)"
	@echo "Current version: $(VERSION)"
	@echo "Package version: $(PKG_VERSION)"
	@echo "Build command: $(BLD_CMD)"

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

# Simply build the binary for the current OS and architecture
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
dist: prepare $(BINARIES) upx gpg-sign check-sign packages

prepare: pull
	mkdir -p dist

dist/katenary-linux-amd64: $(SOURCES) Makefile go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for linux-amd64...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=amd64 OUTPUT=$@
	strip $@

dist/katenary-linux-arm64: $(SOURCES) Makefile go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for linux-arm...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=arm64 OUTPUT=$@

dist/katenary.exe: $(SOURCES) Makefile go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for windows...\033[0m"
	$(MAKE) katenary GOOS=windows GOARCH=amd64 OUTPUT=$@

dist/katenary-darwin-amd64: $(SOURCES) Makefile go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for darwin...\033[0m"
	$(MAKE) katenary GOOS=darwin GOARCH=amd64 OUTPUT=$@

dist/katenary-freebsd-amd64: $(SOURCES) Makefile go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for freebsd...\033[0m"
	$(MAKE) katenary GOOS=freebsd GOARCH=amd64 OUTPUT=$@
	strip $@

dist/katenary-freebsd-arm64: $(SOURCES) Makefile go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for freebsd-arm64...\033[0m"
	$(MAKE) katenary GOOS=freebsd GOARCH=arm64 OUTPUT=$@

dist/katenary-windows-setup.exe: nsis/EnVar.dll dist/katenary.exe packager-oci-image $(SOURCES) Makefile go.mod go.sum
	PKG_OCI_WDIR=/opt/katenary 
	podman run $(PKG_OCI_OPTS) \
		makensis -DAPP_VERSION=$(VERSION) nsis/katenary.nsi
	mv nsis/katenary-windows-setup.exe dist/katenary-windows-setup.exe

# Download the EnVar plugin for NSIS, put it in the nsis directory, and clean up
nsis/EnVar.dll:
	curl https://nsis.sourceforge.io/mediawiki/images/7/7f/EnVar_plugin.zip -o nsis/EnVar_plugin.zip
	cd nsis
	unzip -o EnVar_plugin.zip Plugins/x86-unicode/EnVar.dll
	mv Plugins/x86-unicode/EnVar.dll EnVar.dll
	rm -rf EnVar_plugin.zip Plugins

upx: dist/katenary-linux-amd64 dist/katenary-linux-arm64 dist/katenary-darwin-amd64
	$(UPX) dist/katenary-linux-amd64
	$(UPX) dist/katenary-linux-arm64
	#$(UPX) dist/katenary.exe
	$(UPX) dist/katenary-darwin-amd64 --force-macos

## Linux / FreeBSD packages

DESCRIPTION := $(shell cat packaging/description | sed ':a;N;$$!ba;s/\n/\\n/g')


FPM_OPTS=--name katenary \
	--version $(PKG_VERSION) \
	--url https://katenary.org \
	--vendor "Katenary Project" \
	--maintainer "Patrice Ferlet <metal3d@gmail.com>" \
	--license "MIT" \
	--description="$$(printf "$(DESCRIPTION)" | fold -s)"

FPM_COMMON_FILES=../doc/share/man/man1/katenary.1=/usr/local/share/man/man1/katenary.1 \
	../LICENSE=/usr/local/share/doc/katenary/LICENSE \
	../README.md=/usr/local/share/doc/katenary/README.md \

packages: manpage packager-oci-image
	@PKG_OCI_WDIR=/opt/katenary/dist
	for arch in amd64 arm64; do \
		for target in rpm deb pacman tar; do \
			echo "==> Building $$target package for arch $$arch..."; \
			podman run $(PKG_OCI_OPTS) fpm -s dir -t $$target -a $$arch -f $(FPM_OPTS) \
				$(FPM_COMMON_FILES) \
				./katenary-linux-$$arch=/usr/local/bin/katenary; \
		done
		mv dist/katenary.tar dist/katenary-linux-$(PKG_VERSION).$$arch.tar
		for target in freebsd tar; do \
			echo "==> Building $$target package for arch $$arch"; \
			podman run $(PKG_OCI_OPTS) fpm -s dir -t $$target -a $$arch -f $(FPM_OPTS) \
				$(FPM_COMMON_FILES) \
				./katenary-freebsd-$$arch=/usr/local/bin/katenary;
		done
		mv dist/katenary-$(PKG_VERSION).txz dist/katenary-$(PKG_VERSION).$$arch.txz
		mv dist/katenary.tar dist/katenary-freebsd-$(PKG_VERSION).$$arch.tar
	done

packager-oci-image:
	@podman build -t packaging:fedora ./packaging/oci 1>/dev/null

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

manpage:
	@echo "=> Generating manpage from documentation"
	@cd doc && \
		[ -d venv ] || python -m venv venv; \
		source venv/bin/activate && \
		echo "==> Installing requirements in the virtual env..." && \
		pip install -qq -r requirements.txt && \
		pip install -qq -r manpage_requirements.txt && \
		echo "==> Generating manpage..." && \
		MANPAGE=true mkdocs build && \
		rm -rf site &&
		echo "==> Manpage generated in doc/share/man/man1/katenary.1"

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
