package assets

import (
	"fmt"
	"strings"
)

var _deploy_csi_driver_000_csi_driver_yaml = []byte(`apiVersion: storage.k8s.io/v1beta1
kind: CSIDriver
metadata:
  name: csi.ovirt.org
spec:
  attachRequired: true
  podInfoOnMount: true`)

func deploy_csi_driver_000_csi_driver_yaml() ([]byte, error) {
	return _deploy_csi_driver_000_csi_driver_yaml, nil
}

var _deploy_csi_driver_010_storageclass_yaml = []byte(`apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ovirt-csi-sc
provisioner: csi.ovirt.org
annotations:
  storageclass.kubernetes.io/is-default-class: "true"
parameters:
  # the name of the storage domain. "nfs" is just an example.
  storageDomainName: "nfs"
  thinProvisioning: "true"
`)

func deploy_csi_driver_010_storageclass_yaml() ([]byte, error) {
	return _deploy_csi_driver_010_storageclass_yaml, nil
}

var _deploy_csi_driver_020_autorization_yaml = []byte(`apiVersion: v1
kind: Namespace
metadata:
  name: ovirt-csi-driver
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ovirt-csi-node-sa
  namespace: ovirt-csi-driver
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ovirt-csi-controller-sa
  namespace: ovirt-csi-driver
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ovirt-csi-controller-provisioner-binding
subjects:
  - kind: ServiceAccount
    name: ovirt-csi-controller-sa
    namespace: ovirt-csi-driver
roleRef:
  kind: ClusterRole
  name: csi-external-provisioner
  apiGroup: rbac.authorization.k8s.io
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ovirt-csi-controller-attacher-binding
subjects:
  - kind: ServiceAccount
    name: ovirt-csi-controller-sa
    namespace: ovirt-csi-driver
roleRef:
  kind: ClusterRole
  name: csi-external-attacher
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ovirt-csi-controller-cr
rules:
  # Allow managing ember resources
  - apiGroups: ['ember-csi.io']
    resources: ['*']
    verbs: ['*']
  # Allow listing and creating CRDs
  - apiGroups: ['apiextensions.k8s.io']
    resources: ['customresourcedefinitions']
    verbs: ['list', 'create']
  - apiGroups: ['']
    resources: ['persistentvolumes']
    verbs: ['create', 'delete', 'get', 'list', 'watch', 'update', 'patch']
  - apiGroups: ['']
    resources: ['secrets']
    verbs: ['get', 'list']
  - apiGroups: ['']
    resources: ['persistentvolumeclaims']
    verbs: ['get', 'list', 'watch', 'update']
  - apiGroups: [""]
    resources: ["persistentvolumeclaims/status"]
    verbs: ["update", "patch"]
  - apiGroups: ['']
    resources: ['nodes']
    verbs: ['get', 'list', 'watch']
  - apiGroups: ['storage.k8s.io']
    resources: ['volumeattachments']
    verbs: ['get', 'list', 'watch', 'update', 'patch']
  - apiGroups: ['storage.k8s.io']
    resources: ['storageclasses']
    verbs: ['get', 'list', 'watch']
  - apiGroups: ['csi.storage.k8s.io']
    resources: ['csidrivers']
    verbs: ['get', 'list', 'watch', 'update', 'create']
  - apiGroups: ['']
    resources: ['events']
    verbs: ['list', 'watch', 'create', 'update', 'patch']
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotcontents"]
    verbs: ["create", "get", "list", "watch", "update", "delete"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshots"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshots/status"]
    verbs: ["update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["csinodes"]
    verbs: ["get", "list", "watch"]
---

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ovirt-csi-node-cr
rules:
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "update", "create", "delete"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: ["csi.storage.k8s.io"]
    resources: ["csinodeinfos"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["csinodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["list", "watch", "create", "update", "patch"]
  - apiGroups: ["security.openshift.io"]
    resources: ["securitycontextconstraints"]
    verbs: ["use"]
    resourceNames: ["privileged"]

---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: openshift:csi-driver-controller-leader-election
rules:
  - apiGroups: [""]
    resources: ["configmaps", "endpoints"]
    verbs: ["get", "list", "watch", "update", "create", "delete"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ovirt-csi-controller-binding
subjects:
  - kind: ServiceAccount
    name: ovirt-csi-controller-sa
    namespace: ovirt-csi-driver
roleRef:
  kind: ClusterRole
  name: ovirt-csi-controller-cr
  apiGroup: rbac.authorization.k8s.io
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ovirt-csi-leader-binding
subjects:
  - kind: ServiceAccount
    name: ovirt-csi-controller-sa
    namespace: ovirt-csi-driver
roleRef:
  kind: ClusterRole
  name: openshift:csi-driver-controller-leader-election
  apiGroup: rbac.authorization.k8s.io
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ovirt-csi-node-binding
subjects:
  - kind: ServiceAccount
    name: ovirt-csi-node-sa
    namespace: ovirt-csi-driver
roleRef:
  kind: ClusterRole
  name: ovirt-csi-node-cr
  apiGroup: rbac.authorization.k8s.io
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ovirt-csi-node-leader-binding
subjects:
  - kind: ServiceAccount
    name: ovirt-csi-node-sa
    namespace: ovirt-csi-driver
roleRef:
  kind: ClusterRole
  name: openshift:csi-driver-controller-leader-election
  apiGroup: rbac.authorization.k8s.io
---

`)

