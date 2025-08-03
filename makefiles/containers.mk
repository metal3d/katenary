
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
BUILD_IMAGE=docker.io/golang:$(GOVERSION)
	
GO_OCI:=$(CTN) run --rm -it \
	-v $(PWD):/go/src/katenary:z \
	-w /go/src/katenary \
	-e CGO_ENABLED=$(CGO_ENABLED) \
	-e GOOS=$(GOOS) \
	-e GOARCH=$(GOARCH) \
	$(CTN_USERMAP) \
	go-builder:$(GOVERSION)

packager-oci-image:
	@$(CTN) build -t packaging:fedora ./oci/packager 1>/dev/null

builder-oci-image:
	@$(CTN) build -t go-builder:$(GOVERSION) ./oci/builder \
		--build-arg GOVERSION=$(GOVERSION) 1>/dev/null
