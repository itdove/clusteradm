# Copyright Contributors to the Open Cluster Management project

BEFORE_SCRIPT := $(shell build/before-make.sh)

SCRIPTS_PATH ?= build

# Install software dependencies
INSTALL_DEPENDENCIES ?= ${SCRIPTS_PATH}/install-dependencies.sh

GOPATH := ${shell go env GOPATH}

export PROJECT_DIR            = $(shell 'pwd')
export PROJECT_NAME			  = $(shell basename ${PROJECT_DIR})

export GOPACKAGES   = $(shell go list ./... | grep -v /vendor | grep -v /build | grep -v /test )

.PHONY: clean
clean: clean-test
	kind delete cluster --name ${PROJECT_NAME}-functional-test
	
.PHONY: deps
deps:
	@$(INSTALL_DEPENDENCIES)

.PHONY: build
build: 
	go install ./cmd/clusteradm.go

.PHONY: install
install: build

.PHONY: plugin
plugin: build
	cp ${GOPATH}/bin/clusteradm ${GOPATH}/bin/oc-clusteradm
	cp ${GOPATH}/bin/clusteradm ${GOPATH}/bin/kubectl-clusteradm

.PHONY: check
## Runs a set of required checks
check: check-copyright

.PHONY: check-copyright
check-copyright:
	@build/check-copyright.sh

.PHONY: test
test:
	@build/run-unit-tests.sh

.PHONY: clean-test
clean-test: 
	-rm -r ./test/unit/coverage
	-rm -r ./test/unit/tmp
	-rm -r ./test/functional/tmp
	-rm -r ./test/out

.PHONY: functional-test-full
functional-test-full: deps install
	@build/run-functional-tests.sh
