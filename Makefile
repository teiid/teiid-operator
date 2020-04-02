#REGISTRY=quay.io/teiid
REGISTRY?=`whoami`
IMAGE=teiid-operator
TAG?=latest
CRC_REGISTRY=image-registry.openshift-image-registry.svc:5000

IMAGE_NAME=$(REGISTRY)/$(IMAGE):$(TAG)
CRC_IMAGE_NAME=$(CRC_REGISTRY)/`oc project --short`/$(IMAGE):$(TAG)
GO_FLAGS ?= GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on
SDK_VERSION=v0.15.1
GOPATH ?= "$(HOME)/go"
FMT_LOG=fmt.log

.PHONY: all
all: build

.PHONY: dep
dep:
	export GO111MODULE=on
	echo $(IMAGE_NAME)

.PHONY: vet
vet: dep sdk-generate
	go vet ./...

.PHONY: fmt
fmt:
	gofmt -s -l -w cmd/ pkg/ 

.PHONY: test
test: vet fmt
	GOCACHE=on 
	go test ./...

.PHONY: sdk-generate
sdk-generate: dep
	operator-sdk generate k8s

.PHONY: build
build: format test
	@echo Building...
	go generate ./...
	@${GO_FLAGS} operator-sdk build --image-builder buildah $(IMAGE_NAME)

.PHONY: clean
clean:
	rm -rf build/_output
	./scripts/clean.sh $(IMAGE_NAME)

.PHONY: lint
lint:
	scripts/go-lint.sh

images-push:
	buildah push $(IMAGE_NAME)

.PHONY: deploy
deploy: images-push
	./scripts/manualDeploy.sh $(IMAGE_NAME) $(CRC_IMAGE_NAME)

.PHONY: install-tools
install-tools:
	@${GO_FLAGS} go install \
		golang.org/x/lint/golint \
		github.com/securego/gosec/cmd/gosec \
		golang.org/x/tools/cmd/goimports \
		k8s.io/code-generator/cmd/client-gen \
		k8s.io/kube-openapi/cmd/openapi-gen

.PHONY: install-sdk
install-sdk:
	@echo Installing SDK ${SDK_VERSION}
	@SDK_VERSION=$(SDK_VERSION) GOPATH=$(GOPATH) ./scripts/install-sdk.sh

.PHONY: install
install: install-sdk install-tools

.PHONY: ci
ci: install ensure-generate-is-noop check format lint build test

.PHONY: generate
generate: internal-generate format

.PHONY: internal-generate
internal-generate:
	@GOPATH=${GOPATH} ./scripts/generate.sh

.PHONY: format
format:
	@echo Formatting code...
	@GOPATH=${GOPATH} ./scripts/format.sh

.PHONY: ensure-generate-is-noop
ensure-generate-is-noop: generate format
	@git diff -s --exit-code pkg/apis/teiid/v1alpha1/zz_generated.*.go || (echo "Build failed: a model has been changed but the generated resources aren't up to date. Run 'make generate' and update your PR." && exit 1)

.PHONY: check
check:
	@echo Checking...
	@GOPATH=${GOPATH} ./scripts/format.sh > $(FMT_LOG)
	@[ ! -s "$(FMT_LOG)" ] || (echo "Go fmt, license check, or import ordering failures, run 'make format'" | cat - $(FMT_LOG) && false)
