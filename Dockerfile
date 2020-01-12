# A conventience dev/test Dockerfile.
FROM fedora:30
RUN dnf install -y e2fsprogs
COPY bin/ovirt-csi-driver .

ENTRYPOINT ["./ovirt-csi-driver"]
