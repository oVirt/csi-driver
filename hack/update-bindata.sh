#!/bin/sh

set -e

SRC_DIR="$(git rev-parse --show-toplevel)"

OUTPUT_FILE="${OUTPUT_FILE:-./pkg/assets/bindata.go}"

# ensure go-bindata
cd "${SRC_DIR}"
go build -o ./bin/go-bindata ./vendor/github.com/jteeuwen/go-bindata/go-bindata

# go-bindata generates code assets from the yaml we want to deploy by the operator.
"./bin/go-bindata" \
        -nocompress \
        -pkg "assets" \
        -o "${OUTPUT_FILE}" \
        -ignore "OWNERS" \
        -ignore ".*\.sw.?" \
	./deploy/csi-driver/... && \
gofmt -s -w "${OUTPUT_FILE}"
