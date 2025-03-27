# Development guide

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#summary) specification
for commit messages to automate the release process. The spec is enforced by github action based validation.

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

4. Enable development logging and other options if needed
  
   In `config/default/kustomization.yaml` uncomment the dev patch:

   ```yaml
   # Uncomment below for development
   - manager_config_dev_patch.yaml
   ```

   Adapt the `config/default/manager_config_dev_patch.yaml` if needed.

5. Deploy controller on a cluster

   ```sh
   make deploy IMG=docker.io/library/kube-startup-cpu-boost:dev
   ```
