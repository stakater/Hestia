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
  deploymentSelector:
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
- Set `deploymentSelector.matchLabels` to match the labels of your target Deployment, StatefulSet, DaemonSet, or DeploymentConfig.
- **Note:** In Hestia Operator, `deploymentSelector` is used for Deployments, StatefulSets, DaemonSets, and DeploymentConfigs.
- The operator will watch for changes in any of these resource types (Deployments, StatefulSets, DaemonSets, and DeploymentConfigs) that match the selector and trigger the job accordingly.

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
  deploymentSelector:
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
- Works for Deployments, StatefulSets, DaemonSets, or DeploymentConfigs—just match the label using `deploymentSelector`.
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
  deploymentSelector:
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

- Use the same pattern for any resource type by adjusting the `matchLabels` in `deploymentSelector`. In Hestia Operator, `deploymentSelector` is used for Deployments, StatefulSets, DaemonSets, and DeploymentConfigs.
- For OpenShift, `deploymentSelector` will also match DeploymentConfigs and DaemonSets.
- For more advanced scenarios, see the `config/samples/` directory and test fixtures.

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

