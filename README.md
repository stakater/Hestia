# Hestia

Hestia automates the management and execution of jobs in your Kubernetes cluster, supporting a variety of workload types (Deployments, DaemonSets, StatefulSets, DeploymentConfigs, and more). It enables you to define, schedule, and monitor custom job runners using Kubernetes-native resources.

## Why name "Hestia"?

In Greek mythology, Hestia is the goddess of the hearth, home, and stability. Just like her role in maintaining the foundation of the home, this operator ensures the foundational stability of your applications before they move forward — by automatically running E2E tests only when the system is truly ready.

## Features

- **Custom Job Runners:** Define and manage custom job execution logic via CRDs.
- **Scheduling:** Supports both immediate and scheduled (cron) job execution.
- **Resource Watching:** Automatically reacts to changes in Deployments, StatefulSets, DaemonSets, and DeploymentConfigs.
- **Status Reporting:** Tracks and reports job execution status and results.
- **Extensible:** Easily integrate with your CI/CD or automation workflows.

## Getting Started

### Usage Examples

#### 1. Unified Runner CR for Deployments, StatefulSets, DaemonSets, or DeploymentConfigs

```yaml
apiVersion: e2e.stakater.com/v1alpha1
kind: Runner
metadata:
  name: my-generic-runner
  labels:
    app: my-app
spec:
  workloadSelector:
    matchLabels:
      app: my-app
  template:
    spec:
      containers:
        - name: my-job
          image: busybox
          imagePullPolicy: IfNotPresent
          command: ["sh", "-c", "sleep 1 && echo done && exit 0"]
      restartPolicy: Never
```

