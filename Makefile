# kernel-style V=1 build verbosity
ifeq ("$(origin V)", "command line")
       BUILD_VERBOSE = $(V)
endif

ifeq ($(BUILD_VERBOSE),1)
       Q =
else
       Q = @
endif

#export CGO_ENABLED:=0

.PHONY: all
all: build

.PHONY: dep
dep:
	./hack/go-dep.sh

.PHONY: format
format:
	./hack/go-fmt.sh

.PHONY: go-generate
go-generate: dep
	$(Q)go generate ./...

.PHONY: sdk-generate
sdk-generate: dep
	operator-sdk generate k8s
	# operator-sdk generate openapi

.PHONY: vet
vet:
	./hack/go-vet.sh

.PHONY: test
test:
	./hack/go-test.sh

.PHONY: lint
lint:
	# Temporarily disabled
	# ./hack/go-lint.sh
	# ./hack/yaml-lint.sh

.PHONY: build
build:
	./hack/go-build.sh

#.PHONY: rhel
#rhel:
#	LOCAL=true ./hack/go-build.sh rhel

#.PHONY: rhel-scratch
#rhel-scratch:
#	./hack/go-build.sh rhel

#.PHONY: rhel-release
#rhel-release:
#	./hack/go-build.sh rhel release

.PHONY: clean
clean:
	rm -rf build/_output

.PHONY: deploy
deploy:
	- docker push quay.io/teiid/teiid-operator:0.2.0
	- oc create -f deploy/crds/virtualdatabase.crd.yaml
	- oc create -f deploy/service_account.yaml
	- oc create -f deploy/role.yaml
	- oc create -f deploy/role_binding.yaml
	- oc delete -f deploy/operator.yaml
	oc create -f deploy/operator.yaml
