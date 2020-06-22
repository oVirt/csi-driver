package ovirtcsioperator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	configv1 "github.com/openshift/api/config/v1"
	openshiftapi "github.com/openshift/api/operator/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorhelpersv1 "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1helpers "github.com/ovirt/csi-driver/pkg/apis/ovirt/helpers"

	"github.com/ovirt/csi-driver/pkg/apis/ovirt/v1alpha1"
	"github.com/ovirt/csi-driver/pkg/resourceapply"
)

const (
	finalizerName = "ovirt.csidriver.storage.openshift.io"
	apiTimeout    = time.Minute
)

var (
	infos = []clusterRoleBindingInfo{
		{"ovirt-csi-controller-cr-role-binding", "ovirt-csi-controller-sa", "ovirt-csi-controller-cr"},
		{"ovirt-csi-controller-le-role-binding", "ovirt-csi-controller-sa", "openshift:csi-driver-controller-leader-election"},
		{"ovirt-csi-node-cr-role-binding", "ovirt-csi-node-sa", "ovirt-csi-node-cr"},
		{"ovirt-csi-node-le-role-binding", "ovirt-csi-node-sa", "openshift:csi-driver-controller-leader-election"},
	}
)

type clusterRoleBindingInfo struct {
	name           string
	serviceAccount string
	roleRefName    string
}

func (r *ReconcileOvirtCSIOperator) handleCSIDriverDeployment(instance *v1alpha1.OvirtCSIOperator) error {
	var errs []error
	newInstance := instance.DeepCopy()

	if newInstance.DeletionTimestamp != nil {
		// The deployment is being deleted, clean up.
		// Allow deletion without validation.
		newInstance.Status.State = openshiftapi.Removed
		newInstance, errs = r.cleanupCSIDriverDeployment(newInstance)

	} else {
		// The deployment was created / updated
		newInstance.Status.State = openshiftapi.Managed
		// Sync the CSIDriverDeployment only when validation passed.
		newInstance, errs = r.syncCSIDriverDeployment(newInstance)
	}
	if errs != nil {
		// Send errors as events
		for _, e := range errs {
			glog.Warning(e.Error())
			if !errors.IsConflict(e) {
				r.recorder.Event(newInstance, corev1.EventTypeWarning, "SyncError", e.Error())
			}
		}
	}

	err := r.syncStatus(instance, newInstance)
	if err != nil {
		// This error has not been logged above
		glog.Warning(err.Error())
		if !errors.IsConflict(err) {
			r.recorder.Event(newInstance, corev1.EventTypeWarning, "SyncError", err.Error())
		}
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return utilerrors.NewAggregate(errs)
	}

	return nil
}

func (r *ReconcileOvirtCSIOperator) syncFinalizer(cr *v1alpha1.OvirtCSIOperator) (*v1alpha1.OvirtCSIOperator, error) {
	logf.Log.Info("Syncing CSIDriverDeployment.Finalizers")

	if hasFinalizer(cr.Finalizers, finalizerName) {
		return cr, nil
	}

	newCR := cr.DeepCopy()
	if newCR.Finalizers == nil {
		newCR.Finalizers = []string{}
	}
	newCR.Finalizers = append(newCR.Finalizers, finalizerName)

	ctx, cancel := r.apiContext()
	defer cancel()
	glog.V(4).Infof("Updating CSIDriverDeployment.Finalizers")
	if err := r.client.Update(ctx, newCR); err != nil {
		return cr, err
	}

	return newCR, nil
}

