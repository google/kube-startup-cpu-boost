// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/tls"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	autoscalingv1alpha1 "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	"github.com/google/kube-startup-cpu-boost/internal/boost"
	"github.com/google/kube-startup-cpu-boost/internal/config"
	"github.com/google/kube-startup-cpu-boost/internal/controller"
	"github.com/google/kube-startup-cpu-boost/internal/util"
	boostWebhook "github.com/google/kube-startup-cpu-boost/internal/webhook"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

var (
	scheme           = runtime.NewScheme()
	setupLog         = ctrl.Log.WithName("setup")
	leaderElectionID = "8fd077db.x-k8s.io"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(autoscalingv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	cfg, err := config.NewEnvConfigProvider(os.LookupEnv).LoadConfig()
	if err != nil {
		setupLog.Error(err, "unable to load configuration")
		os.Exit(1)
	}
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	tlsOpts := []func(*tls.Config){}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
		Port:    9443,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   cfg.MetricsProbeBindAddr,
			SecureServing: cfg.SecureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: cfg.HealthProbeBindAddr,
		LeaderElection:         cfg.LeaderElection,
		LeaderElectionID:       leaderElectionID,
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	certsReady := make(chan struct{})
	if err = util.ManageCerts(mgr, cfg.Namespace, certsReady); err != nil {
		setupLog.Error(err, "Unable to set up certificates")
		os.Exit(1)
	}

	boostMgr := boost.NewManager(mgr.GetClient())
	go setupControllers(mgr, boostMgr, certsReady)

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
	if err := mgr.Add(boostMgr); err != nil {
		setupLog.Error(err, "unable to add boost manager to controller-runtime manager")
	}
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupControllers(mgr ctrl.Manager, boostMgr boost.Manager, certsReady chan struct{}) {
	setupLog.Info("Waiting for certificate generation to complete")
	<-certsReady
	setupLog.Info("Certificate generation has completed")

	if failedWebhook, err := boostWebhook.Setup(mgr); err != nil {
		setupLog.Error(err, "Unable to create webhook", "webhook", failedWebhook)
		os.Exit(1)
	}
	cpuBoostWebHook := boostWebhook.NewPodCPUBoostWebHook(boostMgr, scheme)
	mgr.GetWebhookServer().Register("/mutate-v1-pod", cpuBoostWebHook)

	if err := (&controller.StartupCPUBoostReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Log:     ctrl.Log.WithName("startup-cpu-boost-reconciler"),
		Manager: boostMgr,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StartupCPUBoost")
		os.Exit(1)
	}
	/*
		if err = (&autoscalingv1alpha1.StartupCPUBoost{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "StartupCPUBoost")
			os.Exit(1)
		}
	*/
	//+kubebuilder:scaffold:builder
}
