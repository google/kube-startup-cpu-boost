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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	"github.com/google/kube-startup-cpu-boost/internal/boost"
	corev1 "k8s.io/api/core/v1"
)

// StartupCPUBoostReconciler reconciles a StartupCPUBoost object
type StartupCPUBoostReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Log     logr.Logger
	Manager boost.Manager
}

//+kubebuilder:rbac:groups=autoscaling.x-k8s.io,resources=startupcpuboosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling.x-k8s.io,resources=startupcpuboosts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=autoscaling.x-k8s.io,resources=startupcpuboosts/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the StartupCPUBoost object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *StartupCPUBoostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var boostObj autoscaling.StartupCPUBoost
	if err := r.Client.Get(ctx, req.NamespacedName, &boostObj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("Reconciling StartupCPUBoost")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StartupCPUBoostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	boostPodHandler := &boostPodHandler{
		manager: r.Manager,
		log:     r.Log,
	}
	lsPredicate, err := predicate.LabelSelectorPredicate(*boostPodHandler.GetPodLabelSelector())
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscaling.StartupCPUBoost{}).
		Watches(&corev1.Pod{},
			boostPodHandler,
			builder.WithPredicates(lsPredicate)).
		WithEventFilter(r).
		Complete(r)
}

func (r *StartupCPUBoostReconciler) Create(e event.CreateEvent) bool {
	boostObj, ok := e.Object.(*autoscaling.StartupCPUBoost)
	if !ok {
		return true
	}
	log := r.Log.WithValues("StartupCPUBoost", klog.KObj(boostObj))
	log.V(2).Info("StartupCPUBoost create event")
	ctx := ctrl.LoggerInto(context.Background(), log)
	if err := r.Manager.AddStartupCPUBoost(ctx, boostObj); err != nil {
		log.Error(err, "Failed to add startupCPUBoost to boost manager")
	}
	return true
}

func (r *StartupCPUBoostReconciler) Delete(e event.DeleteEvent) bool {
	boostObj, ok := e.Object.(*autoscaling.StartupCPUBoost)
	if !ok {
		return true
	}
	log := r.Log.WithValues("StartupCPUBoost", klog.KObj(e.Object))
	log.V(2).Info("StartupCPUBoost delete event")
	r.Manager.DeleteStartupCPUBoost(boostObj)
	return true
}

func (r *StartupCPUBoostReconciler) Update(e event.UpdateEvent) bool {
	log := r.Log.WithValues("StartupCPUBoost", klog.KObj(e.ObjectNew))
	log.V(2).Info("StartupCPUBoost update event")
	return true
}

func (r *StartupCPUBoostReconciler) Generic(e event.GenericEvent) bool {
	log := r.Log.WithValues("StartupCPUBoost", klog.KObj(e.Object))
	log.V(2).Info("StartupCPUBoost generic event")
	return true
}
