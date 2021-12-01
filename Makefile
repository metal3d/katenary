VERSION=0.1.1
build: katenary

katenary: *.go generator/*.go compose/*.go helm/*.go
	podman run --rm -v $(PWD):/go/src/katenary -w /go/src/katenary --userns keep-id -it golang go build -o katenary  -ldflags="-X 'main.AppVersion=$(VERSION)'" . 


clean:
	rm -f katenary

