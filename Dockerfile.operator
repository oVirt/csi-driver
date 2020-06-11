FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/csi-operator \
    USER_UID=1001 \
    USER_NAME=csi-operator

# install operator binary
COPY bin/ovirt-csi-operator ${OPERATOR}
COPY manifests /manifests

USER ${USER_UID}
ENTRYPOINT ["/usr/local/bin/entrypoint"]

