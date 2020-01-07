# A conventience dev/test Dockerfile.
FROM registry.svc.ci.openshift.org/origin/4.1:base

COPY bin/ovirt-csi-driver .

ENTRYPOINT ["./ovirt-csi-driver"]
