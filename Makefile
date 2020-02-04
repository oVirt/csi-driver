# Image URL to use all building/pushing image targets
IMG ?= quay.io/rgolangh/ovirt-csi-driver:latest
IMG-OPERATOR ?= quay.io/rgolangh/ovirt-csi-operator:latest

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
build: build-driver build-operator

build-driver:
	go build -o $(BINDIR)/ovirt-csi-driver -ldflags '-X version.Version=$(REV) -extldflags "-static"' github.com/ovirt/csi-driver/cmd/ovirt-csi-driver
build-operator:
	go build -o $(BINDIR)/ovirt-csi-operator -ldflags '-X version.Version=$(REV) -extldflags "-static"' github.com/ovirt/csi-driver/cmd/manager


.PHONY: verify
verify:
	hack/verify-gofmt.sh
	hack/verify-govet.sh

.PHONY: image
image: image-driver image-operator

image-driver:
	podman build . -f Dockerfile -t ${IMG}
image-operator:
	podman build . -f Dockerfile.operator -t ${IMG-OPERATOR}

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify

$(BINDATA):
	go build -o $(BINDATA) ./vendor/github.com/jteeuwen/go-bindata/go-bindata
