// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"os"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/controllers"
	"github.com/shipwright-io/triggers/pkg/inventory"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonapibeta "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

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
	utilruntime.Must(buildapi.AddToScheme(scheme))
	utilruntime.Must(tektonapi.AddToScheme(scheme))
	utilruntime.Must(tektonapibeta.AddToScheme(scheme))
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
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
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

	customRunReconciler := controllers.NewCustomRunReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
	)
	if err = customRunReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to bootstrap controller", "controller", "CustomRun")
		os.Exit(1)
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
