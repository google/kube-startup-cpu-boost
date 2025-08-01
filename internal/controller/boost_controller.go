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

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	BoostActiveConditionTrueReason   = "Ready"
	BoostActiveConditionTrueMessage  = "Can boost new containers"
	BoostActiveConditionFalseReason  = "NotFound"
	BoostActiveConditionFalseMessage = "StartupCPUBoost not found"
	WantedServerVersionForNewRevert  = "v1.32.0"
)

// StartupCPUBoostReconciler reconciles a StartupCPUBoost object
type StartupCPUBoostReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	Log              logr.Logger
	Manager          boost.Manager
	LegacyRevertMode bool
}

//+kubebuilder:rbac:groups=autoscaling.x-k8s.io,resources=startupcpuboosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling.x-k8s.io,resources=startupcpuboosts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=autoscaling.x-k8s.io,resources=startupcpuboosts/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;update;patch;watch
//+kubebuilder:rbac:groups="",resources=pods/resize,verbs=patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *StartupCPUBoostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result,
	error) {
	var boostObj autoscaling.StartupCPUBoost
	var err error
	if err = r.Client.Get(ctx, req.NamespacedName, &boostObj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log := r.Log.WithValues("name", boostObj.Name, "namespace", boostObj.Namespace)
	newBoostObj := boostObj.DeepCopy()
	activeCondition := metav1.Condition{
		Type:    "Active",
		Status:  metav1.ConditionFalse,
		Reason:  BoostActiveConditionFalseReason,
		Message: BoostActiveConditionFalseMessage,
	}
	boost, ok := r.Manager.GetRegularCPUBoost(ctx, boostObj.Name, boostObj.Namespace)
	if ok {
		log.V(5).Info("found boost in a manager")
		stats := boost.Stats()
		activeCondition.Status = metav1.ConditionTrue
		activeCondition.Reason = BoostActiveConditionTrueReason
		activeCondition.Message = BoostActiveConditionTrueMessage
		newBoostObj.Status.ActiveContainerBoosts = int32(stats.ActiveContainerBoosts)
		newBoostObj.Status.TotalContainerBoosts = int32(stats.TotalContainerBoosts)
	}
	meta.SetStatusCondition(&newBoostObj.Status.Conditions, activeCondition)
	if !equality.Semantic.DeepEqual(newBoostObj.Status, boostObj.Status) {
		log.V(5).Info("updating boost status")
		err = r.Client.Status().Update(ctx, newBoostObj)
	}
	if err != nil {
		if apierrors.IsConflict(err) {
			log.V(5).Info("boost status update conflict, requeueing")
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "boost status update error")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StartupCPUBoostReconciler) SetupWithManager(mgr ctrl.Manager,
	serverVersion string) error {
	boostPodHandler := NewBoostPodHandler(r.Manager, ctrl.Log.WithName("pod-handler"))
	lsPredicate, err := predicate.LabelSelectorPredicate(*boostPodHandler.GetPodLabelSelector())
	if err != nil {
		return err
	}
	r.LegacyRevertMode = shouldUseLegacyRevertMode(serverVersion)
	ctrl.Log.WithName("boost-controller-setup").WithValues("legacyRevertMode", r.LegacyRevertMode).
		V(5).Info("setting legacy revert mode")
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
	log := r.Log.WithValues("name", boostObj.Name, "namespace", boostObj.Namespace)
	log.V(5).Info("handling boost create event")
	ctx := ctrl.LoggerInto(context.Background(), log)
	boost, err := boost.NewStartupCPUBoost(r.Client, boostObj, r.LegacyRevertMode)
	if err != nil {
		log.Error(err, "boost creation error")
	}
	if err := r.Manager.AddRegularCPUBoost(ctx, boost); err != nil {
		log.Error(err, "boost registration error")
	}
	return true
}

func (r *StartupCPUBoostReconciler) Delete(e event.DeleteEvent) bool {
	boostObj, ok := e.Object.(*autoscaling.StartupCPUBoost)
	if !ok {
		return true
	}
	log := r.Log.WithValues("name", boostObj.Name, "namespace", boostObj.Namespace)
	log.V(5).Info("handling boost delete event")
	ctx := ctrl.LoggerInto(context.Background(), log)
	r.Manager.DeleteRegularCPUBoost(ctx, boostObj.Namespace, boostObj.Name)
	return true
}

func (r *StartupCPUBoostReconciler) Update(e event.UpdateEvent) bool {
	boostObj, ok := e.ObjectNew.(*autoscaling.StartupCPUBoost)
	if !ok {
		return true
	}
	log := r.Log.WithValues("name", boostObj.Name, "namespace", boostObj.Namespace)
	log.V(5).Info("handling boost update event")
	ctx := ctrl.LoggerInto(context.Background(), log)
	if err := r.Manager.UpdateRegularCPUBoost(ctx, boostObj); err != nil {
		log.Error(err, "boost update error")
	}
	return true
}

func (r *StartupCPUBoostReconciler) Generic(e event.GenericEvent) bool {
	log := r.Log.WithValues("object", klog.KObj(e.Object))
	log.V(5).Info("handling generic event")
	return true
}

// shouldUseLegacyRevertMode determines if legacy resource revert mode should be used
// basing on server version
func shouldUseLegacyRevertMode(serverVersion string) (legacyMode bool) {
	return version.CompareKubeAwareVersionStrings(WantedServerVersionForNewRevert,
		serverVersion) < 0
}
