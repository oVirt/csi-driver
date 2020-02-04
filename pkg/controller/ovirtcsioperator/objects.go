package ovirtcsioperator

import (
	"path"
	"regexp"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	csidriverv1alpha1 "github.com/ovirt/csi-driver/pkg/apis/ovirt/v1alpha1"

	"github.com/ovirt/csi-driver/pkg/apis/ovirt/v1alpha1"
)

const (
	daemonSetLabel  = "csidriver.storage.openshift.io/daemonset"
	deploymentLabel = "csidriver.storage.openshift.io/deployment"

	defaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

	// Port where livenessprobe listens
	livenessprobePort = 9808

	// How many probe failures lead to driver restart
	livenessprobeFailureThreshold = 3

	// Default timeout of the probe (in seconds)
	livenessprobeDefaultTimeout = int32(30)

	// Name of volume with CSI driver socket
	driverSocketVolume = "socket-dir"

	// Path where driverSocketVolume is mounted into all sidecar container
	socketDir = "/csi"

	// The socket file
	socketFile = "csi.sock"

	driverName = "csi.ovirt.org"

	configVolumeName = "config"

	configVolumePath = "/tmp/config"
	// Name of volume with /var/lib/kubelet
	kubeletRootVolumeName = "kubelet-root"

	namespace = "openshift-ovirt-infra"

	// OwnerLabelNamespace is name of label with namespace of owner CSIDriverDeployment.
	OwnerLabelNamespace = "csidriver.storage.openshift.io/owner-namespace"
	// OwnerLabelName is name of label with name of owner CSIDriverDeployment.
	OwnerLabelName = "csidriver.storage.openshift.io/owner-name"
)

var (
	replicas             = int32(1)
	reclaimPolicy        = v1.PersistentVolumeReclaimDelete
	allowVolumeExpansion = false
)

// generateServiceAccount prepares a ServiceAccount that will be used by all pods (controller + daemon set) with
// CSI drivers and its sidecar containers.
func (r *ReconcileOvirtCSIOperator) generateServiceAccount(name string, cr *v1alpha1.OvirtCSIOperator) *v1.ServiceAccount {
	sa := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	r.addOwnerLabels(&sa.ObjectMeta, cr)
	return &sa
}