func deploy_csi_driver_020_autorization_yaml() ([]byte, error) {
	return _deploy_csi_driver_020_autorization_yaml, nil
}

var _deploy_csi_driver_030_node_yaml = []byte(`#TODO: Force DaemonSet to not run on master.
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: ovirt-csi-node
  namespace: ovirt-csi-driver
spec:
  selector:
    matchLabels:
      app: ovirt-csi-driver
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: ovirt-csi-driver
    spec:
      serviceAccount: ovirt-csi-node-sa
      initContainers:
        - name: prepare-ovirt-config
          env:
            - name: OVIRT_URL
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_url
            - name: OVIRT_USERNAME
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_username
            - name: OVIRT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_password
            - name: OVIRT_CAFILE
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_cafile
            - name: OVIRT_INSECURE
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_insecure
          image: busybox
          command:
            - /bin/sh
            - -c
            - |
              #!/bin/sh
              cat << EOF > /tmp/config/ovirt-config.yaml
              ovirt_url: $OVIRT_URL
              ovirt_username: $OVIRT_USERNAME
              ovirt_password: $OVIRT_PASSWORD
              ovirt_cafile: $OVIRT_CAFILE
              ovirt_insecure: $OVIRT_INSECURE
              EOF
          volumeMounts:
            - name: config
              mountPath: /tmp/config

      containers:
        - name: csi-driver-registrar
          imagePullPolicy: Always
          image: quay.io/k8scsi/csi-node-driver-registrar:v1.2.0
          args:
            - "--v=5"
            - "--csi-address=/csi/csi.sock"
            - "--kubelet-registration-path=/var/lib/kubelet/plugins/ovirt.org/csi.sock"
          env:
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: registration-dir
              mountPath: /registration
        - name: ovirt-csi-driver
          securityContext:
            privileged: true
            allowPrivilegeEscalation: true
          imagePullPolicy: Always
          image: quay.io/rgolangh/ovirt-csi-driver:latest
          args:
            - "--endpoint=unix:/csi/csi.sock"
            - "--namespace=ovirt-csi-driver"
            - "--node-name=$(KUBE_NODE_NAME)"
          env:
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: OVIRT_CONFIG
              value: /tmp/config/ovirt-config.yaml
          volumeMounts:
#            - name: kubelet-dir
#              mountPath: /var/lib/kubelet
#              mountPropagation: "Bidirectional"
            - name: socket-dir
              mountPath: /csi
            - name: config
              mountPath: /tmp/config/
            - name: plugin-dir
              mountPath: /var/lib/kubelet/plugins
              mountPropagation: Bidirectional
            - name: host-dev
              mountPath: /dev
            - name: mountpoint-dir
              mountPath: /var/lib/kubelet/pods
              mountPropagation: Bidirectional
      volumes:
        - name: registration-dir
          hostPath:
            path: /var/lib/kubelet/plugins_registry/
            type: Directory
        - name: kubelet-dir
          hostPath:
            path: /var/lib/kubelet
            type: Directory
        - name: plugin-dir
          hostPath:
            path: /var/lib/kubelet/plugins
            type: Directory
        - name: socket-dir
          hostPath:
            path: /var/lib/kubelet/plugins/ovirt.org/
            type: DirectoryOrCreate
        - name: host-dev
          hostPath:
            path: /dev
        - name: config
          emptyDir: {}
        - name: mountpoint-dir
          hostPath:
            path: /var/lib/kubelet/pods
            type: DirectoryOrCreate
`)

