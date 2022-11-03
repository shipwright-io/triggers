package integration

import (
	"context"
	"time"

	"github.com/onsi/gomega/types"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/gomega"
)

var (
	timeoutDefault = 30 * time.Second
	zero           = int64(0)
	deleteNowOpts  = &client.DeleteOptions{GracePeriodSeconds: &zero}
)

// eventuallyWithTimeoutFn wraps the informed function on Eventually() with default timeout.
func eventuallyWithTimeoutFn(fn func() int) types.AsyncAssertion {
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

// createAndUpdatePipelineRun create and update the PipelineRun in order to preserve the status
// attribute, which gets removed by envtest[0] during marshaling. This method implements the
// workaround described in the issue #1835[1].
//
//	[0] https://github.com/kubernetes-sigs/controller-runtime/pull/1640
//	[1] https://github.com/kubernetes-sigs/controller-runtime/issues/1835
func createAndUpdatePipelineRun(ctx context.Context, pipelineRun tknv1beta1.PipelineRun) error {
	status := pipelineRun.Status.DeepCopy()

	var err error
	if err = kubeClient.Create(ctx, &pipelineRun); err != nil {
		return err
	}

	var created tknv1beta1.PipelineRun
	key := pipelineRun.GetNamespacedName()
	if err = kubeClient.Get(ctx, key, &created); err != nil {
		return err
	}

	created.Status = *status
	return kubeClient.Status().Update(ctx, &created)
}