func (r *ReconcileOvirtCSIOperator) generateClusterRoleController(cr *v1alpha1.OvirtCSIOperator) *rbacv1.ClusterRole {
	role := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ovirt-csi-controller-cr",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"list", "create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs: []string{"create",
					"delete",
					"get",
					"list",
					"watch",
					"update",
					"patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"list", "create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumeclaims/status"},
				Verbs:     []string{"watch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"volumeattachments"},
				Verbs:     []string{"get", "watch", "list", "update", "patch"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{"csi.storage.k8s.io"},
				Resources: []string{"csidrivers"},
				Verbs:     []string{"get", "watch", "list", "update", "create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "watch", "list", "update", "patch"},
			},
			{
				APIGroups: []string{"snapshot.storage.k8s.io"},
				Resources: []string{"volumesnapshotclasses"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{"snapshot.storage.k8s.io"},
				Resources: []string{"volumesnapshotcontents"},
				Verbs:     []string{"get", "watch", "list", "update", "create", "delete"},
			},
			{
				APIGroups: []string{"snapshot.storage.k8s.io"},
				Resources: []string{"volumesnapshots"},
				Verbs:     []string{"get", "watch", "list", "update"},
			},
			{
				APIGroups: []string{"snapshot.storage.k8s.io"},
				Resources: []string{"volumesnapshots/status"},
				Verbs:     []string{"update"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"csinodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	r.addOwnerLabels(&role.ObjectMeta, cr)
	return &role
}

func (r *ReconcileOvirtCSIOperator) generateClusterRoleNode(cr *v1alpha1.OvirtCSIOperator) *rbacv1.ClusterRole {
	role := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ovirt-csi-node-cr",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs:     []string{"create", "delete", "get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "watch", "list", "update", "patch"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"persistentvolumeclaims"},
				Verbs:     []string{"get", "watch", "list", "update"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{"csi.storage.k8s.io"},
				Resources: []string{"csinodes"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "watch", "list", "update", "patch"},
			},
			{
				APIGroups: []string{"csi.storage.k8s.io"},
				Resources: []string{"csinodeinfos"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{"csi.storage.k8s.io"},
				Resources: []string{"csinodes"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{"csi.storage.k8s.io"},
				Resources: []string{"volumeattachments"},
				Verbs:     []string{"get", "watch", "list", "update"},
			},
			{
				APIGroups:     []string{"security.openshift.io"},
				Resources:     []string{"securitycontextconstraints"},
				Verbs:         []string{"use"},
				ResourceNames: []string{"privileged"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"csinodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	r.addOwnerLabels(&role.ObjectMeta, cr)
	return &role
}

func (r *ReconcileOvirtCSIOperator) generateClusterRoleLeaderElection(cr *v1alpha1.OvirtCSIOperator) *rbacv1.ClusterRole {
	role := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openshift:csi-driver-controller-leader-election",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs:     []string{"create", "delete", "get", "list", "watch", "update"},
			},
		},
	}
	r.addOwnerLabels(&role.ObjectMeta, cr)
	return &role
}

// generateClusterRoleBinding prepares a ClusterRoleBinding that gives a ServiceAccount privileges needed by
// sidecar containers.
func (r *ReconcileOvirtCSIOperator) generateClusterRoleBinding(cr *v1alpha1.OvirtCSIOperator, name, serviceAccount, roleName string) *rbacv1.ClusterRoleBinding {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{

			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     roleName,
		},
	}
	r.addOwnerLabels(&crb.ObjectMeta, cr)
	return crb
}

// generateLeaderElectionRoleBinding prepares a RoleBinding that gives a ServiceAccount privileges needed by
// attacher and provisioner leader election.
func (r *ReconcileOvirtCSIOperator) generateLeaderElectionRoleBinding(cr *v1alpha1.OvirtCSIOperator, serviceAccount *v1.ServiceAccount) *rbacv1.RoleBinding {
	rbName := "leader-election-" + cr.Name
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      rbName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount.Name,
				Namespace: serviceAccount.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     r.config.LeaderElectionClusterRoleName,
		},
	}
	r.addOwner(&rb.ObjectMeta, cr)
	return rb
}

// generateDaemonSet prepares a DaemonSet with CSI driver and driver registrar sidecar containers.
func (r *ReconcileOvirtCSIOperator) generateDaemonSet(cr *v1alpha1.OvirtCSIOperator) *appsv1.DaemonSet {

	initContainers := []v1.Container{
		{
			Name: "prepare-ovirt-config",
			Env: []v1.EnvVar{
				{
					Name: "OVIRT_URL",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_url",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
				{
					Name: "OVIRT_USERNAME",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_username",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
				{
					Name: "OVIRT_PASSWORD",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_password",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
				{
					Name: "OVIRT_CAFILE",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_cafile",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
				{
					Name: "OVIRT_INSECURE",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_insecure",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
			},
			Image: "busybox",
			Command: []string{
				"/bin/sh",
				"-c",
				`#!/bin/sh
cat << EOF > /tmp/config/ovirt-config.yaml
ovirt_url: $OVIRT_URL
ovirt_username: $OVIRT_USERNAME
ovirt_password: $OVIRT_PASSWORD
ovirt_cafile: $OVIRT_CAFILE
ovirt_insecure: $OVIRT_INSECURE
EOF`,
			},
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "config",
					MountPath: "/tmp/config",
				},
			},
		},
	}

	// Add CSI Registrar sidecar
	registrar := v1.Container{
		Name:            "csi-driver-registrar",
		Image:           "quay.io/k8scsi/csi-node-driver-registrar:v1.2.0",
		ImagePullPolicy: v1.PullAlways,
		Args: []string{
			"--v=5",
			"--csi-address=/csi/csi.sock",
			"--kubelet-registration-path=/var/lib/kubelet/plugins/ovirt.org/csi.sock",
		},
		Env: []v1.EnvVar{
			{
				Name: "KUBE_NODE_NAME",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
		},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "socket-dir",
				MountPath: "/csi",
			},
			{
				Name:      "registration-dir",
				MountPath: "/registration",
			},
		},
	}

	mountPropogationType := v1.MountPropagationBidirectional
	csiDriver := v1.Container{
		Name:            "ovirt-csi-driver",
		Image:           "quay.io/rgolangh/ovirt-csi-driver:latest",
		ImagePullPolicy: v1.PullAlways,
		SecurityContext: &v1.SecurityContext{
			Privileged:               boolPtr(true),
			AllowPrivilegeEscalation: boolPtr(true),
		},
		Args: []string{
			"--endpoint=unix:/csi/csi.sock",
			"--namespace=" + cr.Namespace,
			"--node-name=$(KUBE_NODE_NAME)",
		},
		Env: []v1.EnvVar{
			{
				Name: "KUBE_NODE_NAME",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name:  "OVIRT_CONFIG",
				Value: "/tmp/config/ovirt-config.yaml",
			},
		},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "socket-dir",
				MountPath: "/csi",
			},
			{
				Name:      "config",
				MountPath: "/tmp/config",
			},
			{
				Name:             "plugin-dir",
				MountPath:        "/var/lib/kubelet/plugins",
				MountPropagation: &mountPropogationType,
			},
			{
				Name:      "host-dev",
				MountPath: "/dev",
			},
			{
				Name:             "mountpoint-dir",
				MountPath:        "/var/lib/kubelet/pods",
				MountPropagation: &mountPropogationType,
			},
		},
	}

	hostPathDirectory := v1.HostPathDirectory
	hostPathDirectoryOrCreate := v1.HostPathDirectoryOrCreate

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ovirt-csi-node",
			Namespace: cr.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "ovirt-csi-driver",
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "ovirt-csi-driver",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "ovirt-csi-node-sa",
					InitContainers:     initContainers,
					Containers: []v1.Container{
						registrar,
						csiDriver,
					},
					Volumes: []v1.Volume{
						{
							Name: "registration-dir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins_registry/",
									Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "kubelet-dir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/kubelet",
									Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "plugin-dir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins",
									Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "socket-dir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins/ovirt.org/",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "host-dev",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/dev",
								},
							},
						},
						{
							Name: "config",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "mountpoint-dir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/pods",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}
	r.addOwnerLabels(&ds.ObjectMeta, cr)
	return ds
}

// generateStatefulSet prepares a Deployment with CSI driver and attacher and provisioner sidecar containers.
func (r *ReconcileOvirtCSIOperator) generateStatefulSet(cr *v1alpha1.OvirtCSIOperator) *appsv1.StatefulSet {
	labels := map[string]string{
		"app": "ovirt-csi-driver",
	}

	var containers []v1.Container

	initContainers := []v1.Container{
		{
			Name: "prepare-ovirt-config",
			Env: []v1.EnvVar{
				{
					Name: "OVIRT_URL",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_url",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
				{
					Name: "OVIRT_USERNAME",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_username",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
				{
					Name: "OVIRT_PASSWORD",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_password",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
				{
					Name: "OVIRT_CAFILE",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_cafile",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
				{
					Name: "OVIRT_INSECURE",
					ValueFrom: &v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							Key: "ovirt_insecure",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "ovirt-credentials",
							},
						},
					},
				},
			},
			Image: "busybox",
			Command: []string{
				"/bin/sh",
				"-c",
				`#!/bin/sh
cat << EOF > /tmp/config/ovirt-config.yaml
ovirt_url: $OVIRT_URL
ovirt_username: $OVIRT_USERNAME
ovirt_password: $OVIRT_PASSWORD
ovirt_cafile: $OVIRT_CAFILE
ovirt_insecure: $OVIRT_INSECURE
EOF`,
			},
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      configVolumeName,
					MountPath: configVolumePath,
				},
			},
		},
	}

	sidecarSocketPath := path.Join(socketDir, socketFile)

	containers = append(containers, v1.Container{
		Name:  "csi-external-provisioner",
		Image: "quay.io/k8scsi/csi-provisioner:v1.5.0",
		Args: []string{
			"--v=5",
			"--csi-address=" + sidecarSocketPath,
			"--provisioner=" + driverName,
		},

		VolumeMounts: []v1.VolumeMount{
			{
				Name:      driverSocketVolume,
				MountPath: socketDir,
			},
		},
	})

	containers = append(containers, v1.Container{
		Name:  "csi-external-attacher",
		Image: "quay.io/k8scsi/csi-attacher:v2.0.0",
		Args: []string{
			"--v=5",
			"--csi-address=/csi/csi.sock",
		},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      driverSocketVolume,
				MountPath: socketDir,
			},
		},
	})

	containers = append(containers, v1.Container{
		Name:  "ovirt-csi-driver",
		Image: "quay.io/rgolangh/ovirt-csi-driver:latest",
		Args: []string{
			"--v=5",
			"--namespace=" + "openshift-ovirt-infra",
			"--endpoint=unix:" + sidecarSocketPath,
		},
		Env: []v1.EnvVar{
			{
				Name: "KUBE_NODE_NAME",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name:  "OVIRT_CONFIG",
				Value: configVolumePath + "/ovirt-config.yaml",
			},
		},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      driverSocketVolume,
				MountPath: socketDir,
			},
			{
				Name:      configVolumeName,
				MountPath: configVolumePath,
			},
		},
	})

	volumes := []v1.Volume{
		{
			Name: driverSocketVolume,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: configVolumeName,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      "ovirt-csi-controller",
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "ovirt-csi-driver",
					},
				},
				Spec: v1.PodSpec{
					InitContainers: initContainers,
					Containers:     containers,
					Volumes:        volumes,
				},
			},
			Replicas: &replicas,
		},
	}

	r.addOwnerLabels(&statefulSet.ObjectMeta, cr)
	return statefulSet
}

