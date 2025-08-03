## Linux / FreeBSD packages with fpm

DESCRIPTION := $(shell cat oci/description | sed ':a;N;$$!ba;s/\n/\\n/g')

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
