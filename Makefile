# Strict mode
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.ONESHELL:
.DELETE_ON_ERROR:
.PHONY: all binaries build check-dist-all check-dist-archlinux check-dist-debian check-dist-fedora check-dist-rocky check-dist-ubuntu check-sign clean-all clean-dist clean-go-cache clean-package-signer cover deb dist dist-full doc freebsd gpg-sign help install install-gomarkdoc katenary manpage packager-oci-image packages pacman prepare pull rpm rpm-sign sast serve-doc show-cover tar test uninstall upx warn-docker
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

# Get a version string from git
CUR_SHA=$(shell git log -n1 --pretty='%h')
CUR_BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --exact-match --tags $(CUR_SHA) 2>/dev/null || echo $(CUR_BRANCH)-$(CUR_SHA))# use by golang flags

# Go build command and environment variables for target OS and architecture
GOVERSION=1.24
GO=container# container, local
OUTPUT=katenary
GOOS=linux
GOARCH=amd64
CGO_ENABLED=0
PREFIX=~/.local

# UPX compression
UPX_OPTS =
UPX ?= upx $(UPX_OPTS)

# List of source files
SOURCES=$(shell find -name "*.go" -or -name "*.tpl" -type f | grep -v -P "^./example|^./vendor")
# List of binaries to build and sign
BINARIES=\
	dist/katenary-linux-amd64\
	dist/katenary-linux-arm64\
	dist/katenary-darwin-amd64\
	dist/katenary-freebsd-amd64\
	dist/katenary-freebsd-arm64\
	dist/katenary.exe\
	dist/katenary-windows-setup.exe

## GPG
# List of signatures to build
ASC_BINARIES=$(patsubst %,%.asc,$(BINARIES))
# GPG signer
SIGNER=metal3d@gmail.com

# Browser command to see coverage report after tests
BROWSER=$(shell command -v epiphany || echo xdg-open)

include makefiles/build.mk
include makefiles/containers.mk
include makefiles/doc.mk
include makefiles/gpg.mk
include makefiles/packager.mk
include makefiles/test.mk

all: build

# if docker is used instead of podman, we warn the user
warn-docker:
	@echo -e "\033[1;31mWarning: Docker is not recommended, use Podman instead.\033[0m"
	sleep 5

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
	$$ make binaries
	EOF


## installation and uninstallation

install: build
	install -Dm755 katenary $(PREFIX)/bin/katenary

uninstall:
	rm -f $(PREFIX)/bin/katenary

## Miscellaneous

clean-all: clean-dist clean-package-signer clean-go-cache

clean-dist:
	rm -rf dist
	rm -f katenary

clean-package-signer:
	rm -f .secret.gpg .rpmmacros

clean-go-cache:
	$(CTN) volume rm -f go-cache

