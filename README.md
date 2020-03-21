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