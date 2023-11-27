# Kube Startup CPU Boost

Kube Startup CPU Boost is a controller that increases CPU resource requests and limits during
Kubernetes workload startup time. Once the workload is up and running,
the resources are set back to their original values.

[![Build](https://github.com/google/kube-startup-cpu-boost/actions/workflows/build.yml/badge.svg)](https://github.com/google/kube-startup-cpu-boost/actions/workflows/build.yml)
[![Version](https://img.shields.io/github/v/release/google/kube-startup-cpu-boost?label=version)](https://img.shields.io/github/v/release/google/kube-startup-cpu-boost?label=version)
[![Go Report Card](https://goreportcard.com/badge/github.com/google/kube-startup-cpu-boost)](https://goreportcard.com/report/github.com/google/kube-startup-cpu-boost)
![GitHub](https://img.shields.io/github/license/google/kube-startup-cpu-boost)

Note: this is not an officially supported Google product.

## Description

The primary use cases for Kube Startup CPU Boosts are workloads that require extra CPU resources during
startup phase - typically JVM based applications.

The Kube Startup CPU Boost leverages [In-place Resource Resize for Kubernetes Pods](https://kubernetes.io/blog/2023/05/12/in-place-pod-resize-alpha/)
feature introduced in Kubernetes 1.27. It allows to revert workload's CPU resource requests and limits
back to their original values without the need to recreate the Pods.

The increase of resources is achieved by Mutating Admission Webhook.

## Installation

**Requires Kubernetes 1.27 on newer with `InPlacePodVerticalScaling` feature gate
enabled.**

To install the latest release of Kube Startup CPU Boost in your cluster, run the following command:

```sh
kubectl apply -f https://github.com/google/kube-startup-cpu-boost/releases/download/v0.0.2/manifests.yaml
```

The Kube Startup CPU Boost components run in `kube-startup-cpu-boost-system` namespace.

### Installation on Kind cluster

You can use [KIND](https://github.com/kubernetes-sigs/kind) to get a local cluster for testing.

```sh
cat <<EOF > kind-poc-cluster.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: poc
nodes:
- role: control-plane
- role: worker
- role: worker
featureGates:
  InPlacePodVerticalScaling: true 
EOF
kind create cluster --config kind-poc-cluster.yaml
```

### Installation on GKE cluster

You can use [GKE Alpha cluster](https://cloud.google.com/kubernetes-engine/docs/concepts/alpha-clusters)
to run against the remote cluster.

```sh
gcloud container clusters create poc \
    --enable-kubernetes-alpha \
    --no-enable-autorepair \
    --no-enable-autoupgrade \
    --region europe-central2
```

## Usage

1. Create `StartupCPUBoost` object in your workload's namespace

   ```sh
   apiVersion: autoscaling.x-k8s.io/v1alpha1
   kind: StartupCPUBoost
   metadata:
     name: boost-001
     namespace: demo
   selector:
     matchExpressions:
     - key: app
       operator: In
       values: ["app-001", "app-002"]
   spec:
     boostPercent: 50
     durationPolicy:
       podCondition:
         type: Ready
         status: "True"
   ```

   The above example will boost CPU requests and limits of all PODs with `app=app-001` and `app=app-002`
   labels in `demo` namespace. The resources will be increased by 50% until the
   [POD Condition](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions)
   `Ready` becomes `True`.

2. Schedule your workloads and observe the results

## License

[Apache License 2.0](LICENSE)
