module github.com/ovirt/csi-driver

go 1.13

require (
	github.com/container-storage-interface/spec v1.2.0
	github.com/kubernetes-csi/csi-lib-utils v0.7.0
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/ovirt/go-ovirt v4.3.4+incompatible
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	google.golang.org/grpc v1.26.0
	gopkg.in/yaml.v2 v2.2.7
	k8s.io/apimachinery v0.17.1-beta.0
	k8s.io/client-go v0.17.0
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	sigs.k8s.io/controller-runtime v0.4.0
)
