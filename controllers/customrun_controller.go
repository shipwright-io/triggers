// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/pkg/filter"

	"github.com/go-logr/logr"
	tektonapibeta "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
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

// CustomRunReconciler watches over Tekton CustomRun instances carrying a Shipwright Build reference,
// that's the approach Tekton takes to Custom-Tasks, which makes possible to utilize third-party
// controller resources.
type CustomRunReconciler struct {
	client.Client                 // kubernetes client
	Scheme        *runtime.Scheme // shared scheme
	Clock                         // local clock instance
}

//+kubebuilder:rbac:groups=shipwright.io,resources=buildruns,verbs=create;get;list;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=customruns,verbs=get;list;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=customruns/status,verbs=update;patch
//+kubebuilder:rbac:groups=tekton.dev,resources=customruns/finalizers,verbs=update;patch

// generateBuildRun generates a BuildRun instance owned by the informed Tekton CustomRun object, the
// BuildRun name is randomly generated using the Run's name as base.
func (r *CustomRunReconciler) generateBuildRun(
	ctx context.Context,
	customRun *tektonapibeta.CustomRun,
) (*buildapi.BuildRun, error) {
	br := buildapi.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: customRun.GetNamespace(),
			Name:      names.SimpleNameGenerator.GenerateName(fmt.Sprintf("%s-", customRun.Name)),
			Labels: map[string]string{
				filter.OwnedByTektonCustomRun: customRun.Name,
			},
		},
		Spec: buildapi.BuildRunSpec{
			Build: buildapi.ReferencedBuild{
				Name: &customRun.Spec.CustomRef.Name,
			},
			ParamValues: filter.TektonCustomRunParamsToShipwrightParamValues(customRun),
			Timeout:     customRun.Spec.Timeout,
		},
	}
	err := controllerutil.SetControllerReference(customRun, &br, r.Scheme)
	if err != nil {
		return nil, err
	}
	return &br, nil
}

// reflectBuildRunStatusOnTektonRun reflects the BuildRun status on the Run instance.
func (r *CustomRunReconciler) reflectBuildRunStatusOnTektonCustomRun(
	logger logr.Logger,
	customRun *tektonapibeta.CustomRun,
	br *buildapi.BuildRun,
) {
	if br.Status.CompletionTime != nil {
		customRun.Status.CompletionTime = br.Status.CompletionTime
	}
	if customRun.Status.Conditions == nil {
		customRun.Status.Conditions = knativev1.Conditions{}
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

		customRun.Status.SetCondition(&apis.Condition{
			Type:               apis.ConditionType(string(c.Type)),
			Status:             c.Status,
			LastTransitionTime: apis.VolatileTime{Inner: c.LastTransitionTime},
			Reason:             c.Reason,
			Message:            c.Message,
			Severity:           severity,
		})
	}

	if len(customRun.Status.Conditions) == 0 {
		customRun.Status.Conditions = []apis.Condition{{
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
func (r *CustomRunReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var customRun tektonapibeta.CustomRun
	err := r.Get(ctx, req.NamespacedName, &customRun)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Unable to fetch Run")
		}
		return RequeueOnError(client.IgnoreNotFound(err))
	}
	originalCustomRun := customRun.DeepCopy()

	// making sure the current run object status is recorded in the logs
	logger = logger.WithValues("successful", customRun.IsSuccessful(), "cancelled", customRun.IsCancelled())

	// when the customRun instance is marked as done, no further actions needs to take place
	if customRun.IsDone() {
		logger.V(0).Info("Tekton Run is synchronized, all done!")
		return Done()
	}

	// extracting the meta-information recorded in the Tekton CustomRun status, in this section the name of
	// the BuildRun issued for the object is recorded
	var extraFields filter.ExtraFields
	if err := customRun.Status.DecodeExtraFields(&extraFields); err != nil {
		logger.V(0).Error(err, "Trying to decode Run Status' ExtraFields")
		return RequeueOnError(err)
	}

	var br = &buildapi.BuildRun{}
	// when the status extra-fields is empty, it means the BuildRun instance is not created yet, thus
	// the first step is issuing the instance and later on watching over its status updates
	if extraFields.IsEmpty() {
		if customRun.IsCancelled() {
			logger.V(0).Info("Tekton CustomRun is cancelled, skipping issuing a BuildRun!")
			return Done()
		}

		if br, err = r.generateBuildRun(ctx, &customRun); err != nil {
			logger.V(0).Error(err, "Issuing BuildRun returned error")
			return RequeueOnError(err)
		}
		logger = logger.WithValues("buildrun", br.GetName())

		// recording the BuildRun namespace and name using ExtraFields
		extraFields = filter.NewExtraFields(br)
		if err = customRun.Status.EncodeExtraFields(&extraFields); err != nil {
			logger.V(0).Error(err, "Encoding Tekton's ExtraFields")
			return RequeueOnError(err)
		}
		now := metav1.Now()
		customRun.Status.StartTime = &now

		// storing the ExtraFields on the Tekton Run instance status
		if err = r.Client.Status().Patch(ctx, &customRun, client.MergeFrom(originalCustomRun)); err != nil {
			logger.V(0).Error(err, "trying to patch Tekton CustomRun status")
			return RequeueOnError(err)
		}
		logger.V(0).Info("Tekton CustomRun Status ExtraFields updated with BuildRun coordinates")

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

		if customRun.IsCancelled() && !br.IsCanceled() {
			logger.V(0).Info("Tekton CustomRun instance is cancelled, cancelling the BuildRun too")

			originalBr := br.DeepCopy()
			br.Spec.State = buildapi.BuildRunRequestedStatePtr(buildapi.BuildRunStateCancel)
			if err = r.Client.Patch(ctx, br, client.MergeFrom(originalBr)); err != nil {
				logger.V(0).Error(err, "trying to patch BuildRun with cancellation state")
				return RequeueOnError(err)
			}
		} else {
			// reflecting BuildRuns' status conditions on the Tekton Run owner instance
			r.reflectBuildRunStatusOnTektonCustomRun(logger, &customRun, br)

			logger.V(0).Info("Updating Tekton CustomRun instance status...")
			if err = r.Client.Status().Patch(ctx, &customRun, client.MergeFrom(originalCustomRun)); err != nil {
				logger.V(0).Error(err, "trying to patch Tekton CustomRun status")
				return RequeueOnError(err)
			}
		}
	}

	return Done()
}

// SetupWithManager instantiate this controller using controller runtime manager.
func (r *CustomRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	return ctrl.NewControllerManagedBy(mgr).
		// watches Tekton Run instances, that's the principal resource for this controller
		For(&tektonapibeta.CustomRun{}).
		// it also watches over BuildRun instances that are owned by this controller
		Owns(&buildapi.BuildRun{}).
		// filtering out objects that aren't ready for reconciliation
		WithEventFilter(predicate.NewPredicateFuncs(filter.CustomRunEventFilterPredicate)).
		// making sure the controller reconciles one instance at the time in order to not create a
		// race between the two resources being watched by this controller, Tekton Run and BuildRun
		// instances
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

// NewCustomRunReconciler instantiate the CustomRunReconciler.
func NewCustomRunReconciler(
	ctrlClient client.Client,
	scheme *runtime.Scheme,
) *CustomRunReconciler {
	return &CustomRunReconciler{
		Client: ctrlClient,
		Scheme: scheme,
	}
}
