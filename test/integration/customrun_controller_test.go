// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"context"
	"fmt"
	"time"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/pkg/filter"
	"github.com/shipwright-io/triggers/test/stubs"

	tektonapibeta "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CustomRun Controller", Ordered, Serial, func() {
	// asserts the CustomRunReconciler is able to watch Tekton CustomRun instances, and trigger BuildRuns
	// for the pipelines using Shipwright Builds, as well as updating the Tekton CustomRun status as the
	// BuildRun progresses.
	Context("Tekton CustomRun", func() {
		ctx := context.TODO()

		build := stubs.ShipwrightBuild("shipwright.io/triggers", "build")

		// assertTektonCustomRunCustomTask asserts the Tekton CustomRun Status ExtraFields is recorded with
		// coordinates to a existing BuildRun instance.
		assertTektonCustomRunCustomTask := func(runNamespacedName types.NamespacedName) error {
			brNamespacedName, err := extractBuildRunNamespacedNameFromCustomRunExtraFields(runNamespacedName)
			if err != nil {
				return err
			}

			return assertBuildRun(*brNamespacedName, func(br *buildapi.BuildRun) error {
				brOwner := filter.ExtractBuildRunCustomRunOwner(br)
				if brOwner == nil {
					return fmt.Errorf("BuildRun doesn't have the owner set")
				}

				if runNamespacedName.Namespace != brOwner.Namespace ||
					runNamespacedName.Name != brOwner.Name {
					return fmt.Errorf("BuildRun (%v) is not owned by Tekton CustomRun (%v)",
						runNamespacedName, brOwner)
				}
				return nil
			})
		}

		// assertTektonCustomRunUnknownCondition inspects the informed Tekton CustomRun instance to make sure the
		// condition recorded is "unknown".
		assertTektonCustomRunUnknownCondition := func(customRun *tektonapibeta.CustomRun) error {
			if len(customRun.Status.Conditions) != 1 {
				return fmt.Errorf("Unexpected amount of conditions on Tekton CustomRun status")
			}

			condition := customRun.Status.Conditions[0]
			if condition.Status != corev1.ConditionUnknown {
				return fmt.Errorf("Condition is not on unknown status")
			}
			if condition.LastTransitionTime.Inner.IsZero() {
				return fmt.Errorf("Condition does not contain last transition time")
			}
			return nil
		}

		// assertBuildRunConditionIsReflectedOnTektonCustomRun inspects the informed Tekton CustomRun object to
		// assert one of its conditions matches the informed last-transition-time.
		assertBuildRunConditionIsReflectedOnTektonCustomRun := func(
			customRun *tektonapibeta.CustomRun,
			LastTransitionTime metav1.Time,
		) error {
			for _, c := range customRun.Status.Conditions {
				if c.LastTransitionTime.Inner.Time.Unix() >= LastTransitionTime.Unix() {
					return nil
				}
			}
			return fmt.Errorf("last-transition-time %q not found in Tekton CustomRun %q",
				LastTransitionTime.String(), customRun.GetName())
		}

		// testTektonCustomRunTriggersBuildRun test case making sure the informed run instance has
		// triggered BuildRuns accordingly.
		testTektonCustomRunTriggersBuildRun := func(customRun *tektonapibeta.CustomRun) types.NamespacedName {
			By("Issuing a new Tekton CustomRun instance referencing Shipwright")
			Expect(createAndUpdateCustomRun(ctx, customRun)).Should(Succeed())
			runNamespacedName := client.ObjectKeyFromObject(customRun)

			By("Expecting to have a single BuildRun issued")
			eventuallyWithTimeoutFn(amountOfBuildRunsFn).Should(Equal(1))

			By("Inspecting Tekton CustomRun Status ExtraFields is recorded")
			eventuallyWithTimeoutFn(func() error {
				return assertTektonCustomRunCustomTask(runNamespacedName)
			}).Should(Succeed())

			By("Retrieving the BuildRun recorded on the Tekton CustomRun Status ExtraFields")
			var brNamespacedName *types.NamespacedName
			eventuallyWithTimeoutFn(func() error {
				var err error
				brNamespacedName, err = extractBuildRunNamespacedNameFromCustomRunExtraFields(runNamespacedName)
				if err != nil {
					return err
				}
				if brNamespacedName == nil {
					return fmt.Errorf("BuildRun namespaced-named is not set")
				}
				return nil
			}).Should(Succeed())

			By("Asserting the Tekton CustomRun instance contains the expected initial condition")
			eventuallyWithTimeoutFn(
				assertTektonCustomRun(runNamespacedName, assertTektonCustomRunUnknownCondition),
			).Should(Succeed())

			return *brNamespacedName
		}

		// updateBuildRunCondition updates the BuildRun with the condition.
		updateBuildRunCondition := func(
			brNamespacedName types.NamespacedName,
			condition buildapi.Condition,
		) error {
			return assertBuildRun(brNamespacedName, func(br *buildapi.BuildRun) error {
				br.Status = buildapi.BuildRunStatus{Conditions: []buildapi.Condition{condition}}
				return kubeClient.Status().Update(ctx, br)
			})
		}

		// updateBuildRunStatusConditionLastTransitionTime retrieve and update the BuildRun status
		// condition using the last-transition-time informed.
		updateBuildRunStatusConditionLastTransitionTime := func(
			brNamespacedName types.NamespacedName,
			lastTransitionTime metav1.Time,
		) error {
			return updateBuildRunCondition(brNamespacedName, buildapi.Condition{
				Type:               buildapi.Succeeded,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: lastTransitionTime,
				Reason:             "reason",
				Message:            "message",
			})
		}

		// updateBuildRunStatusMarkAsFailed marks the BuildRun as failed.
		updateBuildRunStatusMarkAsFailed := func(brNamespacedName types.NamespacedName) error {
			return updateBuildRunCondition(brNamespacedName, buildapi.Condition{
				Type:               buildapi.Succeeded,
				Status:             corev1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "failed",
				Message:            "failed",
			})
		}

		BeforeAll(func() {
			Expect(deleteAllBuildRuns()).Should(Succeed())
			Expect(kubeClient.Create(ctx, build)).Should(Succeed())
			time.Sleep(gracefulWait)
		})

		AfterAll(func() {
			Expect(kubeClient.Delete(ctx, build, deleteNowOpts)).Should(Succeed())
		})

		It("CustomRun instance without Shipwright TaskRef, no BuildRuns expected", func() {
			customRun := stubs.TektonCustomRunStarted("generic", stubs.TektonTaskRefToTekton)
			Expect(createAndUpdateCustomRun(ctx, customRun)).Should(Succeed())

			time.Sleep(gracefulWait)
			eventuallyWithTimeoutFn(amountOfBuildRunsFn).Should(Equal(0))

			Expect(kubeClient.Delete(ctx, customRun, deleteNowOpts)).Should(Succeed())
			_ = deleteAllBuildRuns()
		})

		It("CustomRun instance with Shipwright TaskRef, BuildRun expected", func() {
			shipwrightTaskRef := stubs.TektonTaskRefToShipwright(build.GetName())
			customRun := stubs.TektonCustomRunStarted("invokes-shipwright-customrun", shipwrightTaskRef)

			brNamespacedName := testTektonCustomRunTriggersBuildRun(customRun)

			// updating the status of the BuildRun in the background, simulating what Shipwright
			// Build Controller would do
			lastTransitionTime := metav1.Now()
			go func() {
				defer GinkgoRecover()

				time.Sleep(gracefulWait)

				By(fmt.Sprintf(
					"Updating the status of the BuildRun created %q with last-transition-time %q",
					brNamespacedName.String(),
					lastTransitionTime,
				))
				eventuallyWithTimeoutFn(func() error {
					return updateBuildRunStatusConditionLastTransitionTime(
						brNamespacedName, lastTransitionTime)
				}).Should(Succeed())
			}()

			By("Asserting the Tekton CustomRun instance got updated reflecting BuildRun status")
			eventuallyWithTimeoutFn(func() error {
				return assertTektonCustomRun(
					client.ObjectKeyFromObject(customRun),
					func(r *tektonapibeta.CustomRun) error {
						return assertBuildRunConditionIsReflectedOnTektonCustomRun(r, lastTransitionTime)
					},
				)
			}).Should(Succeed())

			Expect(kubeClient.Delete(ctx, customRun, deleteNowOpts)).Should(Succeed())
			Expect(deleteAllBuildRuns()).Should(Succeed())
		})

		It("Failed BuildRun status is reflected on Tekton CustomRun", func() {
			shipwrightTaskRef := stubs.TektonTaskRefToShipwright(build.GetName())
			customRun := stubs.TektonCustomRunStarted("invokes-shipwright-customrun", shipwrightTaskRef)

			brNamespacedName := testTektonCustomRunTriggersBuildRun(customRun)

			go func() {
				defer GinkgoRecover()

				time.Sleep(gracefulWait)

				By(fmt.Sprintf(
					"Updating the status of the BuildRun %q marking as failed",
					brNamespacedName.String(),
				))
				eventuallyWithTimeoutFn(func() error {
					return updateBuildRunStatusMarkAsFailed(brNamespacedName)
				}).Should(Succeed())
			}()

			By("Asserting the Tekton CustomRun instance got marked as failed")
			eventuallyWithTimeoutFn(func() error {
				return assertTektonCustomRun(
					client.ObjectKeyFromObject(customRun),
					func(r *tektonapibeta.CustomRun) error {
						if r.IsSuccessful() {
							return fmt.Errorf("run %q is marked as successful", r.GetName())
						}
						return nil
					},
				)
			}).Should(Succeed())

			time.Sleep(gracefulWait)
			Expect(kubeClient.Delete(ctx, customRun, deleteNowOpts)).Should(Succeed())
			Expect(deleteAllBuildRuns()).Should(Succeed())
		})

		It("Canceled Tekton CustomRun is reflected on the BuildRun", func() {
			shipwrightTaskRef := stubs.TektonTaskRefToShipwright(build.GetName())
			customRun := stubs.TektonCustomRunStarted("invokes-shipwright-customrun", shipwrightTaskRef)

			brNamespacedName := testTektonCustomRunTriggersBuildRun(customRun)

			go func() {
				defer GinkgoRecover()

				time.Sleep(gracefulWait)

				By(fmt.Sprintf("Canceling the Tekton CustomRun %q", customRun.GetName()))

				originalRun := customRun.DeepCopy()
				customRun.Spec.Status = tektonapibeta.CustomRunSpecStatusCancelled

				eventuallyWithTimeoutFn(func() error {
					return assertTektonCustomRun(
						client.ObjectKeyFromObject(customRun),
						func(r *tektonapibeta.CustomRun) error {
							return kubeClient.Patch(ctx, customRun, client.MergeFrom(originalRun))
						},
					)
				}).Should(Succeed())
			}()

			By("Asserting the BuildRun is also cancelled")
			eventuallyWithTimeoutFn(func() error {
				return assertBuildRun(
					brNamespacedName,
					func(br *buildapi.BuildRun) error {
						if !br.IsCanceled() {
							return fmt.Errorf("the BuildRun %q should be cancelled", br.GetName())
						}
						return nil
					},
				)
			})

			time.Sleep(gracefulWait)
			Expect(kubeClient.Delete(ctx, customRun, deleteNowOpts)).Should(Succeed())
			Expect(deleteAllBuildRuns()).Should(Succeed())
		})
	})
})
