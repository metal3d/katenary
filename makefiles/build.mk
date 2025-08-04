## BUILD
GO_BUILD=go build -ldflags="-X 'github.com/katenary/katenary/internal/generator.Version=$(VERSION)'" -trimpath -o $(OUTPUT) ./cmd/katenary

# Simply build the binary for the current OS and architecture
build: pull katenary

pull:
ifneq ($(GO),local)
	@echo -e "\033[1;32mPulling $(BUILD_IMAGE) docker image\033[0m"
	@$(CTN) pull $(BUILD_IMAGE)
endif

katenary: $(SOURCES) go.mod go.sum builder-oci-image
ifeq ($(GO),local)
	@echo "=> Build on host using go"
	$(GO_BUILD)
else
	echo "=> Build in container using" $(CTN)
	$(GO_OCI) $(GO_BUILD)
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
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for linux amd64...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=amd64 OUTPUT=$@
	strip $@

dist/katenary-linux-arm64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for linux arm64...\033[0m"
	$(MAKE) katenary GOOS=linux GOARCH=arm64 OUTPUT=$@

dist/katenary.exe: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for windows...\033[0m"
	$(MAKE) katenary GOOS=windows GOARCH=amd64 OUTPUT=$@

dist/katenary-darwin-amd64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for darwin amd64...\033[0m"
	$(MAKE) katenary GOOS=darwin GOARCH=amd64 OUTPUT=$@

dist/katenary-darwin-arm64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for darwin arm64...\033[0m"
	$(MAKE) katenary GOOS=darwin GOARCH=arm64 OUTPUT=$@

dist/katenary-freebsd-amd64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for freebsd amd64...\033[0m"
	$(MAKE) katenary GOOS=freebsd GOARCH=amd64 OUTPUT=$@
	strip $@

dist/katenary-freebsd-arm64: $(SOURCES) go.mod go.sum
	@echo
	@echo -e "\033[1;32mBuilding katenary $(VERSION) for freebsd arm64...\033[0m"
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
