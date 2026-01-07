# Development guide

## Branching Strategy

### Long-Lived Development Branch

We use a **long-lived `development` branch** for feature development. All story branches branch off and merge back to `development` until code complete.

**Branch Structure**:
```
main
  └── development (long-lived)
       ├── story/epic-001-us-001-api-schema
       ├── story/epic-001-us-002-activation-model
       ├── story/epic-002-us-005-restart-detection
       └── ... (other story branches)
```

**Workflow**:
1. **Create story branch** from `development`:
   ```bash
   git checkout development
   git pull origin development
   git checkout -b story/epic-001-us-001-api-schema
   ```

2. **Work on story**:
   - Make commits following [Conventional Commits](#commit-messages)
   - Push branch and create PR targeting `development`

3. **Merge back to development**:
   - PR reviewed and approved
   - Merge to `development` (squash or merge commits as preferred)
   - Delete story branch after merge

4. **Repeat** until all stories complete

5. **Final merge to main**:
   - When all EPICS are code complete
   - Create PR from `development` to `main`
   - After merge, tag release

**Branch Naming Convention**:
- Story branches: `story/epic-{number}-us-{number}-{short-description}`
  - Example: `story/epic-001-us-001-api-schema`
  - Example: `story/epic-002-us-005-restart-detection`
- Hotfix branches: `hotfix/{description}` (if needed)

**Best Practices**:
- Always branch from latest `development`
- Keep story branches focused (one user story per branch)
- Rebase on `development` before creating PR if needed
- Delete merged story branches to keep repository clean

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
