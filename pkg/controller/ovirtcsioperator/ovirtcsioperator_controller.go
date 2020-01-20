package ovirtcsioperator

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ovirtv1alpha1 "github.com/ovirt/csi-driver/pkg/apis/ovirt/v1alpha1"
	"github.com/ovirt/csi-driver/pkg/assets"
)

var log = logf.Log.WithName("controller_ovirtcsioperator")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new OvirtCSIOperator Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileOvirtCSIOperator{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor("ovirt-csi-driver-operartor"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("ovirtcsioperator-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource OvirtCSIOperator
	err = c.Watch(&source.Kind{Type: &ovirtv1alpha1.OvirtCSIOperator{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner OvirtCSIOperator
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ovirtv1alpha1.OvirtCSIOperator{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileOvirtCSIOperator implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileOvirtCSIOperator{}

// ReconcileOvirtCSIOperator reconciles a OvirtCSIOperator object
type ReconcileOvirtCSIOperator struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a OvirtCSIOperator object and makes changes based on the state read
// and what is in the OvirtCSIOperator.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOvirtCSIOperator) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling OvirtCSIOperator")

	// Fetch the OvirtCSIOperator instance
	instance := &ovirtv1alpha1.OvirtCSIOperator{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Pod object
	manifests, err := loadCSIDriverManifests(instance)
	if err != nil {
		return , err
	}


	for _, manifest := range manifests {
		// check if the object exists
		switch manifest.GetObjectKind().GroupVersionKind().Kind {
		case "StorageClass":
			r.syncStorageClass()

		case "CSIDriver":
			r.syncStorageClass()

		case: "StatefulSet":
			r.syncDaemonSet()

		case "CredentialsRequest":
			r.syncStorageClass()

		default:
			// nothing
		}

		err = r.client.Get(context.TODO(), types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, manifest)

	}
	// Set OvirtCSIOperator instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
}

// loadCSIDriverManifests returns a busybox pod with the same name/namespace as the cr
func loadCSIDriverManifests(cr *ovirtv1alpha1.OvirtCSIOperator) ([]runtime.Object, error) {
	yamls := assets.AssetNames()
	retVal := make([]runtime.Object, 0, len(yamls))
	for _, y := range yamls {
		data, err := assets.Asset(y)
		if err != nil {
			return nil, err
		}
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode(data, nil, nil)

		if err != nil {
			return nil, err
		}

		retVal = append(retVal, obj)
	}
	return retVal, nil
}


