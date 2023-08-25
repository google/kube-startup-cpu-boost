# kube-startup-cpu-boost development guide

## Running development cluster

You can use [KIND](https://github.com/kubernetes-sigs/kind) to get a local cluster for development.

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

## Deploying development version

1. Install CRDs on a cluster

   ```sh
   make install
   ```

2. Build docker image

   ```sh
   make docker-build IMG=kube-startup-cpu-boost:dev
   ```

3. Load docker image to development cluster

   ```sh
   kind load docker-image --name poc kube-startup-cpu-boost:dev
   ```

4. Deploy controller on a cluster

   ```sh
   make deploy IMG=docker.io/library/kube-startup-cpu-boost:dev
   ```
