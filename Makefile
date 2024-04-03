# Makefile is self-documenting, comments starting with '##' are extracted as help text.
help: ## Display this help.
	@echo; echo = Targets =
	@grep -E '^[A-Za-z0-9_-]+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -E '^## [A-Z0-9_]+: ' Makefile | sed 's/^## \([A-Z0-9_]*\): \(.*\)/\1#\2/' | column -s'#' -t

include .bingo/Variables.mk	# Versioned tools
include Variables.mk
IMG=$(KORREL8R_IMG)
IMAGE=$(KORREL8R_IMAGE)

## OVERLAY: Name of kustomize directory for `make deploy`.
OVERLAY?=config/overlays/dev
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(or $(shell podman info > /dev/null 2>&1 && which podman), $(shell docker info > /dev/null 2>&1 && which docker))

check: generate lint test ## Lint and test code.

all: check install _site image-build operator ## Build and test everything locally. Recommended before pushing.
	$(MAKE) -C operator all

clean: ## Remove generated files, including checked-in files.
	rm -vrf bin _site $(GENERATED) $(shell find -name 'zz_*')
	$(MAKE) -C operator clean

VERSION_TXT=cmd/korrel8r/version.txt

ifneq ($(VERSION),$(file <$(VERSION_TXT)))
.PHONY: $(VERSION_TXT) # Force update if VERSION_TXT does not match $(VERSION)
endif
$(VERSION_TXT):
	echo $(VERSION) > $@

# List of generated files
GENERATED_DOC=doc/zz_domains.adoc doc/zz_rest_api.adoc doc/zz_crd-ref.adoc
GENERATED=$(VERSION_TXT) pkg/config/zz_generated.deepcopy.go pkg/rest/zz_docs $(GENERATED_DOC) .copyright

generate: $(GENERATED) go.mod ## Generate code and doc.

GO_SRC=$(shell find -name '*.go')

.copyright: $(GO_SRC)
	hack/copyright.sh	# Make sure files have copyright notice.
	@touch $@

go.mod: $(GO_SRC)
	go mod tidy		# Keep modules up to date.
	@touch $@

