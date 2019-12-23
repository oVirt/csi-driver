# Image URL to use all building/pushing image targets
IMG ?= csi-operator:latest

CONTROLLER_MANIFESTS=pkg/generated/bindata.go
CONTROLLER_MANIFESTS_SRC=pkg/generated/manifests
E2E_MANIFESTS=test/e2e/bindata.go
E2E_MANIFESTS_SRC=test/e2e/manifests

BINDIR=bin
#BINDATA=$(BINDIR)/go-bindata
BINDATA=go-bindata

REV=$(shell git describe --long --tags --match='v*' --always --dirty)

all: build

# Run tests
.PHONY: test
test:
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build the binary
.PHONY: build
build: generate
	go build -o $(BINDIR)/ovirt-csi-operator -ldflags '-X main.version=$(REV) -extldflags "-static"' github.com/ovirt/csi-driver/cmd/ovirt-csi-operator
	go build -o $(BINDIR)/ovirt-csi-driver -ldflags '-X main.version=$(REV) -extldflags "-static"' github.com/ovirt/csi-driver/cmd/ovirt-csi-driver

.PHONY: generate
#generate: $(BINDATA)
generate:
	$(BINDATA) -pkg generated -nometadata -prefix $(CONTROLLER_MANIFESTS_SRC) -o $(CONTROLLER_MANIFESTS) $(CONTROLLER_MANIFESTS_SRC)/...
	gofmt -s -w $(CONTROLLER_MANIFESTS)
	$(BINDATA) -pkg e2e -nometadata -prefix $(E2E_MANIFESTS_SRC) -o $(E2E_MANIFESTS) $(E2E_MANIFESTS_SRC)/...
	gofmt -s -w $(E2E_MANIFESTS)

.PHONY: verify
verify:
	hack/verify-all.sh

.PHONY: test-e2e
# usage: KUBECONFIG=/var/run/kubernetes/admin.kubeconfig make test-e2e
test-e2e: generate
	hack/deploy-local.sh
	go test -v ./test/e2e/... -kubeconfig=$(KUBECONFIG)  -root $(PWD) -globalMan deploy/prerequisites/01_crd.yaml

.PHONY: container
# Build the docker image
container: test
	podman build . -t ${IMG}

#$(BINDATA):
#	go build -o $(BINDATA) ./vendor/github.com/jteeuwen/go-bindata/go-bindata