// syncCSIDriverDeployment checks one CSIDriverDeployment and ensures that all "children" objects are either
// created or updated.
func (r *ReconcileOvirtCSIOperator) syncCSIDriverDeployment(cr *v1alpha1.OvirtCSIOperator) (*v1alpha1.OvirtCSIOperator, []error) {
	logf.Log.Info("Syncing CSIDriverDeployment")
	var errs []error

	cr, err := r.syncFinalizer(cr)
	if err != nil {
		// Return now, we can't create subsequent objects without the finalizer because could miss event
		// with CSIDriverDeployment deletion and we could not delete non-namespaced objects.
		return cr, []error{err}
	}

	err = r.syncCSIDriver(cr)
	if err != nil {
		logf.Log.Error(err, "CSI Driver")
		errs = append(errs, err)
	}

	err = r.syncCredentialsReuest(cr)
	if err != nil {
		logf.Log.Error(err, "Cloud creds")

		errs = append(errs, err)
	}
	rolesErrs := r.syncClusterRoles(cr)
	if rolesErrs != nil {
		logf.Log.Error(err, "Cluster roles")
		errs = append(errs, rolesErrs...)
	}
	err = r.syncRBAC(cr)
	if err != nil {
		logf.Log.Error(err, "rbac")

		errs = append(errs, err)
	}
	ds, err := r.syncDaemonSet(cr)
	if err != nil {
		logf.Log.Error(err, "daemonset")
		errs = append(errs, err)
	}
	statefulSet, err := r.syncStatefulSet(cr)
	if err != nil {
		logf.Log.Error(err, "statefulset")
		errs = append(errs, err)
	}

	co, err := r.syncClusterOperator(cr)
	if err != nil {
		logf.Log.Error(err, "cluster operator")
		errs = append(errs, err)
	}

	err = r.syncClusterOperatorConditions(co, statefulSet, ds)
	if err != nil {
		logf.Log.Error(err, "cluster operator sync conditions failed")
		errs = append(errs, err)
	}

	var children []openshiftapi.GenerationHistory

	if ds != nil {
		// Store generation of the DaemonSet so we can check for DaemonSet.Spec changes.
		children = append(children, openshiftapi.GenerationHistory{
			Group:          appsv1.GroupName,
			Resource:       "DaemonSet",
			Namespace:      ds.Namespace,
			Name:           ds.Name,
			LastGeneration: ds.Generation,
		})
	}

	if statefulSet != nil {
		// Store generation of the Deployment so we can check for DaemonSet.Spec changes.
		children = append(children, openshiftapi.GenerationHistory{
			Group:          appsv1.GroupName,
			Resource:       "StatefulSet",
			Namespace:      statefulSet.Namespace,
			Name:           statefulSet.Name,
			LastGeneration: statefulSet.Generation,
		})
	}

	cr.Status.Children = children
	var filteredErrors []error
	for _, e := range errs {
		if !errors.IsAlreadyExists(e) {
			filteredErrors = append(filteredErrors, e)
		}
	}
	if len(filteredErrors) == 0 {
		cr.Status.ObservedGeneration = &cr.Generation
	}
	r.syncConditions(cr, statefulSet, ds, filteredErrors)

	return cr, filteredErrors
}

func (r *ReconcileOvirtCSIOperator) syncServiceAccount(cr *v1alpha1.OvirtCSIOperator) error {
	logf.Log.Info("Syncing ServiceAccount")

	controllerAccount := r.generateServiceAccount("ovirt-csi-controller-sa", cr)
	nodeAccount := r.generateServiceAccount("ovirt-csi-node-sa", cr)

	ctx, cancel := r.apiContext()
	defer cancel()
	_, _, err := resourceapply.ApplyServiceAccount(ctx, r.client, controllerAccount)
	if err != nil {
		return err
	}
	_, _, err = resourceapply.ApplyServiceAccount(ctx, r.client, nodeAccount)
	if err != nil {
		return err
	}
	return nil
}

