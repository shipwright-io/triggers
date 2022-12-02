package controllers

import (
	"context"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/inventory"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// InventoryReconciler reconciles Build instances on the Inventory.
type InventoryReconciler struct {
	client.Client                 // kubernetes client
	Scheme        *runtime.Scheme // shared scheme
	Clock                         // local clock instance

	buildInventory *inventory.Inventory // local build triggers database
}

//+kubebuilder:rbac:groups=shipwright.io,resources=builds,verbs=get;list;watch

// Reconcile reconciles Build instances reflecting it's status on the Inventory.
func (r *InventoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var b v1alpha1.Build
	if err := r.Get(ctx, req.NamespacedName, &b); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Unable to fetch Build, removing from the Inventory")
		}
		r.buildInventory.Remove(req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if b.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.V(0).Info("Adding Build on the Inventory")
		r.buildInventory.Add(&b)
	} else {
		logger.V(0).Info("Removing Build from the Inventory, marked for deletion")
		r.buildInventory.Remove(req.NamespacedName)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager uses the manager to watch over Builds.
func (r *InventoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Build{}).
		Complete(r)
}

// NewInventoryReconciler instantiate the InventoryReconciler.
func NewInventoryReconciler(
	ctrlClient client.Client,
	scheme *runtime.Scheme,
	buildInventory *inventory.Inventory,
) *InventoryReconciler {
	return &InventoryReconciler{
		Client:         ctrlClient,
		Scheme:         scheme,
		buildInventory: buildInventory,
	}
}
