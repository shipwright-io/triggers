package controllers

import (
	"context"
	"fmt"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/constants"
	"github.com/shipwright-io/triggers/pkg/filter"
	"github.com/shipwright-io/triggers/pkg/inventory"

	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PipelineRunReconciler reconciles PipelineRun objects that may have triggers configured to generate
// a BuildRun based on the Pipeline state.
type PipelineRunReconciler struct {
	client.Client                 // kubernetes client
	Scheme        *runtime.Scheme // shared scheme
	Clock                         // local clock instance

	buildInventory *inventory.Inventory // local build triggers database
}

//+kubebuilder:rbac:groups=shipwright.io,resources=builds,verbs=get;list;watch
//+kubebuilder:rbac:groups=shipwright.io,resources=buildruns,verbs=create;get;list;update;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;update;patch;watch

// createBuildRun handles the actual BuildRun creation, uses the informed PipelineRun instance to
// establish ownership. Only returns the created object name and error.
func (r *PipelineRunReconciler) createBuildRun(
	ctx context.Context,
	pipelineRun *tknv1beta1.PipelineRun,
	buildName string,
) (string, error) {
	br := v1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    pipelineRun.GetNamespace(),
			GenerateName: fmt.Sprintf("%s-", buildName),
			Annotations: map[string]string{
				filter.OwnedByTektonPipelineRun: pipelineRun.GetName(),
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: constants.TektonAPIv1beta1,
				Kind:       "PipelineRun",
				Name:       pipelineRun.GetName(),
				UID:        pipelineRun.GetUID(),
			}},
		},
		Spec: v1alpha1.BuildRunSpec{
			BuildRef: &v1alpha1.BuildRef{
				Name: buildName,
			},
		},
	}
	if err := r.Client.Create(ctx, &br); err != nil {
		return "", err
	}
	return br.GetName(), nil
}

// issueBuildRunsForPipelineRun create the BuildRun instances for the informed objects, and updates
// the PipelineRun annotations to documented the created BuildRuns.
func (r *PipelineRunReconciler) issueBuildRunsForPipelineRun(
	ctx context.Context,
	pipelineRun *tknv1beta1.PipelineRun,
	buildNames []string,
) ([]string, error) {
	var created []string
	for _, buildName := range buildNames {
		buildRunName, err := r.createBuildRun(ctx, pipelineRun, buildName)
		if err != nil {
			return created, err
		}
		created = append(created, buildRunName)
	}
	return created, nil
}

// Reconcile inspects the PipelineRun to extract the query parameters for the Build inventory search,
// and at the end creates the BuildRun instance(s). Before firing the BuildRuns it inspects the
// PipelineRun to assert the object is being referred by triggers and it's not part of a Custom-Task
// Pipeline.
func (r *PipelineRunReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var pipelineRun tknv1beta1.PipelineRun
	if err := r.Get(ctx, req.NamespacedName, &pipelineRun); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Unable to fetch PipelineRun")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// making sure a copy of the original object is available to patch the resource later on
	originalPipelineRun := pipelineRun.DeepCopy()

	// creating a objectRef based on the informed PipelineRun, the instance is informed to the
	// inventory query interface to list Shipwright Builds that should be triggered
	objectRef, err := filter.PipelineRunToObjectRef(ctx, r.Clock.Now(), &pipelineRun)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.V(0).Info(
		"Searching for Builds matching criteria",
		"ref-name", objectRef.Name,
		"ref-status", objectRef.Status,
		"ref-selector", objectRef.Selector,
	)

	// search for Builds with Pipeline triggers matching current ObjectRef criteria
	buildsToBeIssued := r.buildInventory.SearchForObjectRef(v1alpha1.PipelineTrigger, objectRef)
	if len(buildsToBeIssued) == 0 {
		return ctrl.Result{}, nil
	}

	buildNames := inventory.ExtractBuildNames(buildsToBeIssued...)
	logger.V(0).Info("Build names in the Inventory matching criteria", "build-names", buildNames)

	// during pipeline re-run a new PipelineRun is issued based on a existing object copying over all
	// the elements, including annotations. To allow re-runs we annotate the current object's name
	// and only check previously triggered builds when the name matches
	var triggeredBuilds = []filter.TriggeredBuild{}
	if filter.PipelineRunAnnotatedNameMatchesObject(&pipelineRun) {
		// extracting existing triggered-builds from the annotation, information needed to detect if
		// the BuildRuns have already beeing issued for the PipelineRun
		triggeredBuilds, err = filter.PipelineRunExtractTriggeredBuildsSlice(&pipelineRun)
		if err != nil {
			logger.V(0).Error(err, "parsing triggered-builds annotation")
			// in case of errors an empty slice takes place, may incur the side effect of issuing
			// duplicated BuildRuns
			triggeredBuilds = []filter.TriggeredBuild{}
		}

		// filtering out the instances that have already been processed, the annotation extracted
		// shows  which build names and the objectRef employed
		if filter.TriggereBuildsContainsObjectRef(triggeredBuilds, buildNames, objectRef) {
			logger.V(0).Info("BuildRuns for PipelineRun have already been issued!")
			return ctrl.Result{}, nil
		}
	} else {
		logger.V(0).Info("PipelineRun annotated name does not match current object!")
	}
	logger.V(0).Info("PipelineRun previously triggered builds", "triggered-builds", triggeredBuilds)

	// firing the BuildRun instances for the informed Builds
	buildRunsIssued, err := r.issueBuildRunsForPipelineRun(ctx, &pipelineRun, buildNames)
	if err != nil {
		logger.V(0).Error(err, "trying to issue BuildRun instances", "buildruns", buildRunsIssued)
		return ctrl.Result{}, err
	}
	logger.V(0).Info("BuildRuns issued", "buildruns", buildRunsIssued)

	// updating annotation appending the current state which triggered BuildRuns instances, this
	// annotation is later on checked to skip the conditions that already triggered builds
	if err = filter.PipelineRunAppendTriggeredBuildsAnnotation(
		&pipelineRun,
		triggeredBuilds,
		buildNames,
		objectRef,
	); err != nil {
		logger.V(0).Error(err, "trying to updated triggered-builds annotation")
		return ctrl.Result{}, err
	}

	// updating label registering all BuildRuns issued
	filter.AppendIssuedBuildRunsLabel(&pipelineRun, buildRunsIssued)
	// annotating object's current name
	filter.PipelineRunAnnotateName(&pipelineRun)

	// patching the PipelineRun to reflect labels and annotations needed on the object
	if err = r.Client.Patch(ctx, &pipelineRun, client.MergeFrom(originalPipelineRun)); err != nil {
		logger.V(0).Error(err, "trying to update PipelineRun metadata")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager uses the manager to watch over PipelineRuns.
func (r *PipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&tknv1beta1.PipelineRun{}).
		WithEventFilter(predicate.NewPredicateFuncs(filter.EventFilterPredicate)).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				return !e.DeleteStateUnknown
			},
		}).
		Complete(r)
}

// NewPipelineRunReconciler instantiate the PipelineRunReconciler.
func NewPipelineRunReconciler(
	ctrlClient client.Client,
	scheme *runtime.Scheme,
	buildInventory *inventory.Inventory,
) *PipelineRunReconciler {
	return &PipelineRunReconciler{
		Client:         ctrlClient,
		Scheme:         scheme,
		buildInventory: buildInventory,
	}
}
