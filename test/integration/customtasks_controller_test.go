package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/filter"
	"github.com/shipwright-io/triggers/test/stubs"

	tknv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Custom-Tasks Controller", Ordered, func() {
	// asserts the CustomTasksReconciler is able to watch Tekton Run instances, and trigger BuildRuns
	// for the pipelines using Shipwright Builds, as well as updating the Tekton Run status as the
	// BuildRun progresses.
	Context("Tekton Run", func() {
		ctx := context.TODO()

		build := stubs.ShipwrightBuild("shipwright.io/triggers", "build")

		// assertTektonRunCustomTask asserts the Tekton Run Status ExtraFields is recorded with
		// coordinates to a existing BuildRun instance.
		assertTektonRunCustomTask := func(runNamespacedName types.NamespacedName) error {
			brNamespacedName, err := extractBuildRunNamespacedNameFromExtraFields(runNamespacedName)
			if err != nil {
				return err
			}

			return assertBuildRun(*brNamespacedName, func(br *v1alpha1.BuildRun) error {
				brOwner := filter.ExtractBuildRunOwner(br)
				if brOwner == nil {
					return fmt.Errorf("BuildRun doesn't have the owner set")
				}

				if runNamespacedName.Namespace != brOwner.Namespace ||
					runNamespacedName.Name != brOwner.Name {
					return fmt.Errorf("BuildRun (%v) is not owned by Tekton Run (%v)",
						runNamespacedName, brOwner)
				}
				return nil
			})
		}

		// assertTektonRunUnknownCondition inspects the informed Tekton Run instance to make sure the
		// condition recorded is "unknown".
		assertTektonRunUnknownCondition := func(run *tknv1alpha1.Run) error {
			if len(run.Status.Conditions) != 1 {
				return fmt.Errorf("Unexpected amount of conditions on Tekton Run status")
			}

			condition := run.Status.Conditions[0]
			if condition.Status != corev1.ConditionUnknown {
				return fmt.Errorf("Condition is not on unknown status")
			}
			if condition.LastTransitionTime.Inner.IsZero() {
				return fmt.Errorf("Condition does not contain last transition time")
			}
			return nil
		}

		// assertBuildRunConditionIsReflectedOnTektonRun inspects the informed Tekton Run object to
		// assert one of its conditions matches the informed last-transition-time.
		assertBuildRunConditionIsReflectedOnTektonRun := func(
			run *tknv1alpha1.Run,
			LastTransitionTime metav1.Time,
		) error {
			for _, c := range run.Status.Conditions {
				if c.LastTransitionTime.Inner.Time.Unix() >= LastTransitionTime.Unix() {
					return nil
				}
			}
			return fmt.Errorf("last-transition-time %q not found in Tekton Run %q",
				LastTransitionTime.String(), run.GetName())
		}

		// testTektonRunTriggersBuildRun test case making sure the informed run instance has
		// triggered BuildRuns accordingly.
		testTektonRunTriggersBuildRun := func(run *tknv1alpha1.Run) types.NamespacedName {
			By("Issuing a new Tekton Run instance referencing Shipwright")
			Expect(createAndUpdateRun(ctx, run)).Should(Succeed())
			runNamespacedName := client.ObjectKeyFromObject(run)

			By("Expecting to have a single BuildRun issued")
			eventuallyWithTimeoutFn(amountOfBuildRunsFn).Should(Equal(1))

			By("Inspecting Tekton Run Status ExtraFields is recorded")
			eventuallyWithTimeoutFn(func() error {
				return assertTektonRunCustomTask(runNamespacedName)
			}).Should(Succeed())

			By("Retrieving the BuildRun recorded on the Tekton Run Status ExtraFields")
			var brNamespacedName *types.NamespacedName
			eventuallyWithTimeoutFn(func() error {
				var err error
				brNamespacedName, err = extractBuildRunNamespacedNameFromExtraFields(runNamespacedName)
				if err != nil {
					return err
				}
				if brNamespacedName == nil {
					return fmt.Errorf("BuildRun namespaced-named is not set")
				}
				return nil
			}).Should(Succeed())

			By("Asserting the Tekton Run instance contains the expected initial condition")
			eventuallyWithTimeoutFn(
				assertTektonRun(runNamespacedName, assertTektonRunUnknownCondition),
			).Should(Succeed())

			return *brNamespacedName
		}

		// updateBuildRunCondition updates the BuildRun with the condition.
		updateBuildRunCondition := func(
			brNamespacedName types.NamespacedName,
			condition v1alpha1.Condition,
		) error {
			return assertBuildRun(brNamespacedName, func(br *v1alpha1.BuildRun) error {
				br.Status = v1alpha1.BuildRunStatus{Conditions: []v1alpha1.Condition{condition}}
				return kubeClient.Status().Update(ctx, br)
			})
		}

		// updateBuildRunStatusConditionLastTransitionTime retrieve and update the BuildRun status
		// condition using the last-transition-time informed.
		updateBuildRunStatusConditionLastTransitionTime := func(
			brNamespacedName types.NamespacedName,
			lastTransitionTime metav1.Time,
		) error {
			return updateBuildRunCondition(brNamespacedName, v1alpha1.Condition{
				Type:               v1alpha1.Succeeded,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: lastTransitionTime,
				Reason:             "reason",
				Message:            "message",
			})
		}

		// updateBuildRunStatusMarkAsFailed marks the BuildRun as failed.
		updateBuildRunStatusMarkAsFailed := func(brNamespacedName types.NamespacedName) error {
			return updateBuildRunCondition(brNamespacedName, v1alpha1.Condition{
				Type:               v1alpha1.Succeeded,
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

		It("Run instance without Shipwright TaskRef, no BuildRuns expected", func() {
			run := stubs.TektonRunStarted("generic", stubs.TektonTaskRefToTekton)
			Expect(createAndUpdateRun(ctx, run)).Should(Succeed())

			time.Sleep(gracefulWait)
			eventuallyWithTimeoutFn(amountOfBuildRunsFn).Should(Equal(0))

			Expect(kubeClient.Delete(ctx, run, deleteNowOpts)).Should(Succeed())
			_ = deleteAllBuildRuns()
		})

		It("Run instance with Shipwright TaskRef, BuildRun expected", func() {
			shipwrightTaskRef := stubs.TektonTaskRefToShipwright(build.GetName())
			run := stubs.TektonRunStarted("invokes-shipwright", shipwrightTaskRef)

			brNamespacedName := testTektonRunTriggersBuildRun(run)

			// updating the status of the BuildRun in the background, simulating what Shipwright
			// Build Controller would do
			lastTransitionTime := metav1.Now()
			go func() {
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

			By("Asserting the Tekton Run instance got updated reflecting BuildRun status")
			eventuallyWithTimeoutFn(func() error {
				return assertTektonRun(
					client.ObjectKeyFromObject(run),
					func(r *tknv1alpha1.Run) error {
						return assertBuildRunConditionIsReflectedOnTektonRun(r, lastTransitionTime)
					},
				)
			}).Should(Succeed())

			Expect(kubeClient.Delete(ctx, run, deleteNowOpts)).Should(Succeed())
			Expect(deleteAllBuildRuns()).Should(Succeed())
		})

		It("Failed BuildRun status is reflected on Tekton Run", func() {
			shipwrightTaskRef := stubs.TektonTaskRefToShipwright(build.GetName())
			run := stubs.TektonRunStarted("invokes-shipwright", shipwrightTaskRef)

			brNamespacedName := testTektonRunTriggersBuildRun(run)

			go func() {
				time.Sleep(gracefulWait)

				By(fmt.Sprintf(
					"Updating the status of the BuildRun %q marking as failed",
					brNamespacedName.String(),
				))
				eventuallyWithTimeoutFn(func() error {
					return updateBuildRunStatusMarkAsFailed(brNamespacedName)
				}).Should(Succeed())
			}()

			By("Asserting the Tekton Run instance got marked as failed")
			eventuallyWithTimeoutFn(func() error {
				return assertTektonRun(
					client.ObjectKeyFromObject(run),
					func(r *tknv1alpha1.Run) error {
						if r.IsSuccessful() {
							return fmt.Errorf("run %q is marked as successful", r.GetName())
						}
						return nil
					},
				)
			}).Should(Succeed())

			time.Sleep(gracefulWait)
			Expect(kubeClient.Delete(ctx, run, deleteNowOpts)).Should(Succeed())
			Expect(deleteAllBuildRuns()).Should(Succeed())
		})

		It("Canceled Tekton Run is reflected on the BuildRun", func() {
			shipwrightTaskRef := stubs.TektonTaskRefToShipwright(build.GetName())
			run := stubs.TektonRunStarted("invokes-shipwright", shipwrightTaskRef)

			brNamespacedName := testTektonRunTriggersBuildRun(run)

			go func() {
				time.Sleep(gracefulWait)

				By(fmt.Sprintf("Canceling the Tekton Run %q", run.GetName()))

				originalRun := run.DeepCopy()
				run.Spec.Status = tknv1alpha1.RunSpecStatusCancelled

				eventuallyWithTimeoutFn(func() error {
					return assertTektonRun(
						client.ObjectKeyFromObject(run),
						func(r *tknv1alpha1.Run) error {
							return kubeClient.Patch(ctx, run, client.MergeFrom(originalRun))
						},
					)
				}).Should(Succeed())
			}()

			By("Asserting the BuildRun is also cancelled")
			eventuallyWithTimeoutFn(func() error {
				return assertBuildRun(
					brNamespacedName,
					func(br *v1alpha1.BuildRun) error {
						if !br.IsCanceled() {
							return fmt.Errorf("the BuildRun %q should be cancelled", br.GetName())
						}
						return nil
					},
				)
			})

			time.Sleep(gracefulWait)
			Expect(kubeClient.Delete(ctx, run, deleteNowOpts)).Should(Succeed())
			Expect(deleteAllBuildRuns()).Should(Succeed())
		})
	})
})
