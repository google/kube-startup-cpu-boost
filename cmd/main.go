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
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	autoscalingv1alpha1 "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	"github.com/google/kube-startup-cpu-boost/internal/boost"
	"github.com/google/kube-startup-cpu-boost/internal/config"
	"github.com/google/kube-startup-cpu-boost/internal/controller"
	"github.com/google/kube-startup-cpu-boost/internal/metrics"
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
	ctrl.SetLogger(config.Logger(cfg.ZapDevelopment, cfg.ZapLogLevel))
	metrics.Register()

	tlsOpts := []func(*tls.Config){}
	if !cfg.HTTP2 {
		setupLog.Info("Disabling HTTP/2")
		tlsOpts = append(tlsOpts, func(cfg *tls.Config) {
			cfg.NextProtos = append(cfg.NextProtos, "http/1.1")
		})
	}

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
	go setupControllers(mgr, boostMgr, cfg, certsReady)

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

func setupControllers(mgr ctrl.Manager, boostMgr boost.Manager, cfg *config.Config, certsReady chan struct{}) {
	setupLog.Info("Waiting for certificate generation to complete")
	<-certsReady
	setupLog.Info("Certificate generation has completed")

	if failedWebhook, err := boostWebhook.Setup(mgr); err != nil {
		setupLog.Error(err, "Unable to create webhook", "webhook", failedWebhook)
		os.Exit(1)
	}
	cpuBoostWebHook := boostWebhook.NewPodCPUBoostWebHook(boostMgr, scheme, cfg.RemoveLimits)
	mgr.GetWebhookServer().Register("/mutate-v1-pod", cpuBoostWebHook)
	boostCtrl := &controller.StartupCPUBoostReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Log:     ctrl.Log.WithName("boost-reconciler"),
		Manager: boostMgr,
	}
	boostMgr.SetStartupCPUBoostReconciler(boostCtrl)
	if err := boostCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StartupCPUBoost")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder
}