// generateStorageClass prepares a StorageClass from given template
func (r *ReconcileOvirtCSIOperator) generateStorageClass(cr *v1alpha1.OvirtCSIOperator) *storagev1.StorageClass {
	var expected = &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "ovirt-csi-sc",
		},
		// ObjectMeta will be filled below
		Provisioner:          driverName,
		Parameters:           map[string]string{"storageDomainName": "", "thinProvisioning": "true"},
		ReclaimPolicy:        &reclaimPolicy,
		MountOptions:         []string{},
		AllowVolumeExpansion: &allowVolumeExpansion,
	}
	expected.Annotations = map[string]string{
		defaultStorageClassAnnotation: "false",
	}
	r.addOwnerLabels(&expected.ObjectMeta, cr)
	return expected
}

// generateCSIDriver prepares a CSIDriver from given template
func (r *ReconcileOvirtCSIOperator) generateCSIDriver(cr *v1alpha1.OvirtCSIOperator) *storagev1beta1.CSIDriver {
	expected := &storagev1beta1.CSIDriver{
		ObjectMeta: metav1.ObjectMeta{Name: driverName},
		Spec: storagev1beta1.CSIDriverSpec{
			AttachRequired: boolPtr(true),
			PodInfoOnMount: boolPtr(true),
		},
	}
	r.addOwnerLabels(&expected.ObjectMeta, cr)
	return expected
}

