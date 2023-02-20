package controllers

import (
	"context"
	"fmt"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/filter"

	"github.com/go-logr/logr"
	tknv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage/names"
	"knative.dev/pkg/apis"
	knativev1 "knative.dev/pkg/apis/duck/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// RunReconciler watches over Tekton Run instances carrying a Shipwright Build reference,
// that's the approach Tekton takes to Custom-Tasks, which makes possible to utilize third-party
// controller resources.
type RunReconciler struct {
	client.Client                 // kubernetes client
	Scheme        *runtime.Scheme // shared scheme
	Clock                         // local clock instance
}

//+kubebuilder:rbac:groups=shipwright.io,resources=buildruns,verbs=create;get;list;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=runs,verbs=get;list;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=runs/status,verbs=update;patch
//+kubebuilder:rbac:groups=tekton.dev,resources=runs/finalizers,verbs=update;patch

// generateBuildRun generates a BuildRun instance owned by the informed Tekton Run object, the
// BuildRun name is randomly generated using the Run's name as base.
func (r *RunReconciler) generateBuildRun(
	ctx context.Context,
	run *tknv1alpha1.Run,
) (*v1alpha1.BuildRun, error) {
	br := v1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: run.GetNamespace(),
			Name:      names.SimpleNameGenerator.GenerateName(fmt.Sprintf("%s-", run.Name)),
			Labels: map[string]string{
				filter.OwnedByTektonRun: run.Name,
			},
		},
		Spec: v1alpha1.BuildRunSpec{
			BuildRef: &v1alpha1.BuildRef{
				APIVersion: &run.Spec.Ref.APIVersion,
				Name:       run.Spec.Ref.Name,
			},
			ParamValues: filter.TektonRunParamsToShipwrightParamValues(run),
			Timeout:     run.Spec.Timeout,
		},
	}
	err := controllerutil.SetControllerReference(run, &br, r.Scheme)
	if err != nil {
		return nil, err
	}
	return &br, nil
}

// reflectBuildRunStatusOnTektonRun reflects the BuildRun status on the Run instance.
func (r *RunReconciler) reflectBuildRunStatusOnTektonRun(
	logger logr.Logger,
	run *tknv1alpha1.Run,
	br *v1alpha1.BuildRun,
) {
	if br.Status.CompletionTime != nil {
		run.Status.CompletionTime = br.Status.CompletionTime
	}
	if run.Status.Conditions == nil {
		run.Status.Conditions = knativev1.Conditions{}
	}

	for _, c := range br.Status.Conditions {
		logger.WithValues(
			"condition-type", c.Type,
			"condition-status", c.Status,
			"condition-reason", c.Reason,
			"condition-message", c.Message,
		).V(0).Info("Reflecting BuildRun status condition on the Tekton Run owner object")

		severity := apis.ConditionSeverityInfo
		if c.Status == corev1.ConditionFalse {
			severity = apis.ConditionSeverityError
		}

		run.Status.SetCondition(&apis.Condition{
			Type:               apis.ConditionType(string(c.Type)),
			Status:             c.Status,
			LastTransitionTime: apis.VolatileTime{Inner: c.LastTransitionTime},
			Reason:             c.Reason,
			Message:            c.Message,
			Severity:           severity,
		})
	}

	if len(run.Status.Conditions) == 0 {
		run.Status.Conditions = []apis.Condition{{
			Type:               apis.ConditionSucceeded,
			Status:             corev1.ConditionUnknown,
			LastTransitionTime: apis.VolatileTime{Inner: metav1.Now()},
		}}
	}
}

