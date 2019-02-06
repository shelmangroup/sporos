package sporos

import (
	"context"
	"fmt"

	shelmanv1alpha1 "github.com/shelmangroup/sporos/pkg/apis/shelman/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_sporos")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Sporos Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSporos{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("sporos-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Sporos
	err = c.Watch(&source.Kind{Type: &shelmanv1alpha1.Sporos{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Sporos
	// err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &shelmanv1alpha1.Sporos{},
	// })
	// if err != nil {
	// 	return err
	// }

	return nil
}

var _ reconcile.Reconciler = &ReconcileSporos{}

// ReconcileSporos reconciles a Sporos object
type ReconcileSporos struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Sporos object and makes changes based on the state read
// and what is in the Sporos.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSporos) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Sporos")

	// Fetch the Sporos instance
	instance := &shelmanv1alpha1.Sporos{}
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

	// After first time reconcile, phase will switch to "Running".
	if instance.Status.Phase == shelmanv1alpha1.ControlplanePhaseInitial {
		if instance.Status.ApiServerIP == "" {
			svc, err := r.createExternalEndpoint(instance)
			if err != nil {
				return reconcile.Result{}, err
			}
			svcReady, err := r.isServiceEndpointReady(instance, svc)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to check if etcd cluster is ready: %v", err)
			}
			if !svcReady {
				reqLogger.Info("Waiting for service (%v) to become ready", svc.Name)
				return reconcile.Result{}, nil
			}
			err = r.prepareAssets(instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		// etcd cluster should only be created in first time reconcile.
		ec, err := r.deployEtcdCluster(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Check if etcd cluster is up and running.
		// If not, we need to wait until etcd cluster is up before proceeding to the next state;
		// Hence, we return from here and let the Watch triggers the handler again.
		ready, err := r.isEtcdClusterReady(ec)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to check if etcd cluster is ready: %v", err)
		}
		if !ready {
			log.Info("Waiting for EtcdCluster (%v) to become ready", ec.Name)
			return reconcile.Result{}, nil
		}

		deploys, err := r.deployControlplane(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		for _, d := range deploys {
			ready, err := r.IsControlplaneReady(d)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to check if %v cluster is ready: %v", d.GetName(), err)
			}
			if !ready {
				log.Info("Waiting for controlplane (%v) to become ready", d.GetName())
				return reconcile.Result{}, nil
			}
		}

		client, err := r.NewKubeClient(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		err = r.csrBootstrap(client)
		if err != nil {
			return reconcile.Result{}, err
		}

		log.Info("%v is ready!", instance.Name)
		instance.Status.Phase = "Running"
		r.client.Update(context.TODO(), instance)
	}

	bup, err := r.backupEtcdCluster(instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	ready, err := r.isEtcdClusterBackupReady(bup)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check if %v cluster is ready: %v", bup.GetName(), err)
	}
	if !ready {
		log.Info("Waiting for backup (%v) to become ready", bup.GetName())
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
	// Define a new Pod object
	// pod := newPodForCR(instance)
	//
	// // Set Sporos instance as the owner and controller
	// if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	// }
	//
	// // Check if this Pod already exists
	// found := &corev1.Pod{}
	// err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	// if err != nil && errors.IsNotFound(err) {
	// 	reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
	// 	err = r.client.Create(context.TODO(), pod)
	// 	if err != nil {
	// 		return reconcile.Result{}, err
	// 	}
	//
	// 	// Pod created successfully - don't requeue
	// 	return reconcile.Result{}, nil
	// } else if err != nil {
	// 	return reconcile.Result{}, err
	// }
	//
	// // Pod already exists - don't requeue
	// reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	// return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *shelmanv1alpha1.Sporos) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
