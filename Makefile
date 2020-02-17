#REGISTRY=quay.io/teiid
REGISTRY?=`whoami`
IMAGE=teiid-operator
TAG=2.0.0-SNAPSHOT

IMAGE_NAME=$(REGISTRY)/$(IMAGE):$(TAG)

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
build: test
	go generate ./...
	operator-sdk build $(IMAGE_NAME)

.PHONY: clean
clean:
	rm -rf build/_output

.PHONY: lint
lint:
	scripts/go-lint.sh

images-push:
	docker push $(IMAGE_NAME)

.PHONY: deploy
deploy: images-push
	./scripts/manualDeploy.sh $(IMAGE_NAME)
