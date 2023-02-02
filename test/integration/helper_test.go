package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/filter"
	"github.com/shipwright-io/triggers/test/stubs"
	"k8s.io/apimachinery/pkg/types"

	tknv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
)

var (
	deleteNowOpts  = &client.DeleteOptions{GracePeriodSeconds: &zero}
	timeoutDefault = 30 * time.Second
	zero           = int64(0)

	// gracefulWait amount of time to wait for the apiserver register a new object, and also wait for
	// the controller actions before asserting the insistence of BuildRuns
	gracefulWait = 3 * time.Second
)

type TektonRunAssertFn func(run *tknv1alpha1.Run) error

// assertTektonRun retrieves the Tekton Run instance and execute the informed func with it.
func assertTektonRun(runNamespacedName types.NamespacedName, fn TektonRunAssertFn) error {
	var run tknv1alpha1.Run
	if err := kubeClient.Get(ctx, runNamespacedName, &run); err != nil {
		return err
	}
	return fn(&run)
}

type BuildRunAssertFn func(br *v1alpha1.BuildRun) error

// assertBuildRun retrieves the BuildRun instance and execute the informed func with it.
func assertBuildRun(brNamespacedName types.NamespacedName, fn BuildRunAssertFn) error {
	var br v1alpha1.BuildRun
	if err := kubeClient.Get(ctx, brNamespacedName, &br); err != nil {
		return err
	}
	return fn(&br)
}

// eventuallyWithTimeoutFn wraps the informed function on Eventually() with default timeout.
func eventuallyWithTimeoutFn(fn interface{}) gomegatypes.AsyncAssertion {
	return Eventually(fn).
		WithPolling(time.Second).
		WithTimeout(timeoutDefault)
}

// amountOfBuildRunsFn counts the amount of BuildRuns on "default" namespace.
func amountOfBuildRunsFn() int {
	var brs v1alpha1.BuildRunList
	err := kubeClient.List(ctx, &brs)
	if err != nil {
		return -1
	}
	return len(brs.Items)
}

// deleteAllBuildRuns deletes all BuildRuns using DeleteAllOf ignoring possible not-found errors.
func deleteAllBuildRuns() error {
	return client.IgnoreNotFound(
		kubeClient.DeleteAllOf(ctx, &v1alpha1.BuildRun{},
			client.InNamespace(stubs.Namespace),
			&client.DeleteAllOfOptions{
				DeleteOptions: client.DeleteOptions{GracePeriodSeconds: &zero},
			}),
	)
}

// createAndUpdatePipelineRun create and update the PipelineRun in order to preserve the status
// attribute, which gets removed by envtest[0] during marshaling. This method implements the
// workaround described in the issue #1835[1].
//
//	[0] https://github.com/kubernetes-sigs/controller-runtime/pull/1640
//	[1] https://github.com/kubernetes-sigs/controller-runtime/issues/1835
func createAndUpdatePipelineRun(ctx context.Context, pipelineRun *tknv1beta1.PipelineRun) error {
	status := pipelineRun.Status.DeepCopy()

	var err error
	if err = kubeClient.Create(ctx, pipelineRun); err != nil {
		return err
	}

	var created tknv1beta1.PipelineRun
	if err = kubeClient.Get(ctx, client.ObjectKeyFromObject(pipelineRun), &created); err != nil {
		return err
	}

	created.Status = *status
	return kubeClient.Status().Update(ctx, &created)
}

// createAndUpdateRun creates and updates a Run instance, using the same workaround described on
// createAndUpdatePipelineRun function.
func createAndUpdateRun(ctx context.Context, run *tknv1alpha1.Run) error {
	err := kubeClient.Create(ctx, run)
	if err != nil {
		return err
	}

	var created tknv1alpha1.Run
	if err = kubeClient.Get(ctx, client.ObjectKeyFromObject(run), &created); err != nil {
		return err
	}
	originalRun := created.DeepCopy()
	created.Status = *run.Status.DeepCopy()
	return kubeClient.Status().Patch(ctx, &created, client.MergeFrom(originalRun))
}

// extractBuildRunNamespacedNameFromExtraFields extracts the BuildRun name from the informed Tekton
// Run reference. It asserts the ExtraFields is populated as expected.
func extractBuildRunNamespacedNameFromExtraFields(
	runNamespacedName types.NamespacedName,
) (*types.NamespacedName, error) {
	var run tknv1alpha1.Run
	err := kubeClient.Get(ctx, runNamespacedName, &run)
	if err != nil {
		return nil, err
	}

	if run.Status.ExtraFields.Size() == 0 {
		return nil, fmt.Errorf("Run's ExtraFields is not populated")
	}

	var extraFields filter.ExtraFields
	if err := run.Status.DecodeExtraFields(&extraFields); err != nil {
		return nil, err
	}
	if extraFields.IsEmpty() {
		return nil, fmt.Errorf("attribute ExtraFields is empty")
	}

	namespacedName := extraFields.GetNamespacedName()
	return &namespacedName, nil
}
