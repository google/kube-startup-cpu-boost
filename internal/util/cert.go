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

package util

import (
	"fmt"

	cert "github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	certDir             = "/tmp/k8s-webhook-server/serving-certs"
	podBoostWebHookName = "kube-startup-cpu-boost-mutating-webhook-configuration"
	caName              = "kube-startup-cpu-boost-ca"
	caOrganization      = "kube-startup-cpu-boost"
	webhookServiceName  = "kube-startup-cpu-boost-webhook-service"
	webhookSecretName   = "kube-startup-cpu-boost-webhook-secret"
)

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=validatingwebhookconfigurations,verbs=get;list;watch;update

func ManageCerts(mgr ctrl.Manager, namespace string, setupFinished chan struct{}) error {
	dnsName := fmt.Sprintf("%s.%s.svc", webhookServiceName, namespace)
	return cert.AddRotator(mgr, &cert.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: namespace,
			Name:      webhookSecretName,
		},
		CertDir:        certDir,
		CAName:         caName,
		CAOrganization: caOrganization,
		DNSName:        dnsName,
		IsReady:        setupFinished,
		Webhooks: []cert.WebhookInfo{{
			Type: cert.Mutating,
			Name: podBoostWebHookName,
		}},
		RequireLeaderElection: false,
	})
}