func (r *ReconcileOvirtCSIOperator) syncClusterRoles(cr *v1alpha1.OvirtCSIOperator) []error {
	ctx, cancel := r.apiContext()
	var errs []error
	defer cancel()

	_, _, err := resourceapply.ApplyClusterRole(ctx, r.client, r.generateClusterRoleController(cr))
	if err != nil {
		logf.Log.Error(err, "ApplyClusterRole generateClusterRoleController")
		if !errors.IsAlreadyExists(err) {
			errs = append(errs, err)
		}
	}

	_, _, err = resourceapply.ApplyClusterRole(ctx, r.client, r.generateClusterRoleNode(cr))
	if err != nil {
		logf.Log.Error(err, "ApplyClusterRole generateClusterRoleNode")
		if !errors.IsAlreadyExists(err) {
			errs = append(errs, err)
		}
	}
	_, _, err = resourceapply.ApplyClusterRole(ctx, r.client, r.generateClusterRoleLeaderElection(cr))
	if err != nil {
		logf.Log.Error(err, "ApplyClusterRole generateClusterRoleLeaderElection")
		if !errors.IsAlreadyExists(err) {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *ReconcileOvirtCSIOperator) syncRBAC(cr *v1alpha1.OvirtCSIOperator) error {
	err := r.syncServiceAccount(cr)
	if err != nil {
		return err
	}
	for _, bindingInfo := range infos {
		err := r.syncClusterRoleBinding(cr, bindingInfo.name, bindingInfo.serviceAccount, bindingInfo.roleRefName)
		if err != nil {
			return err
		}
	}
	return nil

}
func (r *ReconcileOvirtCSIOperator) syncClusterRoleBinding(cr *v1alpha1.OvirtCSIOperator, name string, serviceAccount string, roleName string) error {
	logf.Log.Info("Syncing ClusterRoleBinding")

	crb := r.generateClusterRoleBinding(cr, name, serviceAccount, roleName)

	ctx, cancel := r.apiContext()
	defer cancel()
	_, _, err := resourceapply.ApplyClusterRoleBinding(ctx, r.client, crb)
	return err
}

func (r *ReconcileOvirtCSIOperator) syncLeaderElectionRoleBinding(cr *v1alpha1.OvirtCSIOperator, serviceAccount *corev1.ServiceAccount) error {
	logf.Log.Info("Syncing leader election RoleBinding")

	rb := r.generateLeaderElectionRoleBinding(cr, serviceAccount)

	ctx, cancel := r.apiContext()
	defer cancel()
	_, _, err := resourceapply.ApplyRoleBinding(ctx, r.client, rb)
	return err
}

func (r *ReconcileOvirtCSIOperator) syncDaemonSet(cr *v1alpha1.OvirtCSIOperator) (*appsv1.DaemonSet, error) {
	logf.Log.Info("Syncing DaemonSet")
	requiredDS := r.generateDaemonSet(cr)
	gvk := appsv1.SchemeGroupVersion.WithKind("DaemonSet")
	generation := r.getExpectedGeneration(cr, requiredDS, gvk)

	ctx, cancel := r.apiContext()
	defer cancel()
	ds, _, err := resourceapply.ApplyDaemonSet(ctx, r.client, requiredDS, generation, false)
	if err != nil {
		return requiredDS, err
	}
	return ds, nil
}

func (r *ReconcileOvirtCSIOperator) syncStatefulSet(cr *v1alpha1.OvirtCSIOperator) (*appsv1.StatefulSet, error) {
	logf.Log.Info("Syncing StatefulSet")

	required := r.generateStatefulSet(cr)
	gvk := appsv1.SchemeGroupVersion.WithKind("StatefulSet")
	generation := r.getExpectedGeneration(cr, required, gvk)

	ctx, cancel := r.apiContext()
	defer cancel()
	statefulSet, _, err := resourceapply.ApplyStatefulSet(ctx, r.client, required, generation)
	if err != nil {
		return required, err
	}
	return statefulSet, nil
}

func (r *ReconcileOvirtCSIOperator) syncCSIDriver(cr *v1alpha1.OvirtCSIOperator) error {
	logf.Log.Info("Syncing CSIDriver")

	sc := r.generateCSIDriver(cr)
	ctx, cancel := r.apiContext()
	defer cancel()
	_, _, err := resourceapply.ApplyCSIDriver(ctx, r.client, sc)

	return err
}

func (r *ReconcileOvirtCSIOperator) syncCredentialsReuest(cr *v1alpha1.OvirtCSIOperator) error {
	logf.Log.Info("Syncing CredentialsRequest")

	required, err := r.generateCredentialsRequest(cr)
	if err != nil {
		return err
	}
	ctx, cancel := r.apiContext()
	defer cancel()
	_, _, err = resourceapply.ApplyCredentialsRequest(ctx, r.client, required)
	return err
}

func (r *ReconcileOvirtCSIOperator) syncClusterOperator(cr *v1alpha1.OvirtCSIOperator) (*configv1.ClusterOperator, error) {
	logf.Log.Info("Syncing ClusterOperator")

	co := r.generateClusterOperator(cr)
	ctx, cancel := r.apiContext()
	defer cancel()
	logf.Log.Info(fmt.Sprintf("Cluster operator: %v", co))

	_, _, err := resourceapply.ApplyClusterOperator(ctx, r.client, co)
	logf.Log.Error(err, "error in Syncing ClusterOperator")
	if !errors.IsAlreadyExists(err) {

		return nil, err
	}

	return co, nil
}

func (r *ReconcileOvirtCSIOperator) removeUnexpectedStorageClasses(cr *v1alpha1.OvirtCSIOperator, expectedClasses sets.String) []error {
	list := &storagev1.StorageClassList{}
	opts := client.ListOptions{}
	ctx, cancel := r.apiContext()
	defer cancel()
	err := r.client.List(ctx, list, &opts)
	if err != nil {
		err := fmt.Errorf("cannot list StorageClasses for CSIDriverDeployment %s/%s: %s", cr.Namespace, cr.Name, err)
		return []error{err}
	}

	var errs []error
	for _, sc := range list.Items {
		if !expectedClasses.Has(sc.Name) {
			glog.V(4).Infof("Deleting StorageClass %s", sc.Name)
			ctx, cancel := r.apiContext()
			defer cancel()
			if err := r.client.Delete(ctx, &sc); err != nil {
				if !errors.IsNotFound(err) {
					err := fmt.Errorf("cannot delete StorageClasses %s for CSIDriverDeployment %s/%s: %s", sc.Name, cr.Namespace, cr.Name, err)
					errs = append(errs, err)
				}
			}
		}
	}
	return errs
}

func (r *ReconcileOvirtCSIOperator) syncConditions(instance *v1alpha1.OvirtCSIOperator, statefulSet *appsv1.StatefulSet, ds *appsv1.DaemonSet, errs []error) {
	// OperatorStatusTypeAvailable condition: true if both Deployment and DaemonSet are fully deployed.
	logf.Log.Info("sync Conditions")

	availableCondition := openshiftapi.OperatorCondition{
		Type: openshiftapi.OperatorStatusTypeAvailable,
	}
	available := true
	unknown := false
	var msgs []string

	logf.Log.Info("Checking statefulset")
	if statefulSet != nil {
		if statefulSet.Status.ReadyReplicas != replicas {
			available = false
			msgs = append(msgs, fmt.Sprintf("StatefulSet %q with CSI driver needs %v but has %v ready.", statefulSet.Name, statefulSet.Status.ReadyReplicas, statefulSet.Status.Replicas))
		}
	} else {
		unknown = true
	}

	logf.Log.Info("Checking daemonset")
	if ds != nil {
		if ds.Status.NumberUnavailable > 0 {
			available = false
			msgs = append(msgs, fmt.Sprintf("DaemonSet msgs%q with CSI driver has %v not ready pod(s).", ds.Name, ds.Status.NumberUnavailable))
		}
	} else {
		unknown = true
	}

	switch {
	case unknown:
		availableCondition.Status = openshiftapi.ConditionUnknown
	case available:
		availableCondition.Status = openshiftapi.ConditionTrue
	default:
		availableCondition.Status = openshiftapi.ConditionFalse
	}
	availableCondition.Message = strings.Join(msgs, "\n")
	v1alpha1helpers.SetOperatorCondition(&instance.Status.Conditions, availableCondition)

	// OperatorStatusTypeSyncSuccessful condition: true if no error happened during sync.
	syncSuccessfulCondition := openshiftapi.OperatorCondition{
		Type:    openshiftapi.OperatorStatusTypeSyncSuccessful,
		Status:  openshiftapi.ConditionTrue,
		Message: "",
	}
	if len(errs) > 0 {
		logf.Log.Info("Errors encountered")
		syncSuccessfulCondition.Status = openshiftapi.ConditionFalse
		errStrings := make([]string, len(errs))
		for i := range errs {
			errStrings[i] = errs[i].Error()
		}
		syncSuccessfulCondition.Message = strings.Join(errStrings, "\n")
	}
	ctx, cancel := r.apiContext()
	defer cancel()

	v1alpha1helpers.SetOperatorCondition(&instance.Status.Conditions, syncSuccessfulCondition)
	r.client.Update(ctx, instance)
}

func (r *ReconcileOvirtCSIOperator) syncClusterOperatorConditions(co *configv1.ClusterOperator, sf *appsv1.StatefulSet, ds *appsv1.DaemonSet) error {
	logf.Log.Info("sync ClusterOperator conditions")
	targetLevel := os.Getenv("RELEASE_VERSION")
	logf.Log.Info(fmt.Sprintf("Target level %s", targetLevel))

	ctx, cancel := r.apiContext()
	defer cancel()
	logf.Log.Info(fmt.Sprintf("Updating CSI CVO: %v", co))
	existing := &configv1.ClusterOperator{ObjectMeta: metav1.ObjectMeta{Name: co.Name}}
	err := r.client.Get(ctx, types.NamespacedName{Name: co.Name}, existing)
	if err != nil {
		logf.Log.Error(err, fmt.Sprintf("CO %v fetch failed", existing))
		return err
	}

	reachedAvailable := false

	progressing := []string{}
	dsProgrssing := false
	dsAvailable := false

	if ds.Status.CurrentNumberScheduled < ds.Status.DesiredNumberScheduled {
		progressing = append(progressing, fmt.Sprintf("DaemonSet %q is being scheduled (%d out of %d scheduled)", ds.Name, ds.Status.UpdatedNumberScheduled, ds.Status.DesiredNumberScheduled))
		dsProgrssing = true
		dsAvailable = false
	}
	if ds.Status.NumberUnavailable > 0 {
		progressing = append(progressing, fmt.Sprintf("DaemonSet %q is not fully available (awaiting %d nodes)", ds.Name, ds.Status.NumberUnavailable))
		dsProgrssing = true
		dsAvailable = true
	}
	if ds.Status.NumberReady == 0 {
		progressing = append(progressing, fmt.Sprintf("DaemonSet %q is not yet scheduled on any nodes", ds.Name))
		dsProgrssing = true
		dsAvailable = false
	} else {
		progressing = append(progressing, fmt.Sprintf("DaemonSet %q is available", ds.Name))
		dsProgrssing = true
		dsAvailable = true
	}
	if ds.Generation > ds.Status.ObservedGeneration {
		progressing = append(progressing, fmt.Sprintf("DaemonSet %q update is being processed (generation %d, observed generation %d)", ds.Name, ds.Generation, ds.Status.ObservedGeneration))
		dsProgrssing = true
		dsAvailable = true
	}
	if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
		dsProgrssing = false
		dsAvailable = true
	}

	sfProgrssing := false
	sfAvailable := false

	if sf.Status.ReadyReplicas < *sf.Spec.Replicas {
		sfAvailable = false
		sfProgrssing = true
		progressing = append(progressing, fmt.Sprintf("StatefulSet %q with CSI driver needs %v but has %v ready.", sf.Name, sf.Status.ReadyReplicas, sf.Status.Replicas))
	}
	if sf.Status.ReadyReplicas == 0 {
		sfAvailable = false
		sfProgrssing = true
		progressing = append(progressing, fmt.Sprintf("StatefulSet %q does not have any replicas ready.", sf.Name))
	}
	if sf.Status.ReadyReplicas > 0 {
		sfAvailable = true
		sfProgrssing = true
	}
	if sf.Status.ReadyReplicas == sf.Status.Replicas {
		sfAvailable = true
		sfProgrssing = false
	}

	if sfAvailable && dsAvailable {
		reachedAvailable = true
	}

	if sfProgrssing && !sfAvailable {
		logf.Log.Info("sf progressing but not available")

		condition := configv1.ClusterOperatorStatusCondition{
			Type:   configv1.OperatorDegraded,
			Status: configv1.ConditionTrue,
			Reason: "Some StatefulSets are not ready",
		}
		operatorhelpersv1.SetStatusCondition(&existing.Status.Conditions, condition)
	} else {
		condition := configv1.ClusterOperatorStatusCondition{
			Type:   configv1.OperatorDegraded,
			Status: configv1.ConditionFalse,
		}
		operatorhelpersv1.SetStatusCondition(&existing.Status.Conditions, condition)
	}
	if dsProgrssing && !dsAvailable {
		logf.Log.Info("sf progressing but not available")

		condition := configv1.ClusterOperatorStatusCondition{
			Type:   configv1.OperatorDegraded,
			Status: configv1.ConditionTrue,
			Reason: "Some DaemonSets are not ready",
		}
		operatorhelpersv1.SetStatusCondition(&existing.Status.Conditions, condition)
	} else {
		condition := configv1.ClusterOperatorStatusCondition{
			Type:   configv1.OperatorDegraded,
			Status: configv1.ConditionFalse,
		}
		operatorhelpersv1.SetStatusCondition(&existing.Status.Conditions, condition)
	}

	if reachedAvailable {
		logf.Log.Info("operator available")

		existing.Status.Versions = []configv1.OperandVersion{
			{
				Name:    "operator",
				Version: targetLevel,
			},
		}
		condition := configv1.ClusterOperatorStatusCondition{
			Type:   configv1.OperatorAvailable,
			Status: configv1.ConditionTrue,
		}
		operatorhelpersv1.SetStatusCondition(&existing.Status.Conditions, condition)
	}

	for _, p := range progressing {
		condition := configv1.ClusterOperatorStatusCondition{
			Type:    configv1.OperatorProgressing,
			Status:  configv1.ConditionTrue,
			Reason:  "Deploying",
			Message: p,
		}
		operatorhelpersv1.SetStatusCondition(&existing.Status.Conditions, condition)
	}

	logf.Log.Info(fmt.Sprintf("Updating CSI CVO existing: %v", existing))

	if err := r.client.Status().Update(ctx, existing); err != nil {
		logf.Log.Error(err, "CO Update failed")
		return err
	}

	return nil
}

func (r *ReconcileOvirtCSIOperator) syncStatus(oldInstance, newInstance *v1alpha1.OvirtCSIOperator) error {
	glog.V(4).Info("Syncing CSIDriverDeployment.Status")

	if !equality.Semantic.DeepEqual(oldInstance.Status, newInstance.Status) {
		glog.V(4).Info("Updating CSIDriverDeployment.Status")
		ctx, cancel := r.apiContext()
		defer cancel()
		err := r.client.Status().Update(ctx, newInstance)
		return err
	}
	return nil
}

// cleanupCSIDriverDeployment removes non-namespaced objects owned by the CSIDriverDeployment.
// ObjectMeta.OwnerReference does not work for them.
func (r *ReconcileOvirtCSIOperator) cleanupCSIDriverDeployment(cr *v1alpha1.OvirtCSIOperator) (*v1alpha1.OvirtCSIOperator, []error) {
	glog.V(2).Infof("Cleaning up CSIDriverDeployment %s/%s", cr.Namespace, cr.Name)

	errs := r.cleanupStorageClasses(cr)
	if err := r.cleanupClusterRoleBinding(cr); err != nil {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		// Don't remove the finalizer yet, there is still stuff to clean up
		return cr, errs
	}

	// Remove the finalizer as the last step
	newCR, err := r.cleanupFinalizer(cr)
	if err != nil {
		return cr, []error{err}
	}
	return newCR, nil
}

func (r *ReconcileOvirtCSIOperator) cleanupFinalizer(cr *v1alpha1.OvirtCSIOperator) (*v1alpha1.OvirtCSIOperator, error) {
	newCR := cr.DeepCopy()
	newCR.Finalizers = []string{}
	for _, f := range cr.Finalizers {
		if f == finalizerName {
			continue
		}
		newCR.Finalizers = append(newCR.Finalizers, f)
	}

	glog.V(4).Infof("Removing CSIDriverDeployment.Finalizers")
	ctx, cancel := r.apiContext()
	defer cancel()
	err := r.client.Update(ctx, newCR)
	if err != nil {
		return cr, err
	}
	return newCR, nil
}

func (r *ReconcileOvirtCSIOperator) cleanupClusterRoleBinding(cr *v1alpha1.OvirtCSIOperator) error {
	ctx, cancel := r.apiContext()
	defer cancel()
	for _, bindingInfo := range infos {
		crb := r.generateClusterRoleBinding(cr, bindingInfo.name, bindingInfo.serviceAccount, bindingInfo.roleRefName)
		err := r.client.Delete(ctx, crb)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		glog.V(4).Infof("Deleted ClusterRoleBinding %s", crb.Name)
	}
	return nil
}

func (r *ReconcileOvirtCSIOperator) cleanupStorageClasses(cr *v1alpha1.OvirtCSIOperator) []error {
	return r.removeUnexpectedStorageClasses(cr, sets.NewString())
}

func hasFinalizer(finalizers []string, finalizerName string) bool {
	for _, f := range finalizers {
		if f == finalizerName {
			return true
		}
	}
	return false
}

func (r *ReconcileOvirtCSIOperator) getExpectedGeneration(cr *v1alpha1.OvirtCSIOperator, obj runtime.Object, gvk schema.GroupVersionKind) int64 {
	for _, child := range cr.Status.Children {
		if child.Group != gvk.Group || child.Resource != gvk.Kind {
			continue
		}
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return -1
		}
		if child.Name != accessor.GetName() || child.Namespace != accessor.GetNamespace() {
			continue
		}
		return child.LastGeneration
	}
	return -1
}

func (r *ReconcileOvirtCSIOperator) apiContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), apiTimeout)
}