**How to use:**
- Set `workloadSelector.matchLabels` to match the labels of your target Deployment, StatefulSet, DaemonSet, or DeploymentConfig.
- **Important:** The label `runner.stakater.com/enable: "true"` **must be set on the target workload** (e.g., Deployment, StatefulSet, DaemonSet, or DeploymentConfig) for the operator to watch and trigger jobs for it.
- **Note:** In Hestia Operator, `workloadSelector` is used for Deployments, StatefulSets, DaemonSets, and DeploymentConfigs.
- The operator will watch for changes in any of these resource types (Deployments, StatefulSets, DaemonSets, and DeploymentConfigs) that match the selector and trigger the job accordingly; see [When a job runs](#when-a-job-runs).

#### 2. Scheduled Runner (CronJob) for Any Resource

```yaml
apiVersion: e2e.stakater.com/v1alpha1
kind: Runner
metadata:
  name: my-scheduled-runner
  labels:
    app: my-app
spec:
  schedule: "* * * * *" # every minute
  deadlineSeconds: 120
  workloadSelector:
    matchLabels:
      app: my-app
  template:
    spec:
      containers:
        - name: my-cronjob
          image: busybox
          imagePullPolicy: IfNotPresent
          command: ["sh", "-c", "sleep 1 && echo done && exit 0"]
      restartPolicy: Never
```

**How to use:**
- Works for Deployments, StatefulSets, DaemonSets, or DeploymentConfigs—just match the label using `workloadSelector`.
- **Important:** The label `runner.stakater.com/enable: "true"` **must be set on the target workload** (e.g., Deployment, StatefulSet, DaemonSet, or DeploymentConfig) for the operator to watch and trigger jobs for it.
- The job will be scheduled according to the cron expression in `schedule`.

#### 3. Job Sequence (Chaining Runners)

**Runner 1:**

```yaml
apiVersion: e2e.stakater.com/v1alpha1
kind: Runner
metadata:
  name: runner-1
  labels:
    app: runner-1
    sequence: runner-sequence
spec:
  workloadSelector:
    matchLabels:
      app: runner-1-app
  template:
    spec:
      containers:
        - name: runner1-job
          image: busybox
          command: ["sh", "-c", "sleep 10 && echo done && exit 0"]
      restartPolicy: Never
```

**Runner 2 (waits for Runner 1 to finish):**

```yaml
apiVersion: e2e.stakater.com/v1alpha1
kind: Runner
metadata:
  name: runner-2
  labels:
    sequence: runner-sequence
spec:
  runnerSelector:
    matchLabels:
      app: runner-1
  template:
    spec:
      containers:
        - name: runner2-job
          image: busybox
          command: ["sh", "-c", "sleep 10 && echo done && exit 0"]
      restartPolicy: Never
```

**Tip:**

- Use the same pattern for any resource type by adjusting the `matchLabels` in `workloadSelector`. In Hestia Operator, `workloadSelector` is used for Deployments, StatefulSets, DaemonSets, and DeploymentConfigs.
- **Important:** The label `runner.stakater.com/enable: "true"` **must be set on the target workload** (e.g., Deployment, StatefulSet, DaemonSet, or DeploymentConfig) for the operator to watch and trigger jobs for it.
- For OpenShift, `workloadSelector` will also match DeploymentConfigs and DaemonSets.
- For more advanced scenarios, see the `config/samples/` directory and test fixtures.

### When a job runs

A Runner runs its job once all watched workloads are ready, and again after any of these, once the workloads are ready again:

- a rollout of a watched workload (new image, config change, `kubectl rollout restart`, `oc rollout latest`), however quickly it finishes
- scaling a watched workload, including by an autoscaler
- a workload starting or stopping to match the selector
- a change to the Runner itself

A pod restart or deletion without a rollout reruns the job only when it leaves the workload not ready long enough for the operator to notice. An operator restart or upgrade does not rerun it. No job runs while a watched workload is not ready or the selector matches nothing.

To rerun by hand, delete the Runner's job; it is recreated:

```bash
kubectl delete job -n <namespace> -l runner.stakater.com/name=<runner>
```

### Understanding Runner Status

Each `Runner` resource provides detailed status information to help you track job execution and resource readiness. The key fields in `.status` are:

- **conditions**:  
  An array of condition objects describing the current state of the Runner.  
  Common condition types include:
  - `ReconcileSuccess`: Indicates the controller has successfully reconciled the Runner resource.
  - `JobCompleted`: Indicates whether the most recent job run was completed.
    - `status: "True"`: The job completed successfully.
    - `status: "False"`: The job failed or is still running.
    - `reason`: Provides a short reason such as `Successful`, `Failed`, `Pending`, or `JobNotFound`.
    - `message`: Human-readable details about the job status.
  - `Ready`, `Reconciling`, `Stalled`: [kstatus](https://github.com/kubernetes-sigs/cli-utils/blob/master/pkg/kstatus/README.md) conditions summarising the Runner as a whole (see [Stable-state timeout](#stable-state-timeout)).

- **observedGeneration**:  
  The Runner generation the status was computed for.

- **progressingSince**:  
  When the watched workloads were first seen not ready. Cleared once they are all ready.

- **lastSuccessfulRunTime**:  
  Timestamp of the last successful job execution.

- **lastFailedRunTime**:  
  Timestamp of the last failed job execution.

- **watchedResources**:  
  Lists the resources (Deployments, StatefulSets, DaemonSets, and DeploymentConfigs) being watched by this Runner, including their name, namespace, kind, and readiness status.

#### Example: Runner Status Output

```yaml
status:
  conditions:
    - type: ReconcileSuccess
      status: "True"
      lastTransitionTime: "2024-05-01T12:00:00Z"
      reason: LastReconcileCycleSucceded
      message: ""
    - type: JobCompleted
      status: "True"
      lastTransitionTime: "2024-05-01T12:01:00Z"
      reason: Successful
      message: Reached expected number of succeeded pods
  lastSuccessfulRunTime: "2024-05-01T12:01:00Z"
  watchedResources:
    - name: deployment-1
      namespace: hestia-deployment-1
      kind: Deployment
      ready: true
    - name: statefulset-1
      namespace: hestia-statefulset-1
      kind: StatefulSet
      ready: true
    - name: daemonset-1
      namespace: hestia-daemonset-1
      kind: DaemonSet
      ready: true
    - name: deploymentconfig-1
      namespace: hestia-dc-1
      kind: DeploymentConfig
      ready: true
```

**How to interpret:**
- The `JobCompleted` condition with `status: "True"` and `reason: Successful` means the last job run finished successfully.
- The `watchedResources` array shows which resources (Deployments, StatefulSets, DaemonSets, DeploymentConfigs) are being monitored and their readiness.
- `lastSuccessfulRunTime` gives you the timestamp of the last successful job.

**Tip:**  
- If a job fails, check the `reason` and `message` fields in the conditions for troubleshooting hints.
- The `watchedResources` field helps you verify which resources are being monitored and their readiness.

### Stable-state timeout

When the workloads matched by `workloadSelector` start progressing (for example during a rollout) and do not all become ready within `stableStateTimeout`, the Runner reports `Stalled=True` with reason `StableStateTimeout`. The message names the workloads that are not ready.

```yaml
spec:
  workloadSelector:
    matchLabels:
      app: my-app
  stableStateTimeout: 30m # default; "0s" disables the timeout
```

- The clock starts when a watched workload is first seen not ready and resets once they are all ready.
- A selector that matches no workloads counts as not ready.
- If the workloads become ready after the timeout, the Stall clears and the job runs as normal.
- Every rollout of a watched workload triggers a new job run once it becomes ready again.

The kstatus conditions are derived as follows:

| Workloads | Last job | Ready | Reconciling | Stalled |
|---|---|---|---|---|
| progressing, within timeout | any | False | True `Progressing` | False |
| progressing, timed out | any | False | False | True `StableStateTimeout` |
| ready | pending / not found | False | True `WaitingForJob` | False |
| ready | succeeded | True `JobSucceeded` | False | False |
| ready | failed | False | False | True `JobFailed` |

Tools that understand kstatus work without configuration, e.g. `kubectl wait --for=condition=Ready runner/my-runner`.

### Argo CD health

Argo CD needs a health check per custom resource kind. Hestia ships one at [`config/argocd/e2e.stakater.com/Runner/health.lua`](config/argocd/e2e.stakater.com/Runner/health.lua) which maps `Stalled=True` to **Degraded**, `Ready=True` to **Healthy** and everything else to **Progressing**.

Add it to the `argocd-cm` ConfigMap:

```yaml
data:
  resource.customizations.health.e2e.stakater.com_Runner: |
    <contents of config/argocd/e2e.stakater.com/Runner/health.lua>
```

With the Argo CD operator, set the same script under `spec.resourceHealthChecks` of the `ArgoCD` resource (`group: e2e.stakater.com`, `kind: Runner`).

## Installation & Deployment

This section covers the requirements and methods for installing and deploying the Hestia Operator, both for local development and production clusters.

### Prerequisites

Before you build, deploy, or run the Hestia Operator, ensure you have the following tools and access:

- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Local Deployment (Manual)

Use this method if you want to build and deploy the operator manually for local development or testing purposes.

1. **Build and push the operator image:**
   ```sh
   make docker-build docker-push IMG=<your-registry>/hestia-operator:tag
   ```

2. **Install CRDs:**
   ```sh
   make install
   ```

3. **Deploy the operator:**
   ```sh
   make deploy IMG=<your-registry>/hestia-operator:tag
   ```

4. **Apply sample Runner CRs:**
   ```sh
   kubectl apply -k config/samples/
   ```

#### Uninstall/Remove the Operator (Local)

To remove the operator and its resources from your local or development cluster:

1. **Delete Runner CRs:**
   ```sh
   kubectl delete -k config/samples/
   ```
2. **Uninstall CRDs:**
   ```sh
   make uninstall
   ```
3. **Remove the operator deployment:**
   ```sh
   make undeploy
   ```

### Cluster Installation with OLM (Recommended for Production)

This is the recommended method for installing the operator in a production or shared cluster environment. OLM (Operator Lifecycle Manager) manages the lifecycle of the operator and makes upgrades and management easier.

The operator bundle and catalog images are published automatically via GitHub Actions. You can find the latest images at:
- **Bundle image:** `${BUNDLE_IMG}` (default: `ghcr.io/stakater/hestia-operator-bundle:v0.0.1`)
- **Catalog image:** `${CATALOG_IMG}` (default: `ghcr.io/stakater/hestia-operator-catalog:v0.0.1`)

#### 1. Create a CatalogSource

Apply a `CatalogSource` that points to your published catalog image:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: hestia-operator-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: <your-registry>/hestia-operator-catalog:v0.0.1
  displayName: Hestia Operator Catalog
  publisher: Stakater
```

#### 2. Create a Subscription

After the `CatalogSource` is ready, create a `Subscription` to install the operator:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: hestia-operator
  namespace: <target-namespace>
spec:
  channel: alpha
  name: hestia-operator
  source: hestia-operator-catalog
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
```

- Replace `<your-registry>` with your image registry (e.g., `ghcr.io/stakater`).
- Replace `<target-namespace>` with the namespace where you want the operator installed. On OpenShift, `openshift-operators` is commonly used for cluster-wide operators, but you can use any namespace as needed.
- By default, the CatalogSource is created in the `openshift-marketplace` namespace, which is standard for OpenShift. You can change this if you want to scope the operator to a different namespace.

**Note:** The default channel is `alpha`. You can change this if you have configured other channels in your bundle.

#### 3. Verify Installation

Check that the operator is installed and running:
```sh
kubectl get csv -n <target-namespace>
```

You should see a `ClusterServiceVersion` for `hestia-operator` in the `Succeeded` phase.

## Troubleshooting

- **RBAC Issues:** Ensure your user has cluster-admin privileges if you encounter permission errors.
- **Job Failures:** Check the Runner CR status and related Job/Pod logs for details.

## Contributing

Contributions are welcome! Please see the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html) for more on extending operators.

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