pkg/config/zz_generated.deepcopy.go:  $(filter-out pkg/config/zz_generated.deepcopy.go,$(wildcard pkg/config/*.go)) $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object paths=./pkg/config/...

pkg/rest/zz_docs: $(wildcard pkg/rest/*.go pkg/korrel8r/*.go) $(SWAG)
	@mkdir -p $(dir $@)
	$(SWAG) init -q -g pkg/rest/api.go -o $@
	$(SWAG) fmt pkg/rest
	@touch $@

lint: $(VERSION_TXT) $(GOLANGCI_LINT) ## Run the linter to find and fix code style problems.
	$(GOLANGCI_LINT) run --fix

install: $(VERSION_TXT) ## Build and install the korrel8r binary in $GOBIN.
	go install -tags netgo ./cmd/korrel8r

test: ## Run all tests, requires a cluster.
	$(MAKE) TEST_NO_SKIP=1 test-skip
test-skip: $(VERSION_TXT) ## Run all tests but skip those requiring a cluster if not logged in.
	go test -timeout=1m -race ./...

cover: ## Run tests and show code coverage in browser.
	go test -coverprofile=test.cov ./...
	go tool cover --html test.cov; sleep 2 # Sleep required to let browser start up.

CONFIG=etc/korrel8r/korrel8r.yaml
run: $(GENERATED) ## Run `korrel8r web` using configuration in ./etc/korrel8r
	go run ./cmd/korrel8r web -c $(CONFIG) $(ARGS)

image-build: $(VERSION_TXT) ## Build image locally, don't push.
	$(IMGTOOL) build --tag=$(IMAGE) -f Containerfile .

image: image-build ## Build and push image. IMG must be set to a writable image repository.
	$(IMGTOOL) push -q $(IMAGE)

images: image
	$(MAKE) -C operator images

image-name: ## Print the full image name and tag.
	@echo $(IMAGE)

$(OVERLAY): $(OVERLAY)/kustomization.yaml
	mkdir -p  $@
	hack/replace-image.sh "quay.io/korrel8r/korrel8r" $(IMG) $(VERSION) > $<

WATCH=kubectl get events -A --watch-only& trap "kill %%" EXIT;
HAS_ROUTE={ oc api-versions | grep -q route.openshift.io/v1 ; }

# NOTE: deploy does not depend on 'image', since it may be used to deploy pre-existing images.
# To build and deploy a new image do `make image deploy`
deploy: $(IMAGE_KUSTOMIZATION)	## Deploy to current cluster using kustomize.
	$(WATCH) kubectl apply -k $(OVERLAY)
	$(HAS_ROUTE) && kubectl apply -k config/base/route
	$(WATCH) kubectl wait -n korrel8r --for=condition=available --timeout=60s deployment.apps/korrel8r

undeploy: $(OVERLAY)
	kubectl delete -k $(OVERLAY)

# Run asciidoctor from an image.
ADOC_RUN=$(IMGTOOL) run -iq -v./doc:/doc:z -v./_site:/_site:z quay.io/rhdevdocs/devspaces-documentation
# From github.com:darshandsoni/asciidoctor-skins.git
CSS?=adoc-readthedocs.css
ADOC_ARGS=-a revnumber=$(VERSION) -a allow-uri-read -a stylesdir=css -a stylesheet=$(CSS) -D/_site /doc/index.adoc

# _site is published to github pages by .github/workflows/asciidoctor-ghpages.yml.
_site: $(shell find doc) $(GENERATED_DOC) ## Generate the website HTML.
	@mkdir -p $@/etc
	@cp -r doc/images $@
	$(ADOC_RUN) asciidoctor $(ADOC_ARGS)
	$(ADOC_RUN) asciidoctor-pdf $(ADOC_ARGS) -o ebook.pdf
	$(and $(shell type -p linkchecker),linkchecker --check-extern --ignore-url 'https?://localhost[:/].*' _site)
	@touch $@

.PHONY: doc
doc: $(GENERATED_DOC)

doc/zz_domains.adoc: $(shell find cmd/korrel8r-doc internal pkg -name '*.go')
	go run ./cmd/korrel8r-doc pkg/domains/* > $@

doc/zz_rest_api.adoc: pkg/rest/zz_docs $(shell find etc/swagger) $(SWAGGER)
	$(SWAGGER) -q generate markdown -T etc/swagger -f $</swagger.json --output $@

doc/zz_crd-ref.adoc: $(shell find etc/crd-ref-docs operator/apis pkg/config) $(CRD_REF_DOCS)
	$(CRD_REF_DOCS) --source-path=operator/apis --config=etc/crd-ref-docs/config.yaml --templates-dir=etc/crd-ref-docs/templates --output-path=$@

release: release-commit release-push ## Create and push a new release tag and image. Set VERSION=vX.Y.Z.

release-check:
	@echo "$(VERSION)" | grep -qE "^[0-9]+\.[0-9]+\.[0-9]+$$" || { echo "VERSION=$(VERSION) must be semantic version X.Y.Z"; exit 1; }
	$(MAKE) all
	@test -z "$(shell git status --porcelain)" || { git status -s; echo Workspace is not clean; exit 1; }

release-commit: release-check
	hack/changelog.sh $(VERSION) > CHANGELOG.md	# Update change log
	git commit -q  -m "Release $(VERSION)" -- $(VERSION_TXT) CHANGELOG.md
	git tag $(VERSION) -a -m "Release $(VERSION)"

release-push: release-check image
	git push origin main --follow-tags
	$(IMGTOOL) push -q "$(IMAGE)" "$(IMG):latest"

tools: $(BINGO) ## Download all tools needed for development
	$(BINGO) get
