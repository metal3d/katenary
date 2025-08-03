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
