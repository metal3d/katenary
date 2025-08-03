
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
test: builder-oci-image
	@echo -e "\033[1;33mTesting katenary $(VERSION)...\033[0m"
	$(GO_OCI) go test -v -coverprofile=cover.out ./... || exit 1
	$(MAKE) cover

cover: builder-oci-image
	$(GO_OCI) \
		go tool cover -func=cover.out | grep "total:"
	$(GO_OCI) \
		go tool cover -html=cover.out -o cover.html

show-cover:
	@[ -f cover.html ] || (echo "cover.html is not present, run make test before"; exit 1)
	if [ "$(BROWSER)" = "xdg-open" ]; then
		xdg-open cover.html
	else
		$(BROWSER) -i --new-window cover.html
	fi