// Reconcile reconciles the Custom-Tasks Run instances pointing to Shipwright Builds, by issuing a
// BuildRun and taking advantage of Status.ExtraFields to record the BuildRun namespace/name. When
// the ExtraFields is populated the controller uses the reference to reflect the BuildRun status
// updates on the original Run instance.
func (r *RunReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var run tknv1alpha1.Run
	err := r.Get(ctx, req.NamespacedName, &run)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Unable to fetch Run")
		}
		return RequeueOnError(client.IgnoreNotFound(err))
	}
	originalRun := run.DeepCopy()

	// making sure the current run object status is recorded in the logs
	logger = logger.WithValues("successful", run.IsSuccessful(), "cancelled", run.IsCancelled())

	// when the run instace is marked as done, no further actions needs to take place
	if run.IsDone() {
		logger.V(0).Info("Tekton Run is synchronized, all done!")
		return Done()
	}

	// extracting the meta-information recorded in the Tekton Run status, in this section the name of
	// the BuildRun issued for the object is recorded
	var extraFields filter.ExtraFields
	if err := run.Status.DecodeExtraFields(&extraFields); err != nil {
		logger.V(0).Error(err, "Trying to decode Run Status' ExtraFields")
		return RequeueOnError(err)
	}

	var br = &v1alpha1.BuildRun{}
	// when the status extra-fields is empty, it means the BuildRun instance is not created yet, thus
	// the first step is issueing the instance and later on watching over its status updates
	if extraFields.IsEmpty() {
		if run.IsCancelled() {
			logger.V(0).Info("Tekton Run is cancelled, skipping issuing a BuildRun!")
			return Done()
		}

		if br, err = r.generateBuildRun(ctx, &run); err != nil {
			logger.V(0).Error(err, "Issuing BuildRun returned error")
			return RequeueOnError(err)
		}
		logger = logger.WithValues("buildrun", br.GetName())

		// recording the BuildRun namespace and name using ExtraFields
		extraFields = filter.NewExtraFields(br)
		if err = run.Status.EncodeExtraFields(&extraFields); err != nil {
			logger.V(0).Error(err, "Encoding Tekton's ExtraFields")
			return RequeueOnError(err)
		}
		now := metav1.Now()
		run.Status.StartTime = &now

		// storing the ExtraFields on the Tekton Run instance status
		if err = r.Client.Status().Patch(ctx, &run, client.MergeFrom(originalRun)); err != nil {
			logger.V(0).Error(err, "trying to patch Tekton Run status")
			return RequeueOnError(err)
		}
		logger.V(0).Info("Tekton Run Status ExtraFields updated with BuildRun coordinates")

		if err = r.Client.Create(ctx, br); err != nil {
			logger.V(0).Error(err, "Trying to create a new BuildRun instance")
			return RequeueOnError(err)
		}
		logger.V(0).Info("BuildRun created!")
	} else {
		logger = logger.WithValues("buildrun", extraFields.BuildRunName)
		logger.V(0).Info("Retrieving BuildRun instance...")
		// when the meta-information is populated, we need to extract the BuildRun name and retrieve
		// the object
		if err = r.Client.Get(ctx, extraFields.GetNamespacedName(), br); err != nil {
			logger.V(0).Error(err, "Trying to retrieve BuildRun instance")
			return RequeueOnError(err)
		}

		if run.IsCancelled() && !br.IsCanceled() {
			logger.V(0).Info("Tekton Run instance is cancelled, cancelling the BuildRun too")

			originalBr := br.DeepCopy()
			br.Spec.State = v1alpha1.BuildRunRequestedStatePtr(v1alpha1.BuildRunStateCancel)
			if err = r.Client.Patch(ctx, br, client.MergeFrom(originalBr)); err != nil {
				logger.V(0).Error(err, "trying to patch BuildRun with cancellation state")
				return RequeueOnError(err)
			}
		} else {
			// reflecting BuildRuns' status conditions on the Tekton Run owner instance
			r.reflectBuildRunStatusOnTektonRun(logger, &run, br)

			logger.V(0).Info("Updating Tekton Run instance status...")
			if err = r.Client.Status().Patch(ctx, &run, client.MergeFrom(originalRun)); err != nil {
				logger.V(0).Error(err, "trying to patch Tekton Run status")
				return RequeueOnError(err)
			}
		}
	}

	return Done()
}

// SetupWithManager instantiate this controller using controller runtime manager.
func (r *RunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	return ctrl.NewControllerManagedBy(mgr).
		// watches Tekton Run instances, that's the principal resource for this controller
		For(&tknv1alpha1.Run{}).
		// it also watches over BuildRun instances that are owned by this controller
		Owns(&v1alpha1.BuildRun{}).
		// filtering out objects that aren't ready for reconciliation
		WithEventFilter(predicate.NewPredicateFuncs(filter.RunEventFilterPredicate)).
		// making sure the controller reconciles one instance at the time in order to not create a
		// race between the two resources being watched by this controller, Tekton Run and BuildRun
		// instances
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

// NewRunReconciler instantiate the RunReconciler.
func NewRunReconciler(
	ctrlClient client.Client,
	scheme *runtime.Scheme,
) *RunReconciler {
	return &RunReconciler{
		Client: ctrlClient,
		Scheme: scheme,
	}
}
