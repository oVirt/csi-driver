package config

import (
	csi "github.com/ovirt/csi-driver/pkg/apis/ovirt/v1alpha1"
)

// Config is configuration of the CSI Driver operator.
type Config struct {
	// Default sidecar container images used when CR does not specify anything special.
	DefaultImages csi.CSIDeploymentContainerImages `yaml:"defaultImages,omitempty"`

	// Selector of nodes where Deployment with controller components (provisioner, attacher) can run.
	// When nil, no selector will be set in the Deployment.
	InfrastructureNodeSelector map[string]string `yaml:"infrastructureNodeSelector,omitempty"`

	// Number of replicas of Deployment with controller components.
	DeploymentReplicas int32 `yaml:"deploymentReplicas,omitempty"`

	// Name of cluster role to bind to ServiceAccount that runs all pods with drivers. This role allows to run
	// provisioner, attacher and driver registrar, i.e. read/modify PV, PVC, Node, VolumeAttachment and whatnot in
	// *any* namespace.
	// TODO: should there be multiple ClusterRoles, separate one for provisioner, attacher and driver registrar?
	// In addition, some of them may require variants (e.g. provisioner without access to all secrets and / or attacher
	// without access to all secrets)
	ClusterRoleName string `yaml:"clusterRoleName,omitempty"`

	// Name of cluster role to bind to ServiceAccount that runs all pods with drivers. This role allows attacher and
	// provisioner to run leader election. It will be bound to the ServiceAccount using RoleBind, i.e. leader election
	// will be possible only in the namespace where the drivers run.
	LeaderElectionClusterRoleName string `yaml:"leaderElectionClusterRoleName,omitempty"`

	// Path to /var/lib/kubelet.
	KubeletRootDir string `yaml:"kubeletRootDir,omitempty"`
}
