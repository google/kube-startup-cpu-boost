# Kube Startup CPU Boost

Kube Startup CPU Boost is a controller that increases CPU resource requests and limits during
Kubernetes workload startup time. Once the workload is up and running,
the resources are set back to their original values.

[![Build](https://github.com/google/kube-startup-cpu-boost/actions/workflows/build.yml/badge.svg)](https://github.com/google/kube-startup-cpu-boost/actions/workflows/build.yml)
[![Version](https://img.shields.io/github/v/release/google/kube-startup-cpu-boost?label=version)](https://img.shields.io/github/v/release/google/kube-startup-cpu-boost?label=version)
[![Go Report Card](https://goreportcard.com/badge/github.com/google/kube-startup-cpu-boost)](https://goreportcard.com/report/github.com/google/kube-startup-cpu-boost)
![GitHub](https://img.shields.io/github/license/google/kube-startup-cpu-boost)

Note: this is not an officially supported Google product.

---

## Table of contents

* [Description](#description)
* [Installation](#installation)
* [Usage](#usage)
* [Features](#features)
  * [[Boost target] POD label selector](#boost-target-pod-label-selector)
  * [[Boost resources] percentage increase](#boost-resources-percentage-increase)
  * [[Boost duration] fixed time](#boost-duration-fixed-time)
  * [[Boost duration] POD condition](#boost-duration-pod-condition)
* [License](#license)

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
kubectl apply -f https://github.com/google/kube-startup-cpu-boost/releases/download/v0.2.0/manifests.yaml
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

   ```yaml
   apiVersion: autoscaling.x-k8s.io/v1alpha1
   kind: StartupCPUBoost
   metadata:
     name: boost-001
     namespace: demo
   selector:
     matchExpressions:
     - key: app.kubernetes.io/name
       operator: In
       values: ["spring-demo-app"]
   spec:
     resourcePolicy:
       containerPolicies:
       - containerName: spring-demo-app
         percentageIncrease:
           value: 50
     durationPolicy:
       podCondition:
         type: Ready
         status: "True"
   ```

   The above example will boost CPU requests and limits of a container `spring-demo-app` in a
   PODs with `app.kubernetes.io/name=spring-demo-app` label in `demo` namespace.
   The resources will be increased by 50% until the
   [POD Condition](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions)
   `Ready` becomes `True`.

2. Schedule your workloads and observe the results

## Features

### [Boost target] POD label selector

Define the PODs that will be subject for resource boost with a label selector.

```yaml
spec:
  selector:
    matchExpressions:
    - key: app.kubernetes.io/name
       operator: In
       values: ["spring-rest-jpa"]
```

### [Boost resources] percentage increase

Define the percentage increase for a target container(s). The CPU requests and limits of selected
container(s) will be increase by the given percentage value.

```yaml
spec:
  containerPolicies:
   - containerName: spring-rest-jpa
     percentageIncrease:
       value: 50
```

### [Boost duration] fixed time

Define the fixed amount of time, the resource boost effect will last for it since the POD creation.

```yaml
spec:
 durationPolicy:
  fixedDuration:
    unit: Seconds
    value: 60
```

### [Boost duration] POD condition

Define the POD condition, the resource boost effect will last until the condition is met.

  ```yaml
  spec:
   durationPolicy:
     podCondition:
       type: Ready
       status: "True" 
  ```

## License

[Apache License 2.0](LICENSE)
