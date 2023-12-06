// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/controllers"
	"github.com/shipwright-io/triggers/pkg/inventory"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonapibeta "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

var (
	cfg        *rest.Config
	testEnv    *envtest.Environment
	kubeClient client.Client

	ctx    context.Context
	cancel context.CancelFunc

	buildInventory *inventory.Inventory
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "bin", "crds")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	started := make(chan struct{})
	go func() {
		cfg, err = testEnv.Start()
		close(started)
	}()
	Eventually(started).WithTimeout(time.Minute).Should(BeClosed())
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = scheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = buildapi.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = tektonapi.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = tektonapibeta.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	kubeClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(kubeClient).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())

	buildInventory = inventory.NewInventory()

	inventoryReconciler := controllers.NewInventoryReconciler(
		mgr.GetClient(), mgr.GetScheme(), buildInventory)

	err = inventoryReconciler.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	pipelineRunReconciler := controllers.NewPipelineRunReconciler(
		mgr.GetClient(), mgr.GetScheme(), buildInventory)

	err = pipelineRunReconciler.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	customRunReconciler := controllers.NewCustomRunReconciler(mgr.GetClient(), mgr.GetScheme())

	err = customRunReconciler.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	time.Sleep(gracefulWait)
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
