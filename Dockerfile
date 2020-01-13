# A conventience dev/test Dockerfile.
FROM registry.svc.ci.openshift.org/openshift/release:golang-1.13 AS builder

WORKDIR /src/ovirt-csi-driver
COPY . .
RUN make build

FROM fedora:30

RUN dnf install -y e2fsprogs
COPY --from=builder src/ovirt-csi-driver/bin/ovirt-csi-driver .

ENTRYPOINT ["./ovirt-csi-driver"]
