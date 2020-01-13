# Image URL to use all building/pushing image targets
IMG ?= ovirt-csi-driver:latest

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
build: 
#	go build -o $(BINDIR)/ovirt-csi-operator -ldflags '-X main.version=$(REV) -extldflags "-static"' github.com/ovirt/csi-driver/cmd/ovirt-csi-operator
	go build -o $(BINDIR)/ovirt-csi-driver -ldflags '-X main.version=$(REV) -extldflags "-static"' github.com/ovirt/csi-driver/cmd/ovirt-csi-driver

.PHONY: verify
verify:
	hack/verify-gofmt.sh
	hack/verify-govet.sh

.PHONY: image
image:
	podman build . -f Dockerfile -t quay.io/rgolangh/${IMG}

#$(BINDATA):
#	go build -o $(BINDATA) ./vendor/github.com/jteeuwen/go-bindata/go-bindata
