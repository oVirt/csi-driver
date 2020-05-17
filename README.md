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
  - Modify 010-storageclass.yaml 
    - change the metadata.name field to something descriptive. This is the storage class name you will consume within your cluster.
    - change the parameters.storageDomainName: "nfs" to the name of a valid storage domain in your ovirt cluster, that the provisioner user has access to manage.
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

    
Kubernetes:
  - tbc
  
# Development

# Deploy
  - operator
  - dev/test

# OpenShift vs Kubernetes
- Credential requests require the openshift cloud credentials operator in order to provision. You will need to either deploy the operator and create the ovirt-credentials secret in the kube-system namespace, or provision the ovirt-credentials secret yourself into the ovirt-csi-driver namespace.

# Troubleshooting

Get all the objects for the CSI driver
```
$ oc get all -n ovirt-csi-driver
NAME                       READY   STATUS            RESTARTS   AGE
pod/ovirt-csi-node-2nptq   0/2     PodInitializing   0          2d23h
pod/ovirt-csi-node-7t266   2/2     Running           0          15m
pod/ovirt-csi-plugin-0     0/3     PodInitializing   0          2d23h

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/ovirt-csi-node   1         1         1       1            1           <none>          2d23h

NAME                                READY   AGE
statefulset.apps/ovirt-csi-plugin   0/1     2d23h
```

`ovirt-csi-plugin` is a pod, that is part of the StatefulSet, running the controller's logic (create volume, delete volume, attach volume and more).
It runs the following containers: `csi-external-attacher` (triggers ControllerPublish/UnpublishVolume), `csi-external-provisioner` (mainly for Create/DeleteVolume) and `ovirt-csi-driver`.

`ovirt-csi-node` is a DaemonSet running the `csi-driver-registrar` (provides information about the driver with `GetPluginInfo` and `GetPluginCa
pabilities`) and `ovirt-csi-driver`.

The sidecar containers (`csi-external-attacher`, `csi-external-provisioner`, `csi-driver-registrar`) run alongside `ovirt-csi-driver` and run its code via gRPC.

Get inside the pod's containers:
```
oc -n ovirt-csi-driver rsh -c <pod name> pod/ovirt-csi-node-2nptq
```

Watch logs:
```
oc logs pods/ovirt-csi-node-2nptq -n ovirt-csi-driver -c <pod name> | less
```