func deploy_csi_driver_030_node_yaml() ([]byte, error) {
	return _deploy_csi_driver_030_node_yaml, nil
}

var _deploy_csi_driver_040_controller_yaml = []byte(`kind: StatefulSet
apiVersion: apps/v1
metadata:
  name: ovirt-csi-plugin
  namespace: ovirt-csi-driver
spec:
  serviceName: "ovirt-csi-driver"
  replicas: 1
  selector:
    matchLabels:
      app: ovirt-csi-driver
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: ovirt-csi-driver
    spec:
      serviceAccount: ovirt-csi-controller-sa
      initContainers:
        - name: prepare-ovirt-config
          env:
            - name: OVIRT_URL
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_url
            - name: OVIRT_USERNAME
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_username
            - name: OVIRT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_password
            - name: OVIRT_CAFILE
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_cafile
            - name: OVIRT_INSECURE
              valueFrom:
                secretKeyRef:
                  name: ovirt-credentials
                  key: ovirt_insecure
          image: busybox
          command:
            - /bin/sh
            - -c
            - |
              #!/bin/sh
              cat << EOF > /tmp/config/ovirt-config.yaml
              ovirt_url: $OVIRT_URL
              ovirt_username: $OVIRT_USERNAME
              ovirt_password: $OVIRT_PASSWORD
              ovirt_cafile: $OVIRT_CAFILE
              ovirt_insecure: $OVIRT_INSECURE
              EOF
          volumeMounts:
            - name: config
              mountPath: /tmp/config

      containers:
        - name: csi-external-attacher
          imagePullPolicy: Always
          image: quay.io/k8scsi/csi-attacher:v2.0.0
          args:
            - "--v=4"
            - "--csi-address=/csi/csi.sock"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: csi-external-provisioner
          imagePullPolicy: Always
          image: quay.io/k8scsi/csi-provisioner:v1.5.0
          args:
            - "--v=9"
            - "--csi-address=/csi/csi.sock"
            - "--provisioner=csi.ovirt.org"
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
        - name: ovirt-csi-driver
          imagePullPolicy: Always
          image: quay.io/rgolangh/ovirt-csi-driver:latest
          args:
            - "--endpoint=unix:/csi/csi.sock"
            - "--namespace=ovirt-csi-driver"
            - "--ovirt-conf="
          env:
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: OVIRT_CONFIG
              value: /tmp/config/ovirt-config.yaml
          volumeMounts:
            - name: socket-dir
              mountPath: /csi
            - name: config
              mountPath: /tmp/config/
      volumes:
        - name: socket-dir
          emptyDir: {}
        - name: config
          emptyDir: {}
`)

func deploy_csi_driver_040_controller_yaml() ([]byte, error) {
	return _deploy_csi_driver_040_controller_yaml, nil
}

