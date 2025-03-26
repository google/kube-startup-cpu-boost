// Copyright 2024 Google LLC
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
	"context"
	"crypto/tls"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/version"
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

const (
	InPlacePodVerticalScalingFeatureGateName = "InPlacePodVerticalScaling"
)

var (
	scheme           = runtime.NewScheme()
	setupLog         = ctrl.Log.WithName("setup")
	leaderElectionID = "8fd077db.x-k8s.io"
)

//+kubebuilder:rbac:urls="/metrics",verbs=get

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
	restConfig := ctrl.GetConfigOrDie()

	clusterInfo, err := util.NewClusterInfoFromConfig(context.Background(), restConfig)
	if err != nil {
		setupLog.Error(err, "failed to create cluster info provider")
		os.Exit(1)
	}

	versionInfo, err := clusterInfo.GetClusterVersion()
	if err != nil {
		setupLog.Error(err, "failed to get cluster version, continuing...")
	} else {
		setupLog.Info("cluster info", "version", versionInfo.GitVersion)
	}

	if cfg.ValidateFeatureEnabled {
		validateFeatureGates(clusterInfo)
	}

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

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
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
	go setupControllers(mgr, boostMgr, cfg, versionInfo, certsReady)

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

func setupControllers(mgr ctrl.Manager, boostMgr boost.Manager, cfg *config.Config, serverVersion *version.Info,
	certsReady chan struct{}) {
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
	if err := boostCtrl.SetupWithManager(mgr, serverVersion); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StartupCPUBoost")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder
}

func validateFeatureGates(clusterInfo util.ClusterInfo) {
	setupLog.Info("validating required feature gates")
	featureGates, err := clusterInfo.GetFeatureGates()
	if err == nil {
		if !featureGates.IsEnabledAnyStage(InPlacePodVerticalScalingFeatureGateName) {
			setupLog.Error(
				fmt.Errorf("%s is not enabled at any stage", InPlacePodVerticalScalingFeatureGateName),
				"required feature gates are not enabled")
			os.Exit(1)
		}
	} else {
		setupLog.Error(err, "failed to validate required feature gates, continuing...")
	}
}
