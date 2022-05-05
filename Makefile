CUR_SHA=$(shell git log -n1 --pretty='%h')
CUR_BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --exact-match --tags $(CUR_SHA) 2>/dev/null || echo $(CUR_BRANCH)-$(CUR_SHA))
CTN:=$(shell which podman 2>&1 1>/dev/null && echo "podman" || echo "docker")
PREFIX=~/.local

GO=container
OUT=katenary
BLD_CMD=go build -ldflags="-X 'main.Version=$(VERSION)'" -o $(OUT) ./cmd/katenary/*.go
GOOS=linux
GOARCH=amd64

BUILD_IMAGE=docker.io/golang:1.18-alpine

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

build-all:
	rm -f dist/*
	$(MAKE) _build-all

_build-all: pull dist dist/katenary-linux-amd64 dist/katenary-linux-arm64 dist/katenary.exe dist/katenary-darwin-amd64 dist/katenary-freebsd-amd64 dist/katenary-freebsd-arm64

pull:
ifneq ($(GO),local)
	@echo -e "\033[1;32mPulling $(BUILD_IMAGE) docker image\033[0m"
	@$(CTN) pull $(BUILD_IMAGE)
endif

dist:
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

katenary: $(wildcard */*.go Makefile go.mod go.sum)
ifeq ($(GO),local)
	@echo "=> Build in host using go"
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
	


install: build
	cp katenary $(PREFIX)/bin/katenary

uninstall:
	rm -f $(PREFIX)/bin/katenary

clean:
	rm -rf katenary dist/* release.id 


tests: test
test:
	@echo -e "\033[1;33mTesting katenary $(VERSION)...\033[0m"
	go test -v ./...


.ONESHELL:
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
