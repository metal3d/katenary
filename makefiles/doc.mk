
## Documentation generation
serve-doc: doc
	@cd doc && \
		[ -d venv ] || python -m venv venv; \
		source venv/bin/activate && \
		echo "==> Installing requirements in the virtual env..."
		pip install -qq -r requirements.txt && \
		echo "==> Serving doc with mkdocs..." && \
		mkdocs serve


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
