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

package webhook

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type StartupCPUBoostWebhook struct{}

var _ admission.Validator[*v1alpha1.StartupCPUBoost] = &StartupCPUBoostWebhook{}

func setupWebhookForStartupCPUBoost(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &v1alpha1.StartupCPUBoost{}).
		WithValidator(&StartupCPUBoostWebhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-autoscaling-x-k8s-io-v1alpha1-startupcpuboost,mutating=false,failurePolicy=fail,sideEffects=None,groups=autoscaling.x-k8s.io,resources=startupcpuboosts,verbs=create;update,versions=v1alpha1,name=vstartupcpuboost.autoscaling.x-k8s.io,admissionReviewVersions=v1

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (w *StartupCPUBoostWebhook) ValidateCreate(ctx context.Context, boost *v1alpha1.StartupCPUBoost) (admission.Warnings, error) {
	log := ctrl.LoggerFrom(ctx).WithName("boost-validate-webhook")
	log.V(5).Info("handling create validation", "boost", klog.KObj(boost))
	return validateWarnings(boost), validate(boost)
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (w *StartupCPUBoostWebhook) ValidateUpdate(ctx context.Context, oldObj, boost *v1alpha1.StartupCPUBoost) (admission.Warnings, error) {
	log := ctrl.LoggerFrom(ctx).WithName("boost-validate-webhook")
	log.V(5).Info("handling update validation", "startupcpuboost", klog.KObj(boost))
	return validateWarnings(boost), validate(boost)
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (w *StartupCPUBoostWebhook) ValidateDelete(ctx context.Context, obj *v1alpha1.StartupCPUBoost) (admission.Warnings, error) {
	return nil, nil
}

func validateWarnings(boost *v1alpha1.StartupCPUBoost) admission.Warnings {
	var warnings admission.Warnings
	for i, policy := range boost.Spec.ResourcePolicy.ContainerPolicies {
		//lint:ignore SA1019 backwards-compatible support for deprecated ContainerName
		if policy.ContainerName != "" {
			warnings = append(warnings, fmt.Sprintf(
				"spec.resourcePolicy.containerPolicies[%d].containerName is deprecated; use matchContainers with type=ExactName instead", i,
			))
		}
	}
	return warnings
}

// validate verifies if Startup CPU Boost is valid. This is programmatic
// validation on a top of declarative API validation
func validate(boost *v1alpha1.StartupCPUBoost) error {
	var allErrs field.ErrorList
	if errs := validateContainerPolicies(boost.Spec.ResourcePolicy.ContainerPolicies); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}
	if err := validateDurationPolicy(boost.Spec.DurationPolicy); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{Group: "autoscaling.x-k8s.io", Kind: "StartupCPUBoost"},
			boost.Name, allErrs)
	}
	return nil
}

func validateDurationPolicy(policy v1alpha1.DurationPolicy) *field.Error {
	var cnt int
	fldPath := field.NewPath("spec").Child("")
	if policy.Fixed != nil {
		cnt++
	}
	if policy.PodCondition != nil {
		cnt++
	}
	if cnt != 1 {
		err := errors.New("one type of duration policy should be defined")
		return field.Invalid(fldPath, policy, err.Error())
	}
	return nil
}

func validateContainerPolicies(policies []v1alpha1.ContainerPolicy) field.ErrorList {
	var allErrs field.ErrorList
	baseFldPath := field.NewPath("spec").
		Child("resourcePolicy").
		Child("containerPolicies")
	for i := range policies {
		fldPath := baseFldPath.Index(i)
		if errs := validateContainerPolicyTypes(policies[i], fldPath); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
		if errs := validateContainerPolicyMatchers(policies[i], fldPath); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	return allErrs
}

func validateContainerPolicyTypes(policy v1alpha1.ContainerPolicy, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	var cnt int
	if policy.FixedResources != nil {
		cnt++
	}
	if policy.PercentageIncrease != nil {
		cnt++
	}
	if cnt != 1 {
		allErrs = append(allErrs, field.Invalid(fldPath,
			policy,
			"one type of resource policy should be defined",
		))
	}
	return allErrs
}

func validateContainerPolicyMatchers(policy v1alpha1.ContainerPolicy, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	var cnt int
	//lint:ignore SA1019 backwards-compatible support for deprecated ContainerName
	if policy.ContainerName != "" {
		cnt++
	}
	if policy.MatchContainers != nil {
		cnt++
		if policy.MatchContainers.Type == v1alpha1.MatchContainersTypeRegexName {
			if _, err := regexp.Compile(policy.MatchContainers.Value); err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("matchContainers", "value"),
					policy.MatchContainers.Value,
					fmt.Sprintf("invalid regular expression: %s", err),
				))
			}
		}
	}
	if cnt != 1 {
		allErrs = append(allErrs, field.Invalid(fldPath,
			policy,
			"either container name or container matcher should be defined",
		))
	}
	return allErrs
}
