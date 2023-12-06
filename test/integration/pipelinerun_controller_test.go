// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"encoding/json"
	"time"

	"github.com/shipwright-io/triggers/pkg/filter"
	"github.com/shipwright-io/triggers/test/stubs"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PipelineRun Controller", Ordered, func() {
	// asserts the PIpelineRun controller which is intercepting PipelineRun instances to trigger
	// BuildRuns, when a configured trigger matches the incoming object. The test scenarios also
	// asserts the controller skips Custom-Tasks and incomplete PipelineRun instances
	Context("PipelineRun instances will trigger BuildRuns", func() {
		buildWithPipelineTrigger := stubs.ShipwrightBuildWithTriggers(
			"shipwright.io/triggers",
			"build-with-pipeline-trigger",
			stubs.TriggerWhenPipelineSucceeded,
		)

		BeforeAll(func() {
			Expect(deleteAllBuildRuns()).Should(Succeed())
			Expect(kubeClient.Create(ctx, buildWithPipelineTrigger)).Should(Succeed())
			time.Sleep(gracefulWait)
		})

		AfterAll(func() {
			Expect(kubeClient.Delete(ctx, buildWithPipelineTrigger, deleteNowOpts)).
				Should(Succeed())
		})

		It("PipelineRun without status recorded won't trigger a BuildRun", func() {
			pipelineRun := stubs.TektonPipelineRun(stubs.PipelineNameInTrigger)
			Expect(createAndUpdatePipelineRun(ctx, &pipelineRun)).Should(Succeed())

			time.Sleep(gracefulWait)
			eventuallyWithTimeoutFn(amountOfBuildRunsFn).Should(Equal(0))

			Expect(kubeClient.Delete(ctx, &pipelineRun, deleteNowOpts)).Should(Succeed())
		})

		It("Custom-Task PipelineRun won't trigger a BuildRun", func() {
			pipelineRun := stubs.TektonPipelineRunSucceeded(stubs.PipelineNameInTrigger)
			pipelineRun.Status.PipelineSpec = stubs.TektonPipelineRunStatusCustomTaskShipwright
			Expect(createAndUpdatePipelineRun(ctx, &pipelineRun)).Should(Succeed())

			time.Sleep(gracefulWait)
			eventuallyWithTimeoutFn(amountOfBuildRunsFn).Should(Equal(0))

			Expect(kubeClient.Delete(ctx, &pipelineRun, deleteNowOpts)).Should(Succeed())
		})

		It("PipelineRun already processed won't trigger a BuildRun", func() {
			pipelineRun := stubs.TektonPipelineRunSucceeded(stubs.PipelineNameInTrigger)

			objectRef, err := filter.PipelineRunToObjectRef(ctx, time.Now(), &pipelineRun)
			Expect(err).To(Succeed())

			triggeredBuilds := []filter.TriggeredBuild{{
				BuildName: buildWithPipelineTrigger.GetName(),
				ObjectRef: objectRef,
			}}

			annotationBytes, err := json.Marshal(triggeredBuilds)
			Expect(err).To(Succeed())

			pipelineRun.SetAnnotations(map[string]string{
				filter.TektonPipelineRunName:            pipelineRun.GetName(),
				filter.TektonPipelineRunTriggeredBuilds: string(annotationBytes),
			})
			Expect(createAndUpdatePipelineRun(ctx, &pipelineRun)).Should(Succeed())

			time.Sleep(gracefulWait)
			eventuallyWithTimeoutFn(amountOfBuildRunsFn).Should(Equal(0))

			Expect(kubeClient.Delete(ctx, &pipelineRun, deleteNowOpts)).Should(Succeed())
		})

		It("PipelineRun triggers a BuildRun", func() {
			pipelineRun := stubs.TektonPipelineRunSucceeded(stubs.PipelineNameInTrigger)
			Expect(createAndUpdatePipelineRun(ctx, &pipelineRun)).Should(Succeed())

			eventuallyWithTimeoutFn(amountOfBuildRunsFn).Should(Equal(1))

			eventuallyWithTimeoutFn(func() bool {
				var pr tektonapi.PipelineRun
				if err := kubeClient.Get(ctx, pipelineRun.GetNamespacedName(), &pr); err != nil {
					return false
				}

				objectRef, err := filter.PipelineRunToObjectRef(ctx, time.Now(), &pr)
				if err != nil {
					return false
				}
				triggeredBuilds, err := filter.PipelineRunExtractTriggeredBuildsSlice(&pr)
				if err != nil {
					return false
				}
				return filter.TriggereBuildsContainsObjectRef(
					triggeredBuilds,
					[]string{buildWithPipelineTrigger.GetName()},
					objectRef,
				)
			}).Should(BeTrue())

			Expect(kubeClient.Delete(ctx, &pipelineRun, deleteNowOpts)).Should(Succeed())
		})
	})
})
