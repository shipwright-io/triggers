package main

import (
	"flag"
	"os"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/controllers"
	"github.com/shipwright-io/triggers/pkg/inventory"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	tknv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	//+kubebuilder:scaffold:imports
)

// +kubebuilder:docs-gen:collapse=Imports

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(tknv1beta1.AddToScheme(scheme))
	utilruntime.Must(tknv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	flag.StringVar(
		&metricsAddr,
		"metrics-bind-address",
		":8080",
		"The address the metric endpoint binds to.",
	)
	flag.StringVar(
		&probeAddr,
		"health-probe-bind-address",
		":8081",
		"The address the probe endpoint binds to.",
	)
	flag.BoolVar(
		&enableLeaderElection,
		"leader-elect",
		false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "1337.triggers.shipwright.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	buildInventory := inventory.NewInventory()
	inventoryReconciler := controllers.NewInventoryReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		buildInventory,
	)
	pipelineRunReconciler := controllers.NewPipelineRunReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		buildInventory,
	)

	if err = inventoryReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to bootstrap controller", "controller", "Inventory")
		os.Exit(1)
	}
	if err = pipelineRunReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to bootstrap controller", "controller", "PipelineRun")
		os.Exit(1)
	}

	rm := mgr.GetRESTMapper()

	// Check if the Run resource exists
	if _, err = rm.KindFor(pipeline.RunResource.WithVersion(tknv1alpha1.SchemeGroupVersion.Version)); err != nil {
		// check if the error is the expected error
		if !meta.IsNoMatchError(err) {
			setupLog.Error(err, "unexpected error when checking for existence of Run resource")
			os.Exit(1)
		}
	} else {
		runReconciler := controllers.NewRunReconciler(
			mgr.GetClient(),
			mgr.GetScheme(),
		)
		if err = runReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to bootstrap controller", "controller", "Run")
			os.Exit(1)
		}
	}

	// Check if the CustomRun resource exists
	// Can be removed when https://github.com/tektoncd/pipeline/pull/6199 is available in vendor
	CustomRunResource := schema.GroupResource{
		Group:    pipeline.GroupName,
		Resource: "customruns",
	}
	if _, err = rm.KindFor(CustomRunResource.WithVersion(tknv1beta1.SchemeGroupVersion.Version)); err != nil {
		// check if the error is the expected error
		if !meta.IsNoMatchError(err) {
			setupLog.Error(err, "unexpected error when checking for existence of CustomRun resource")
			os.Exit(1)
		}
	} else {
		customRunReconciler := controllers.NewCustomRunReconciler(
			mgr.GetClient(),
			mgr.GetScheme(),
		)
		if err = customRunReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to bootstrap controller", "controller", "CustomRun")
			os.Exit(1)
		}
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
