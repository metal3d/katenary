CUR_SHA=$(shell git log -n1 --pretty='%h')
CUR_BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --exact-match --tags $(CUR_SHA) 2>/dev/null || echo $(CUR_BRANCH)-$(CUR_SHA))
CTN:=$(shell which podman 2>&1 1>/dev/null && echo "podman" || echo "docker")
PREFIX=~/.local

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
	EOF


build: katenary

katenary: *.go generator/*.go compose/*.go helm/*.go
	@echo Build using $(CTN)
ifeq ($(CTN),podman)
	@podman run --rm -v $(PWD):/go/src/katenary:z -w /go/src/katenary --userns keep-id -it golang go build -o katenary  -ldflags="-X 'main.Version=$(VERSION)'" . 
else
	@docker run --rm -v $(PWD):/go/src/katenary:z -w /go/src/katenary --user $(shell id -u):$(shell id -g) -e HOME=/tmp -it golang go build -o katenary  -ldflags="-X 'main.Version=$(VERSION)'" . 
endif


install: build
	cp katenary $(PREFIX)/bin/katenary

uninstall:
	rm -f $(PREFIX)/bin/katenary

clean:
	rm -f katenary


