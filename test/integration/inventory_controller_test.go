// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"context"
	"time"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/test/stubs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Build Inventory Controller", Ordered, func() {
	// asserts the Inventory instance is being fed by the controller, therefore the Build objects
	// created on the cluster will be stored accordingly, likewise when updated or removed it will be
	// reflected in the Inventory
	Context("Inventory reflect Build instances in the cluster", func() {
		ctx := context.Background()

		buildWithGitHubTrigger := stubs.ShipwrightBuildWithTriggers(
			"shipwright.io/triggers",
			"build-with-github-trigger",
			stubs.TriggerWhenPushToMain,
		)
		// searchForBuildWithGitHubTriggerFn search for the Build with GitHub trigger, returns the
		// amount of instances stored in the Inventory.
		searchForBuildWithGitHubTriggerFn := func() int {
			return len(buildInventory.SearchForGit(
				buildapi.GitHubWebHookTrigger,
				buildWithGitHubTrigger.Spec.Source.Git.URL,
				stubs.Branch,
			))
		}

		buildWithPipelineTrigger := stubs.ShipwrightBuildWithTriggers(
			"shipwright.io/triggers",
			"build-with-pipeline-trigger",
			stubs.TriggerWhenPipelineSucceeded,
		)
		// searchForBuildWithPipelineTriggerFn search for the Build with Pipeline trigger, returns
		// the amount of instances found in the Inventory.
		searchForBuildWithPipelineTriggerFn := func() int {
			return len(buildInventory.SearchForObjectRef(
				buildapi.PipelineTrigger,
				buildWithPipelineTrigger.Spec.Trigger.When[0].ObjectRef,
			))
		}

		AfterAll(func() {
			_ = kubeClient.Delete(ctx, buildWithPipelineTrigger, deleteNowOpts)
			time.Sleep(gracefulWait)
		})

		It("Should add a Build instances (with triggers)", func() {
			Expect(kubeClient.Create(ctx, buildWithGitHubTrigger)).Should(Succeed())
			Expect(kubeClient.Create(ctx, buildWithPipelineTrigger)).Should(Succeed())
		})

		It("Should find the Build (GitHub) in the Inventory", func() {
			eventuallyWithTimeoutFn(searchForBuildWithGitHubTriggerFn).Should(Equal(1))
		})

		It("Should find the Build (GitHub) in the Inventory", func() {
			eventuallyWithTimeoutFn(searchForBuildWithPipelineTriggerFn).Should(Equal(1))
		})

		It("Should remove the Build instances", func() {
			Expect(kubeClient.Delete(ctx, buildWithGitHubTrigger, deleteNowOpts)).Should(Succeed())
			Expect(kubeClient.Delete(ctx, buildWithPipelineTrigger, deleteNowOpts)).Should(Succeed())
		})

		It("Should not find the Build (GitHub) in the Inventory", func() {
			eventuallyWithTimeoutFn(searchForBuildWithGitHubTriggerFn).Should(Equal(0))
		})

		It("Should not find the Build (GitHub) in the Inventory", func() {
			eventuallyWithTimeoutFn(searchForBuildWithPipelineTriggerFn).Should(Equal(0))
		})
	})
})