var _deploy_csi_driver_050_credential_request_yaml = []byte(`apiVersion: cloudcredential.openshift.io/v1
kind: CredentialsRequest
metadata:
  name: ovirt-csi-driver
  namespace: openshift-cloud-credential-operator
spec:
  providerSpec:
    apiVersion: cloudcredential.openshift.io/v1
    kind: OvirtProviderSpec
  secretRef:
    name: ovirt-credentials
    namespace: ovirt-csi-driver 
`)

func deploy_csi_driver_050_credential_request_yaml() ([]byte, error) {
	return _deploy_csi_driver_050_credential_request_yaml, nil
}

var _deploy_csi_driver_example_storage_claim_yaml = []byte(`kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: 1g-ovirt-cow-disk
  annotations:
    volume.beta.kubernetes.io/storage-class: ovirt-csi-sc
spec:
  storageClassName: ovirt-csi-sc
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
`)

func deploy_csi_driver_example_storage_claim_yaml() ([]byte, error) {
	return _deploy_csi_driver_example_storage_claim_yaml, nil
}

var _deploy_csi_driver_example_test_pod_yaml = []byte(`apiVersion: v1 
kind: Pod 
metadata:
  name: testpodwithcsi
spec:
  containers:
  - image: busybox
    name: testpodwithcsi
    command: ["sh", "-c", "while true; do ls -la /opt; echo this file system was made availble using ovirt flexdriver; sleep 1m; done"]
    imagePullPolicy: Always
    volumeMounts:
    - name: pv0002
      mountPath: "/opt"
  volumes:
  - name: pv0002
    persistentVolumeClaim:
      claimName: 1g-ovirt-cow-disk
`)

func deploy_csi_driver_example_test_pod_yaml() ([]byte, error) {
	return _deploy_csi_driver_example_test_pod_yaml, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() ([]byte, error){
	"deploy/csi-driver/000-csi-driver.yaml":         deploy_csi_driver_000_csi_driver_yaml,
	"deploy/csi-driver/010-storageclass.yaml":       deploy_csi_driver_010_storageclass_yaml,
	"deploy/csi-driver/020-autorization.yaml":       deploy_csi_driver_020_autorization_yaml,
	"deploy/csi-driver/030-node.yaml":               deploy_csi_driver_030_node_yaml,
	"deploy/csi-driver/040-controller.yaml":         deploy_csi_driver_040_controller_yaml,
	"deploy/csi-driver/050-credential-request.yaml": deploy_csi_driver_050_credential_request_yaml,
	"deploy/csi-driver/example/storage-claim.yaml":  deploy_csi_driver_example_storage_claim_yaml,
	"deploy/csi-driver/example/test-pod.yaml":       deploy_csi_driver_example_test_pod_yaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func     func() ([]byte, error)
	Children map[string]*_bintree_t
}

var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"deploy": {nil, map[string]*_bintree_t{
		"csi-driver": {nil, map[string]*_bintree_t{
			"000-csi-driver.yaml":         {deploy_csi_driver_000_csi_driver_yaml, map[string]*_bintree_t{}},
			"010-storageclass.yaml":       {deploy_csi_driver_010_storageclass_yaml, map[string]*_bintree_t{}},
			"020-autorization.yaml":       {deploy_csi_driver_020_autorization_yaml, map[string]*_bintree_t{}},
			"030-node.yaml":               {deploy_csi_driver_030_node_yaml, map[string]*_bintree_t{}},
			"040-controller.yaml":         {deploy_csi_driver_040_controller_yaml, map[string]*_bintree_t{}},
			"050-credential-request.yaml": {deploy_csi_driver_050_credential_request_yaml, map[string]*_bintree_t{}},
			"example": {nil, map[string]*_bintree_t{
				"storage-claim.yaml": {deploy_csi_driver_example_storage_claim_yaml, map[string]*_bintree_t{}},
				"test-pod.yaml":      {deploy_csi_driver_example_test_pod_yaml, map[string]*_bintree_t{}},
			}},
		}},
	}},
}}
