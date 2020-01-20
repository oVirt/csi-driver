package apis

import (
	configv1 "github.com/openshift/api/config/v1"
	cloudcredreqv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"github.com/ovirt/csi-driver/pkg/apis/ovirt/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
	AddToSchemes = append(AddToSchemes, configv1.AddToScheme)
	AddToSchemes = append(AddToSchemes, cloudcredreqv1.SchemeBuilder.AddToScheme)
	AddToSchemes = append(AddToSchemes, metav1.AddMetaToScheme)
}
