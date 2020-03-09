package resourceapply

import (
	"context"

	"github.com/golang/glog"
	cloudcredreqv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"github.com/openshift/library-go/pkg/operator/resource/resourcemerge"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func boolPtr(val bool) *bool {
	return &val
}

// ApplyCSIDriver merges objectmeta, tries to write everything else.
func ApplyCSIDriver(ctx context.Context, client client.Client, required *storagev1beta1.CSIDriver) (*storagev1beta1.CSIDriver, bool, error) {
	existing := &storagev1beta1.CSIDriver{}

	err := client.Get(ctx, types.NamespacedName{Name: required.Name, Namespace: required.Namespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		err := client.Create(ctx, required)
		if err != nil {
			return nil, false, err
		}
		glog.V(2).Infof("Created CSIDriver %s/%s", required.Namespace, required.Name)
		return required, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	modified := boolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	if !*modified {
		return existing, false, nil
	}

	err = client.Update(ctx, existing)
	if err != nil {
		return nil, false, err
	}
	glog.V(2).Infof("Updated ServiceAccount %s/%s", required.Namespace, required.Name)
	return existing, true, nil
}

// ApplyServiceAccount merges objectmeta, tries to write everything else.
func ApplyServiceAccount(ctx context.Context, client client.Client, required *corev1.ServiceAccount) (*corev1.ServiceAccount, bool, error) {
	existing := &corev1.ServiceAccount{}
	err := client.Get(ctx, types.NamespacedName{Name: required.Name, Namespace: required.Namespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		err := client.Create(ctx, required)
		if err != nil {
			return nil, false, err
		}
		glog.V(2).Infof("Created ServiceAccount %s/%s", required.Namespace, required.Name)
		return required, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	modified := boolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	if !*modified {
		return existing, false, nil
	}

	err = client.Update(ctx, existing)
	if err != nil {
		return nil, false, err
	}
	glog.V(2).Infof("Updated ServiceAccount %s/%s", required.Namespace, required.Name)
	return existing, true, nil
}

// ApplyClusterRole creates cluster role
func ApplyClusterRole(ctx context.Context, client client.Client, cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, bool, error) {
	err := client.Create(ctx, cr)
	if err != nil {
		return nil, false, err
	}

	return cr, true, nil
}

// ApplyClusterRoleBinding merges objectmeta, requires subjects and role refs
func ApplyClusterRoleBinding(ctx context.Context, client client.Client, required *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, bool, error) {
	existing := &rbacv1.ClusterRoleBinding{}
	err := client.Get(ctx, types.NamespacedName{Name: required.Name, Namespace: required.Namespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		err := client.Create(ctx, required)
		if err != nil {
			return nil, false, err
		}
		glog.V(2).Infof("Created ClusterRoleBinding %s", required.Name)
		return required, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	modified := boolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	contentSame := equality.Semantic.DeepEqual(existing.Subjects, required.Subjects) &&
		equality.Semantic.DeepEqual(existing.RoleRef, required.RoleRef)
	if contentSame && !*modified {
		return existing, false, nil
	}
	existing.Subjects = required.Subjects
	existing.RoleRef = required.RoleRef

	err = client.Update(ctx, existing)
	if err != nil {
		return nil, false, err
	}
	glog.V(2).Infof("Updated ClusterRoleBinding %s", required.Name)
	return existing, true, nil
}

// ApplyRoleBinding merges objectmeta, requires subjects and role refs
func ApplyRoleBinding(ctx context.Context, client client.Client, required *rbacv1.RoleBinding) (*rbacv1.RoleBinding, bool, error) {
	existing := &rbacv1.RoleBinding{}
	err := client.Get(ctx, types.NamespacedName{Name: required.Name, Namespace: required.Namespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		err := client.Create(ctx, required)
		if err != nil {
			return nil, false, err
		}
		glog.V(2).Infof("Created RoleBinding %s/%s", required.Namespace, required.Name)
		return required, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	modified := boolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	contentSame := equality.Semantic.DeepEqual(existing.Subjects, required.Subjects) &&
		equality.Semantic.DeepEqual(existing.RoleRef, required.RoleRef)
	if contentSame && !*modified {
		return existing, false, nil
	}
	existing.Subjects = required.Subjects
	existing.RoleRef = required.RoleRef

	err = client.Update(ctx, existing)
	if err != nil {
		return nil, false, err
	}
	glog.V(2).Infof("Updated RoleBinding %s/%s", required.Namespace, required.Name)
	return existing, true, nil
}

// ApplyStatefulSet merges objectmeta and requires matching generation. It returns the final Object, whether any change as made, and an error
func ApplyStatefulSet(ctx context.Context, client client.Client, required *appsv1.StatefulSet, expectedGeneration int64) (*appsv1.StatefulSet, bool, error) {
	existing := &appsv1.StatefulSet{}
	err := client.Get(ctx, types.NamespacedName{Name: required.Name, Namespace: required.Namespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		err := client.Create(ctx, required)
		if err != nil {
			return nil, false, err
		}
		glog.V(2).Infof("Created StatefulSet %s/%s", required.Namespace, required.Name)
		return required, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	modified := boolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	// there was no change to metadata, the generation was right, and we weren't asked for force the deployment
	if !*modified && existing.ObjectMeta.Generation == expectedGeneration {
		return existing, false, nil
	}

	// at this point we know that we're going to perform a write.  We're just trying to get the object correct
	toWrite := existing // shallow copy so the code reads easier
	toWrite.Spec = *required.Spec.DeepCopy()
	err = client.Update(ctx, toWrite)
	if err != nil {
		return nil, false, err
	}
	glog.V(2).Infof("Updated StatefulSet %s/%s", required.Namespace, required.Name)
	return toWrite, true, nil
}

// ApplyDaemonSet merges objectmeta and requires matching generation. It returns the final Object, whether any change as made, and an error
func ApplyDaemonSet(ctx context.Context, client client.Client, required *appsv1.DaemonSet, expectedGeneration int64, templateChanged bool) (*appsv1.DaemonSet, bool, error) {
	existing := &appsv1.DaemonSet{}
	err := client.Get(ctx, types.NamespacedName{Name: required.Name, Namespace: required.Namespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		err := client.Create(ctx, required)
		if err != nil {
			return nil, false, err
		}
		glog.V(2).Infof("Created DaemonSet %s/%s", required.Namespace, required.Name)
		return required, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	modified := resourcemerge.BoolPtr(false)
	resourcemerge.EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	// there was no change to metadata, the generation was right, and we weren't asked for force the deployment
	if !*modified && existing.ObjectMeta.Generation == expectedGeneration && !templateChanged {
		return existing, false, nil
	}

	// at this point we know that we're going to perform a write.  We're just trying to get the object correct
	toWrite := existing // shallow copy so the code reads easier
	toWrite.Spec = *required.Spec.DeepCopy()
	err = client.Update(ctx, toWrite)
	if err != nil {
		return nil, false, err
	}
	glog.V(2).Infof("Updated DaemonSet %s/%s", required.Namespace, required.Name)
	return toWrite, true, nil
}

// ApplyStorageClass merges objectmeta, tries to write everything else
func ApplyStorageClass(ctx context.Context, client client.Client, required *storagev1.StorageClass) (*storagev1.StorageClass, bool, error) {
	existing := &storagev1.StorageClass{}
	err := client.Get(ctx, types.NamespacedName{Name: required.Name, Namespace: required.Namespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		err := client.Create(ctx, required)
		if err != nil {
			return nil, false, err
		}
		glog.V(2).Infof("Created StorageClass %s", required.Name)
		return required, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	changed := false
	resourcemerge.EnsureObjectMeta(&changed, &existing.ObjectMeta, required.ObjectMeta)

	if !equality.Semantic.DeepEqual(required.MountOptions, existing.MountOptions) {
		changed = true
		existing.MountOptions = required.MountOptions
	}

	allowedExpansionEqual := true
	if existing.AllowVolumeExpansion == nil && required.AllowVolumeExpansion != nil {
		allowedExpansionEqual = false
	}
	if existing.AllowVolumeExpansion != nil && required.AllowVolumeExpansion == nil {
		allowedExpansionEqual = false
	}
	if existing.AllowVolumeExpansion != nil && required.AllowVolumeExpansion != nil && *existing.AllowVolumeExpansion != *required.AllowVolumeExpansion {
		allowedExpansionEqual = false
	}
	if !allowedExpansionEqual {
		changed = true
		existing.AllowVolumeExpansion = required.AllowVolumeExpansion
	}

	if !equality.Semantic.DeepEqual(existing.AllowedTopologies, required.AllowedTopologies) {
		changed = true
		existing.AllowedTopologies = required.AllowedTopologies
	}

	if !changed {
		return existing, false, nil
	}
	err = client.Update(ctx, existing)
	if err != nil {
		return nil, false, err
	}
	glog.V(2).Infof("Updated StorageClass %s", required.Name)
	return existing, true, nil
}

// ApplyStorageClass merges objectmeta, tries to write everything else
func ApplyCredentialsRequest(ctx context.Context, client client.Client, required *cloudcredreqv1.CredentialsRequest) (*cloudcredreqv1.CredentialsRequest, bool, error) {
	existing := &cloudcredreqv1.CredentialsRequest{}
	err := client.Get(ctx, types.NamespacedName{Name: required.Name, Namespace: required.Namespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		err := client.Create(ctx, required)
		if err != nil {
			return nil, false, err
		}
		glog.V(2).Infof("Created CredentialsRequest %s", required.Name)
		return required, true, nil
	}

	return nil, false, err
}
