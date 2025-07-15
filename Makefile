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

warn-docker:
	@echo -e "\033[1;31mWarning: Docker is not recommended, use Podman instead.\033[0m"
	sleep 5

# Get the container (Podman is preferred, but docker can be used too. It may failed with Docker.)
# TODO: propose nerdctl
CTN:=$(shell which podman 2>&1 1>/dev/null && echo "podman" || echo "docker")
ifeq ($(CTN),podman)
	CTN_USERMAP=--userns=keep-id
else
	$(MAKE) warn-docker
	CTN_USERMAP=--user=$(shell id -u):$(shell id -g) -e HOME=/tmp
endif


# Packaging OCI image, to build rpm, deb, pacman, tar packages
# We changes the keep-id uid/gid for Podman, so that the user inside the container is the same as the user outside.
# For Docker, as it doesn't support userns, we use common options, but it may fail...
PKG_OCI_IMAGE=packaging:fedora
ifeq ($(CTN),podman)
	# podman
	PKG_OCI_OPTS:=--rm -it \
		-v ./:/opt/katenary:z \
		--userns keep-id:uid=1001,gid=1001 \
		$(PKG_OCI_IMAGE)
else
	# docker
	PKG_OCI_OPTS:=--rm -it \
		-v ./:/opt/katenary:z \
		-e HOME=/tmp \
		$(CTN_USERMAP) \
		$(PKG_OCI_IMAGE)
endif
GO_BUILD=go build -ldflags="-X 'katenary/generator.Version=$(VERSION)'" -o $(OUTPUT)  ./cmd/katenary


# UPX compression
UPX_OPTS =
UPX ?= upx $(UPX_OPTS)

BUILD_IMAGE=docker.io/golang:$(GOVERSION)

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
	$$ make binaries
	EOF


## BUILD

# Simply build the binary for the current OS and architecture
build: pull katenary

pull:
ifneq ($(GO),local)
	@echo -e "\033[1;32mPulling $(BUILD_IMAGE) docker image\033[0m"
	@$(CTN) pull $(BUILD_IMAGE)
endif

katenary: $(SOURCES) go.mod go.sum
ifeq ($(GO),local)
	@echo "=> Build on host using go"
	$(GO_BUILD)
else
	@echo "=> Build in container using" $(CTN)
	@$(CTN) run \
		-e CGO_ENABLED=$(CGO_ENABLED) \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		--rm -v $(PWD):/go/src/katenary:z \
		-w /go/src/katenary \
		-v go-cache:/go/pkg/mod:z \
		$(CTN_USERMAP) \
		$(BUILD_IMAGE) $(GO_BUILD)
endif


# Make dist, build executables for all platforms, sign them, and compress them with upx if possible.
# Also generate the windows installer.
binaries: prepare $(BINARIES)
dist: binaries upx packages
dist-full: clean-dist dist gpg-sign check-sign rpm-sign check-dist-all

prepare: pull packager-oci-image
	mkdir -p dist

dist/katenary-linux-amd64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for linux-amd64...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=amd64 OUTPUT=$@
	strip $@

dist/katenary-linux-arm64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for linux-arm...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=arm64 OUTPUT=$@

