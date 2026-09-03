# Kube Startup CPU Boost

Kube Startup CPU Boost is a controller that increases CPU resource requests and limits during
a Kubernetes workload's startup time. Once the workload is up and running,
the resources are set back to their original values.

[![Build](https://github.com/google/kube-startup-cpu-boost/actions/workflows/build.yaml/badge.svg)](https://github.com/google/kube-startup-cpu-boost/actions/workflows/build.yaml)
[![Version](https://img.shields.io/github/v/release/google/kube-startup-cpu-boost?label=version)](https://img.shields.io/github/v/release/google/kube-startup-cpu-boost?label=version)
[![Go Report Card](https://goreportcard.com/badge/github.com/google/kube-startup-cpu-boost)](https://goreportcard.com/report/github.com/google/kube-startup-cpu-boost)
![GitHub](https://img.shields.io/github/license/google/kube-startup-cpu-boost)

Note: This is not an officially supported Google product.

---

## Table of contents

* [Description](#description)
* [Installation](#installation)
* [Usage](#usage)
* [Features](#features)
  * [[Boost behavior] container restarts](#boost-behavior-container-restarts)
  * [[Boost target] Pod label selector](#boost-target-pod-label-selector)
  * [[Boost resources] container matcher](#boost-resources-container-matcher)
  * [[Boost resources] percentage increase](#boost-resources-percentage-increase)
  * [[Boost resources] fixed target](#boost-resources-fixed-target)
  * [[Boost duration] fixed time](#boost-duration-fixed-time)
  * [[Boost duration] Pod condition](#boost-duration-pod-condition)
* [Configuration](#configuration)
* [Metrics](#metrics)
* [Side Effects](#side-effects)
* [License](#license)

## Description

The primary use cases for Kube Startup CPU Boost are workloads that require extra CPU resources
during the startup phase — typically JVM-based applications.

Kube Startup CPU Boost leverages the [In-place Resource Resize for Kubernetes Pods](https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/)
feature introduced in Kubernetes 1.27. It allows reverting a workload's CPU resource requests and
limits back to their original values without the need to recreate the Pods.

The increase in resources is achieved via a Mutating Admission Webhook.
When [configured](#configuration), the webhook also removes CPU resource limits if present.
The original resource values are restored by the operator after a given period of time
or when a Pod condition is met.

> [!WARNING]
> While kube-startup-cpu-boost significantly reduces container cold-start times, dynamically
> mutating pod resources at runtime can introduce unintended side effects in specific environments.
> Please read [Side Effects](#side-effects) for more details.

## Installation

### Prerequisites

* **Requires Kubernetes 1.33 or newer**.
* For older clusters (>= 1.27), enable the `InPlacePodVerticalScaling` feature gate.

### Install with manifests file

 <!-- x-release-please-start-version -->
```sh
kubectl apply -f https://github.com/google/kube-startup-cpu-boost/releases/download/v0.21.1/manifests.yaml
```
 <!-- x-release-please-end -->

The Kube Startup CPU Boost components run in the `kube-startup-cpu-boost-system` namespace.

### Install with Kustomize

You can use [Kustomize](https://github.com/kubernetes-sigs/kustomize) to install Kube Startup CPU
Boost with your own kustomization file.

 <!-- x-release-please-start-version -->
```sh
cat <<EOF > kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- https://github.com/google/kube-startup-cpu-boost?ref=v0.21.1
EOF
kubectl kustomize | kubectl apply -f -
```
 <!-- x-release-please-end -->

### Install with Helm

Helm installation uses the self-hosted Helm chart repo [kube-startup-cpu-boost](https://google.github.io/kube-startup-cpu-boost).

```sh
helm repo add kube-startup-cpu-boost https://google.github.io/kube-startup-cpu-boost
helm repo update
helm install --create-namespace -n kube-startup-cpu-boost-system kube-startup-cpu-boost kube-startup-cpu-boost/kube-startup-cpu-boost
```

### Installation on a GKE cluster

Ensure that GKE is running version 1.33 or newer by using the
[GKE Rapid release channel](https://cloud.google.com/kubernetes-engine/docs/concepts/release-channels).

```sh
gcloud container clusters create poc \
    --release-channel rapid \
    --region europe-central2
```

### Running HA setup

Kube Startup CPU Boost supports multi-replica High Availability (HA) deployments. Running an HA
setup requires leader election to be enabled (`LEADER_ELECTION=true`), which is
**enabled by default**.

#### Helm

When installing via Helm, you can scale replicas and enable a `PodDisruptionBudget` to ensure at
least one replica remains available during cluster maintenance:

```sh
helm install \
  --create-namespace -n kube-startup-cpu-boost-system \
  kube-startup-cpu-boost kube-startup-cpu-boost/kube-startup-cpu-boost \
  --set controllerManager.replicas=2 \
  --set controllerManager.pdb.create=true \
  --set controllerManager.pdb.minAvailable=1
```

#### Kustomize

When installing via Kustomize from source (`config/default`), uncomment the HA component in
`config/default/kustomization.yaml`. This will increase number of replicas and
add `PodDisruptionBudget`.

## Usage

1. Create a `StartupCPUBoost` object in your workload's namespace

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
       - matchContainers:
           type: ExactName
           value: spring-rest-jpa
         percentageIncrease:
           value: 50
     durationPolicy:
       podCondition:
         type: Ready
         status: "True"
   ```

   The above example will boost the CPU requests and limits of the `spring-demo-app` container in
   Pods with the `app.kubernetes.io/name=spring-demo-app` label in the `demo` namespace.
   The resources will be increased by 50% until the
   [Pod Condition](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions)
   `Ready` becomes `True`.

2. Schedule your workloads and observe the results

## Features

### [Boost behavior] container restarts

Since [v0.21.0](https://github.com/google/kube-startup-cpu-boost/releases/tag/v0.21.0),
the operator can re-apply CPU boosts to existing Pods when their containers restart.
This ensures that the boost is applied throughout the entire lifecycle of the workload.

The operator continues watching Pods even after CPU resources have been restored to their
original values. Container restarts are detected by observing container status changes and reacting
to the `terminated` state.

> [!NOTE]
> This feature requires the `BOOST_ON_RESTART` environment variable to be set to `true`.

### [Boost target] Pod label selector

Define the Pods that will be subject to a resource boost with a label selector.

```yaml
spec:
  selector:
    matchExpressions:
    - key: app.kubernetes.io/name
       operator: In
       values: ["spring-rest-jpa"]
```

### [Boost resources] container matcher

Define the container(s) that will be subject to a resource boost with a container matcher.

The matcher can be defined with an exact container name or a regex pattern
matching a container name using [Go syntax](https://pkg.go.dev/regexp).

```yaml
spec:
  resourcePolicy:
    containerPolicies:
     - containerName: spring-rest-jpa # legacy container name matcher
       percentageIncrease:
         value: 50
    - matchContainers:
        type: ExactName
        value: spring-rest-jpa
      percentageIncrease:
        value: 50
    - matchContainers:
        type: RegexName
        value: "spring-.+"
      percentageIncrease:
        value: 50
```

### [Boost resources] percentage increase

Define the percentage increase for the target container(s). The CPU requests and limits of the
selected container(s) will be increased by the given percentage value.

```yaml
spec:
  resourcePolicy:
    containerPolicies:
    - matchContainers:
        type: ExactName
        value: spring-rest-jpa
      percentageIncrease:
        value: 50
```

### [Boost resources] fixed target

Define the fixed resources for the target container(s). The CPU requests and limits of the selected
container(s) will be set to the given values. Note that the specified requests and limits must be
higher than the container's original values.

```yaml
spec:
  resourcePolicy:
    containerPolicies:
     - matchContainers:
         type: ExactName
         value: spring-rest-jpa
       fixedResources:
         requests: "1"
         limits: "2"
```

### [Boost duration] fixed time

Define a fixed duration for which the resource boost will last, measured from the
**Pod's schedule time**.

```yaml
spec:
 durationPolicy:
  fixedDuration:
    unit: Seconds
    value: 60
```

### [Boost duration] Pod condition

Define a Pod condition; the resource boost effect will remain active until the condition is met.

  ```yaml
  spec:
   durationPolicy:
     podCondition:
       type: Ready
       status: "True" 
  ```

> **Note:** You can define multiple duration policies in a single boost (i.e. both `podCondition`
and `fixedDuration`). When combined, the boost is removed as soon as **either** condition is met.

## Configuration

The Kube Startup CPU Boost operator can be configured with environment variables.

| Variable | Type | Default | Description |
| --- | --- | --- | --- |
| `POD_NAMESPACE` | `string` | `kube-startup-cpu-boost-system` |  Kube Startup CPU Boost operator namespace |
| `MGR_CHECK_INTERVAL` | `int` | `5` | Duration in seconds between boost manager checks for time-based boost duration policies |
| `LEADER_ELECTION` | `bool` | `false` | Enables leader election for controller manager |
| `METRICS_PROBE_BIND_ADDR` | `string` | `:8080` | Address the metrics endpoint binds to |
| `HEALTH_PROBE_BIND_ADDR` | `string` | `:8081` | Address the health probe endpoint binds to |
| `SECURE_METRICS` | `bool` | `false` | Determines if the metrics endpoint is served securely |
| `ZAP_LOG_LEVEL` | `int` | `0` | Log level for ZAP logger |
| `ZAP_DEVELOPMENT` | `bool` | `false` | Enables development mode for ZAP logger |
| `HTTP2` | `bool` | `false` | Determines if the HTTP/2 protocol is used for webhook and metrics servers |
| `REMOVE_LIMITS` | `bool` | `false` | Enables the operator to remove container CPU limits during the boost period |
| `VALIDATE_FEATURE_ENABLED` | `bool` | `true` | Enables validation of the required feature gate on operator startup |
| `BOOST_ON_RESTART` | `bool` | `false` | Enables the [boost on container restart](#boost-behavior-container-restarts) feature |

## Metrics

Kube Startup CPU Boost exposes [Prometheus](https://prometheus.io) metrics to monitor the health of
the system and the status of Startup CPU Boosts.

| Metric name | Type | Description | Labels |
| --- | --- | --- | --- |
| `boost_configurations` | Gauge | Number of registered Kube Startup CPU Boost configurations | `namespace`: the namespace of the Kube Startup CPU Boost |
| `boost_containers_total` | Counter | Number of containers whose CPU resources were increased | `namespace`: the namespace of the container's Pod, `boost`: the name of the Kube Startup CPU Boost that increased the container's resources  |
| `boost_containers_active` | Gauge | Number of containers whose CPU resources have not yet been reverted to their original values | `namespace`: the namespace of the container's Pod, `boost`: the name of the Kube Startup CPU Boost that increased the container's resources  |

### Scraping: Google Cloud Managed Service for Prometheus

The following [PodMonitoring](https://github.com/GoogleCloudPlatform/prometheus-engine/blob/main/doc/api.md#monitoring.googleapis.com/v1.PodMonitoring)
example can be used to scrape Kube Startup CPU Boost metrics with
[Google Cloud Managed Service for Prometheus](https://cloud.google.com/stackdriver/docs/managed-prometheus).

```yaml
apiVersion: monitoring.googleapis.com/v1
kind: PodMonitoring
metadata:
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: controller-manager-metrics-monitor
    app.kubernetes.io/instance: controller-manager-metrics-monitor
    app.kubernetes.io/component: metrics
    app.kubernetes.io/created-by: kube-startup-cpu-boost
    app.kubernetes.io/part-of: kube-startup-cpu-boost
  name: controller-manager-metrics-monitor
  namespace: kube-startup-cpu-boost-system
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  endpoints:
  - port: metrics
    interval: 15s

```

## Side Effects

While `kube-startup-cpu-boost` significantly reduces container cold-start times, dynamically
mutating pod resources at runtime can introduce unintended side effects in specific environments.

### Runtime ergonomics

Many modern language runtimes, most notably the JVM, are container-aware and inspect Linux cgroups
at startup to determine available CPU shares and quotas. JVM uses this data to calculate the
`ActiveProcessorCount`, which governs the size of internal thread pools (e.g., Garbage Collection
threads, JIT compiler queues, and application worker pools).

* **The Impact**: Because this operator boosts CPU resources during the startup phase, the runtime
  will calculate an artificially high processor count and provision its thread pools accordingly.
  When the operator reverts the resources back to the lower steady-state limits without restarting
  the pod, these internal thread pools do not dynamically shrink.

* **The Result**: A container with a steady-state limit of 1 CPU might be running thread pools sized
  for 4 CPUs. This can lead to increased context-switching and CPU throttling, which can negatively
  impact steady-state performance.

* **Mitigation**:

  * **Keep Boosts Modest**: Adding massive amounts of CPU during startup rarely yields proportional
    speedups. By limiting your boost delta to 1-2 cores, you minimize the steady-state thread pool
    bloat while still securing faster startup times.

  * **Manual Runtime Flags**: For JVM workloads, you can manually override the container detection
    behavior by explicitly setting the `-XX:ActiveProcessorCount=<steady-state-cores>` flag.

    *Note: Hardcoding this flag may bottleneck the startup phase by limiting the threads available
    for JIT compilation and GC. It is a trade-off between peak startup speed and steady-state
    ergonomics.*

### Cluster Autoscaler

Cluster schedulers and autoscalers (like Kubernetes Cluster Autoscaler or Karpenter) make
provisioning decisions based on a pod's `resources.requests`. In-place pod resource resizing makes
these resource requirements mutable.

* **The Impact**: When a pod starts, the operator increases its CPU requests, making the workload
  temporarily "heavy". In highly optimized clusters, the scheduler may not find existing capacity
  for this boosted pod, triggering the  autoscaler to provision a new node.

* **The Result**: After the pod finishes starting, the operator reduces the resource requests back
  to the steady state.  The newly provisioned is underutilized now. To optimize costs,
  the autoscaler may cordon the node, evict the pod, and reschedule it elsewhere. When the pod is
  scheduled on a new node, the operator intercepts it, boosts it again, and triggers another
  scale-out event, possibly leading to a deployment loop.

* **When is it safe to use?**: Mutating CPU requests for startup boosts is best suited for specific
  cluster topologies:

  * **Static Node Pools**: Clusters with fixed capacity where autoscaling is disabled.

  * **Over-provisioned Clusters (Headroom)**: Clusters that intentionally maintain "buffer" nodes,
    where the initial startup boost can be absorbed by existing nodes without triggering
    a scale-up event.

  * **Conservative Scale-Down**: Clusters where scale-down or consolidation is either disabled or
    configured with very long grace periods.

### CPU limits removal

The `kube-startup-cpu-boost` operator can be configured to completely remove CPU limits during
startup (`REMOVE_LIMITS=true`) to maximize burst capacity.
**This is discouraged for JVM workloads**.

While CPU limits are often viewed purely as a throttling mechanism for CPU schedulers, the JVM
container detection mechanism uses them to set JVM internals.

* **The Mechanism**: Kubernetes CPU limits translate directly to `cpu.quota` in Linux cgroups.
  The JVM reads this quota to calculate its `ActiveProcessorCount`, which hardcodes the size of its
  GC threads, JIT compiler queues and worker pools.

* **The JDK Enhancement (JDK-8281181)**: Historically, the JVM used CPU requests (`cpu.shares`)
  to detect core count. This was changed, and modern JVMs (including backports to Java 11 and 17)
  **no longer use CPU requests to compute processor count**
  (see [JDK-8281181](https://bugs.openjdk.org/browse/JDK-8281181)).

* **The Result**: If you configure the operator to remove CPU limits (or deploy pods without them),
  the JVM defaults to the **total physical cores of the underlying Kubernetes node**
  (ignoring CPU requests). A JVM running in a pod requesting `500m` on a 64-core node will have 64
  active processors, resulting in side effects described in [Runtime ergonomics](#runtime-ergonomics).

**Recommendation**: For JVM workloads, treat CPU limits as a strict architectural boundary,
not just a throttle control. Always keep limits intact for JVM applications, or explicitly set the
internal core count by using the `-XX:ActiveProcessorCount=<N>` flag.

## License

[Apache License 2.0](LICENSE)