// sanitizeDriverName sanitizes CSI driver name to be usable as a directory name. All dangerous characters are replaced
// by '-'.
func sanitizeDriverName(driver string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9-.]")
	name := re.ReplaceAllString(driver, "-")
	return name
}

func (r *ReconcileOvirtCSIOperator) addOwner(meta *metav1.ObjectMeta, cr *v1alpha1.OvirtCSIOperator) {
	bTrue := true
	meta.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: csidriverv1alpha1.SchemeGroupVersion.String(),
			Kind:       "OvirtCSIOperator",
			Name:       cr.Name,
			UID:        cr.UID,
			Controller: &bTrue,
		},
	}
}

func (r *ReconcileOvirtCSIOperator) addOwnerLabels(meta *metav1.ObjectMeta, cr *csidriverv1alpha1.OvirtCSIOperator) bool {
	changed := false
	if meta.Labels == nil {
		meta.Labels = map[string]string{}
		changed = true
	}
	if v, exists := meta.Labels[OwnerLabelNamespace]; !exists || v != cr.Namespace {
		meta.Labels[OwnerLabelNamespace] = cr.Namespace
		changed = true
	}
	if v, exists := meta.Labels[OwnerLabelName]; !exists || v != cr.Name {
		meta.Labels[OwnerLabelName] = cr.Name
		changed = true
	}

	return changed
}

func (r *ReconcileOvirtCSIOperator) uniqueGlobalName(i *v1alpha1.OvirtCSIOperator) string {
	return "ovirtcsioperator-" + string(i.UID)
}

func boolPtr(val bool) *bool {
	return &val
}