dist/katenary.exe: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for windows...\033[0m"
	$(MAKE) katenary GOOS=windows GOARCH=amd64 OUTPUT=$@

dist/katenary-darwin-amd64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for darwin...\033[0m"
	$(MAKE) katenary GOOS=darwin GOARCH=amd64 OUTPUT=$@

dist/katenary-freebsd-amd64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for freebsd...\033[0m"
	$(MAKE) katenary GOOS=freebsd GOARCH=amd64 OUTPUT=$@
	strip $@

dist/katenary-freebsd-arm64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for freebsd-arm64...\033[0m"
	$(MAKE) katenary GOOS=freebsd GOARCH=arm64 OUTPUT=$@

dist/katenary-windows-setup.exe: nsis/EnVar.dll dist/katenary.exe
	@$(CTN) run -w /opt/katenary $(PKG_OCI_OPTS) \
		makensis -DAPP_VERSION=$(VERSION) nsis/katenary.nsi
	mv nsis/katenary-windows-setup.exe dist/katenary-windows-setup.exe

# Download the EnVar plugin for NSIS, put it in the nsis directory, and clean up
nsis/EnVar.dll:
	curl https://nsis.sourceforge.io/mediawiki/images/7/7f/EnVar_plugin.zip -o nsis/EnVar_plugin.zip
	cd nsis
	unzip -o EnVar_plugin.zip Plugins/x86-unicode/EnVar.dll
	mv Plugins/x86-unicode/EnVar.dll EnVar.dll
	rm -rf EnVar_plugin.zip Plugins

# UPX compression
upx: upx-linux upx-darwin

upx-linux: dist/katenary-linux-amd64 dist/katenary-linux-arm64 
	$(UPX) $^

upx-darwin: dist/katenary-darwin-amd64
	$(UPX) --force-macos $^

## Linux / FreeBSD packages with fpm

DESCRIPTION := $(shell cat packaging/description | sed ':a;N;$$!ba;s/\n/\\n/g')

FPM_OPTS=--name katenary \
	--url https://katenary.org \
	--vendor "Katenary Project" \
	--maintainer "Patrice Ferlet <metal3d@gmail.com>" \
	--license "MIT" \
	--description="$$(printf "$(DESCRIPTION)" | fold -s)"

# base files (doc...)
FPM_BASES=../LICENSE=/usr/local/share/doc/katenary/LICENSE \
	../README.md=/usr/local/share/doc/katenary/README.md

FPM_COMMON_FILES=$(FPM_BASES) ../doc/share/man/man1/katenary.1=/usr/local/share/man/man1/katenary.1

# ArchLinux has got inconsistent /usr/local/man directory
FPM_COMMON_FILES_ARCHLINUX=$(FPM_BASES) ../doc/share/man/man1/katenary.1=/usr/local/man/man1/katenary.1 \

# Pacman refuses dashes in version, and should start with a number
PACMAN_VERSION=$(shell echo $(VERSION) | sed 's/-/./g; s/^v//')

define RPM_MACROS
%_signature gpg
%_gpg_path /home/builder/.gnupg
%_gpg_name $(SIGNER)
%_gpgbin /usr/bin/gpg2
%__gpg_sign_cmd %{__gpg} gpg --force-v3-sigs --batch --verbose --no-armor --no-secmem-warning -u "%{_gpg_name}" -sbo %{__signature_filename} --digest-algo sha256 %{__plaintext_filename}'
endef

rpm: dist/katenary-linux-$(GOARCH)
	@echo "==> Building RPM packages for $(GOARCH)..."
	$(CTN) run -w /opt/katenary/dist $(PKG_OCI_OPTS) \
		fpm -s dir -t rpm -a $(GOARCH) -f $(FPM_OPTS) --version=$(VERSION) \
			$(FPM_COMMON_FILES) \
			./katenary-linux-$(GOARCH)=/usr/local/bin/katenary
rpm-sign:
	[ -f .rpmmacros ]  || echo "$(RPM_MACROS)" > .rpmmacros
	[ -f .secret.gpg ] || gpg --export-secret-keys -a $(SIGNER) > .secret.gpg
	$(CTN) run -w /opt/katenary/dist \
		-v ./.secret.gpg:/home/builder/signer.gpg \
		-v packager-gpg:/home/builder/.gnupg \
		$(PKG_OCI_OPTS) \
		gpg --import /home/builder/signer.gpg
	$(CTN) run -w /opt/katenary/dist \
		-v .rpmmacros:/home/builder/.rpmmacros:z \
		-v packager-gpg:/home/builder/.gnupg \
		$(PKG_OCI_OPTS) \
		bash -c 'for rpm in $$(find . -iname "*.rpm"); do echo signing: $$rpm; rpm --addsign $$rpm; done'

deb:
	@echo "==> Building DEB packages for $(GOARCH)..."
	$(CTN) run -w /opt/katenary/dist $(PKG_OCI_OPTS) \
		fpm -s dir -t deb -a $(GOARCH) -f $(FPM_OPTS) --version=$(VERSION) \
			$(FPM_COMMON_FILES) \
			./katenary-linux-$(GOARCH)=/usr/local/bin/katenary

pacman:
	@echo "==> Building Pacman packages for $(GOARCH)..."
	$(CTN) run -w /opt/katenary/dist $(PKG_OCI_OPTS) \
		fpm -s dir -t pacman -a $(GOARCH) -f $(FPM_OPTS) --version=$(PACMAN_VERSION) \
			$(FPM_COMMON_FILES_ARCHLINUX) \
			./katenary-linux-$(GOARCH)=/usr/local/bin/katenary

freebsd:
	@echo "==> Building FreeBSD packages for $(GOARCH)..."
	$(CTN) run -w /opt/katenary/dist $(PKG_OCI_OPTS) \
		fpm -s dir -t freebsd -a $(GOARCH) -f $(FPM_OPTS) --version=$(VERSION)\
			$(FPM_COMMON_FILES) \
			./katenary-freebsd-$(GOARCH)=/usr/local/bin/katenary
	mv dist/katenary-$(VERSION).txz dist/katenary-freebsd-$(VERSION).$(GOARCH).txz

tar:
	@echo "==> Building TAR packages for $(GOOS) $(GOARCH)..."
	$(CTN) run -w /opt/katenary/dist $(PKG_OCI_OPTS) \
		fpm -s dir -t tar -a $(GOARCH) -f $(FPM_OPTS) \
			$(FPM_COMMON_FILES) \
			./katenary-$(GOOS)-$(GOARCH)=/usr/local/bin/katenary
	mv dist/katenary.tar dist/katenary-$(GOOS)-$(VERSION).$(GOARCH).tar

packages: manpage packager-oci-image
	for arch in amd64 arm64; do \
		$(MAKE) rpm GOARCH=$$arch; \
		$(MAKE) deb GOARCH=$$arch; \
		$(MAKE) pacman GOARCH=$$arch; \
		$(MAKE) freebsd GOARCH=$$arch; \
		$(MAKE) tar GOARCH=$$arch GOOS=linux; \
		$(MAKE) tar GOARCH=$$arch GOOS=freebsd; \
	done

packager-oci-image:
	@$(CTN) build -t packaging:fedora ./packaging/oci 1>/dev/null

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
	@echo "=> checking in blank environment..."
	keyid=$(shell gpg -k --with-colons $(SIGNER)| grep '^pub' | cut -d: -f5);
	$(CTN) run --rm -it -e GPGKEY=$${keyid} -v ./dist:/opt/dist:z \
		packaging:fedora \
		bash -c '
		gpg --recv-key $$GPGKEY || exit 1;
		echo "Trusting $(SIGNER) key...";
		echo "trusted-key 483493B2DD0845DA8F21A26DF3702E3FAD8F76DC" >> ~/.gnupg/gpg.conf;
		gpg --update-trustdb;
		rm -f ~/.gnupg/gpg.conf;
		for f in /opt/dist/*.asc; do echo "==> $${f}"; gpg --verify $${f}; done;
		echo "=> Listing imported keys...";
		gpg -k
		'

dist/%.asc: dist/%
	gpg --armor --detach-sign  --default-key $(SIGNER) $< &>/dev/null || exit 1


check-dist-rocky:
	@echo "=> Checking Rocky Linux package..."
	p=$(wildcard dist/*x86_64.rpm);
	$(CTN) run --rm -it -v ./dist:/opt:z quay.io/rockylinux/rockylinux:latest bash -c "
		rpm -ivh /opt/$$(basename $$p);
		katenary version;
	"

check-dist-fedora:
	@echo "=> Checking Fedora package..."
	p=$(wildcard dist/*x86_64.rpm);
	$(CTN) run --rm -it -v ./dist:/opt:z quay.io/fedora/fedora:latest bash -c "
		rpm -ivh /opt/$$(basename $$p);
		katenary version;
	"

check-dist-archlinux:
	echo "=> Checking ArchLinux package..."
	p=$(wildcard dist/*x86_64.pkg.tar.zst);
	$(CTN) run --rm -it -v ./dist:/opt:z quay.io/archlinux/archlinux bash -c "
		pacman -U /opt/$$(basename $$p) --noconfirm;
		katenary version;
	"

check-dist-debian:
	@echo "=> Checking Debian package..."
	p=$(wildcard dist/*amd64.deb);
	$(CTN) run --rm -it -v ./dist:/opt:z debian:latest bash -c "
		dpkg -i /opt/$$(basename $$p);
		katenary version;
	"
check-dist-ubuntu:
	@echo "=> Checking Ubuntu package..."
	p=$(wildcard dist/*amd64.deb);
	$(CTN) run --rm -it -v ./dist:/opt:z ubuntu:latest bash -c "
		dpkg -i /opt/$$(basename $$p);
		katenary version;
	"

check-dist-all:
	$(MAKE) check-dist-fedora
	$(MAKE) check-dist-rocky
	$(MAKE) check-dist-debian
	$(MAKE) check-dist-ubuntu
	$(MAKE) check-dist-archlinux

## installation and uninstallation

install: build
	install -Dm755 katenary $(PREFIX)/bin/katenary

uninstall:
	rm -f $(PREFIX)/bin/katenary

serve-doc: doc
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
	@go tool cover -func=cover.out | grep "total:"
	go tool cover -html=cover.out -o cover.html

show-cover:
	@[ -f cover.html ] || (echo "cover.html is not present, run make test before"; exit 1)
	if [ "$(BROWSER)" = "xdg-open" ]; then
		xdg-open cover.html
	else
		$(BROWSER) -i --new-window cover.html
	fi

## Miscellaneous

clean-all: clean-dist clean-package-signer clean-go-cache

clean-dist:
	rm -rf dist
	rm -f katenary

clean-package-signer:
	rm -f .secret.gpg .rpmmacros

clean-go-cache:
	$(CTN) volume rm -f go-cache

