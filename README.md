# Kube Startup CPU Boost

Kube Startup CPU Boost is a controller that increases CPU resource requests and limits during
Kubernetes workload startup time. Once the workload is up and running,
the resources are set back to their original values.

[![Build](https://github.com/google/kube-startup-cpu-boost/actions/workflows/build.yaml/badge.svg)](https://github.com/google/kube-startup-cpu-boost/actions/workflows/build.yaml)
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
  * [[Boost resources] fixed target](#boost-resources-fixed-target)
  * [[Boost duration] fixed time](#boost-duration-fixed-time)
  * [[Boost duration] POD condition](#boost-duration-pod-condition)
* [Configuration](#configuration)
* [License](#license)

## Description

The primary use cases for Kube Startup CPU Boosts are workloads that require extra CPU resources during
startup phase - typically JVM based applications.

The Kube Startup CPU Boost leverages [In-place Resource Resize for Kubernetes Pods](https://kubernetes.io/blog/2023/05/12/in-place-pod-resize-alpha/)
feature introduced in Kubernetes 1.27. It allows to revert workload's CPU resource requests and limits
back to their original values without the need to recreate the Pods.

The increase of resources is achieved by Mutating Admission Webhook. By default, the webhook also
removes CPU resource limits if present. The original resource values are set by operator after given
period of time or when the POD condition is met.

## Installation

**Requires Kubernetes 1.27 on newer with `InPlacePodVerticalScaling` feature gate
enabled.**

To install the latest release of Kube Startup CPU Boost in your cluster, run the following command:

 <!-- x-release-please-start-version -->
```sh
kubectl apply -f https://github.com/google/kube-startup-cpu-boost/releases/download/v0.10.0/manifests.yaml
```
 <!-- x-release-please-end -->

The Kube Startup CPU Boost components run in `kube-startup-cpu-boost-system` namespace.

### Install with Kustomize

You can use [Kustomize](https://github.com/kubernetes-sigs/kustomize) to install the Kube Startup CPU
Boost with your own kustomization file.

 <!-- x-release-please-start-version -->
```sh
cat <<EOF > kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- https://github.com/google/kube-startup-cpu-boost?ref=v0.10.0
EOF
kubectl kustomize | kubectl apply -f -
```
 <!-- x-release-please-end -->

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

### [Boost resources] fixed target

Define the fixed resources for a target container(s). The CPU requests and limits of selected
container(s) will be set to the given values. Note that specified requests and limits have to be
higher than the ones in the container.

```yaml
spec:
  containerPolicies:
   - containerName: spring-rest-jpa
     fixedResources:
       requests: "1"
       limits: "2"
```

### [Boost resources] auto

Define the percentage increase for a target container(s). The CPU requests and limits of selected
container(s) will be increase by the given percentage value.

```yaml
spec:
  containerPolicies:
   - containerName: spring-rest-jpa
     autoPolicy: 
       apiEndpoint: "http://exampleUrl:examplePort"
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

### [Boost duration] auto

Define the POD condition, the resource boost effect will last for the predicted duration.

  ```yaml
  spec:
   durationPolicy:
     autoPolicy: 
       apiEndpoint: "http://exampleUrl:examplePort"
  ```

## Configuration

Kube Startup CPU Boost operator can be configured with environmental variables.

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `POD_NAMESPACE` | `string` | `kube-startup-cpu-boost-system` |  Kube Startup CPU Boost operator namespace |
| `MGR_CHECK_INTERVAL` | `int` | `5` | Duration in seconds between boost manager checks for time based boost duration policy |
| `LEADER_ELECTION` | `bool` | `false` | Enables leader election for controller manager |
| `METRICS_PROBE_BIND_ADDR` | `string` | `:8080` | Address the metrics endpoint binds to |
| `HEALTH_PROBE_BIND_ADDR` | `string` | `:8081` | Address the health probe endpoint binds to |
| `SECURE_METRICS` | `bool` | `false` | Determines if the metrics endpoint is served securely |
| `ZAP_LOG_LEVEL` | `int` | `0` | Log level for ZAP logger |
| `ZAP_DEVELOPMENT` | `bool` | `false` | Enables development mode for ZAP logger |
| `HTTP2` | `bool` | `false` | Determines if the HTTP/2 protocol is used for webhook and metrics servers|
| `REMOVE_LIMITS` | `bool` | `true` | Enables operator to remove container CPU limits during the boost time |

## License

[Apache License 2.0](LICENSE)
