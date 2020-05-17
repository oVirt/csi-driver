# oVirt CSI driver

Implementation of a CSI driver for oVirt.

This work is a continuation of the work done in github.com/ovirt/ovirt-openshift-extensions, 
and is the future of the development of a storage driver for oVirt.

This repo also contains an operator to deploy the driver on OpenShift or Kubernetes 
with most of the code based on openshift/csi-operator.

# Prerequisites
Before installation, please ensure that you are running a k8s cluster which supports the [CSI](https://kubernetes.io/docs/concepts/storage/#csi) implementation, such as [Openshift](https://www.okd.io/)

# Installation

Openshift/OKD 4:
  - Clone this repository
  - Ensure that you have oc installed locally
  - Run:
    ```
    export KUBECONFIG=</path/to/your>/auth/kubeconfig
    oc create -f deploy/csi-driver
    ```
  - validate:
    ```
    export KUBECONFIG=</path/to/your>/auth/kubeconfig
    oc get pods -n ovirt-csi-driver
    oc new-project zzz-test
    oc create -f deploy/csi-driver/example
    oc get pods -n zzz-test
    ```

Examples for StorageClass, PVC and Pod:
StorageClass:
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ovirt-csi-sc
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: csi.ovirt.org
parameters:
  # the name of the oVirt storage domain. "nfs" is just an example.
  storageDomainName: "nfs"
  thinProvisioning: "true"
```

PVC:
```yaml
kind: PersistentVolumeClaim
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
```

Pod:
```yaml
apiVersion: v1 
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
```

Kubernetes:
  - tbc
  
# Development

# Deploy
  - operator
  - dev/test

# OpenShift vs Kubernetes
- Credential requests require the openshift cloud credentials operator in order to provision. You will need to either deploy the operator and create the ovirt-credentials secret in the kube-system namespace, or provision the ovirt-credentials secret yourself into the ovirt-csi-driver namespace.