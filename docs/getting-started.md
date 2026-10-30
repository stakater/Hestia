# Getting started — Hello, Hestia

This guide takes you from an empty cluster to a working Runner that automatically
runs a job every time a Deployment becomes ready. Budget ~10 minutes.

By the end you will understand the **two things Hestia needs**:

1. A **Runner** custom resource — *what* job to run and *which* workloads to watch.
2. The label **`runner.stakater.com/enable: "true"` on the target workload** — your
   explicit opt-in so Hestia watches it.

---

## Prerequisites

- A Kubernetes cluster you can `kubectl apply` to. A local
  [kind](https://kind.sigs.k8s.io/) or [minikube](https://minikube.sigs.k8s.io/)
  cluster is perfect.
- `kubectl` configured for that cluster.
- The Hestia operator installed. For a quick local run from a checkout:

  ```sh
  make install          # install the Runner CRD
  make run              # run the controller against your current kube-context
  ```

  (For a real cluster, use `make deploy IMG=...` or the
  [OLM install](../README.md#cluster-installation-with-olm-recommended-for-production).)

---

## Step 1 — Create a namespace and a workload to watch

```sh
kubectl create namespace hello-hestia
```

Deploy a plain nginx Deployment. **Note the `runner.stakater.com/enable` label** —
without it, Hestia ignores the workload entirely.

```yaml
# app.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: hello-hestia
  labels:
    app: web
    runner.stakater.com/enable: "true"   # <-- opt in to Hestia
spec:
  replicas: 2
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
        - name: web
          image: nginx:1.27-alpine
          ports:
            - containerPort: 80
```

```sh
kubectl apply -f app.yaml
```

---

## Step 2 — Create a Runner

The Runner says: *watch workloads labelled `app: web`, and when they are ready, run
this job.*

```yaml
# runner.yaml
apiVersion: e2e.stakater.com/v1alpha1
kind: Runner
metadata:
  name: web-smoke-test
  namespace: hello-hestia
spec:
  workloadSelector:
    matchLabels:
      app: web
  template:
    spec:
      containers:
        - name: smoke
          image: curlimages/curl:8.11.1
          command:
            - sh
            - -c
            - |
              echo "Running smoke test against web..."
              curl -fsS http://web.hello-hestia.svc.cluster.local && echo " OK"
      restartPolicy: Never
```

> You'll need a Service named `web` for the curl above to resolve. For a
> zero-dependency first run, replace the command with
> `["sh","-c","echo smoke-ok && exit 0"]`.

```sh
kubectl apply -f runner.yaml
```

---

## Step 3 — Watch it run

Once the Deployment reports all replicas ready, Hestia creates a Job:

```sh
kubectl get jobs -n hello-hestia -l runner.stakater.com/name=web-smoke-test
kubectl logs  -n hello-hestia  -l runner.stakater.com/name=web-smoke-test
```

Check the Runner's status — this is where Hestia reports the verdict:

```sh
kubectl get runner web-smoke-test -n hello-hestia -o yaml | yq '.status'
```

You should see a `JobCompleted` condition (`reason: Successful`), a
`lastSuccessfulRunTime`, and a `watchedResources` list showing `web` as `ready: true`.

Because Hestia emits [kstatus](https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus)
conditions, this also works:

```sh
kubectl wait --for=condition=Ready runner/web-smoke-test -n hello-hestia --timeout=120s
```

---

## Step 4 — See it re-run on a rollout

This is the whole point of Hestia. Trigger a rollout and watch a **new** job appear:

```sh
kubectl set image deployment/web web=nginx:1.27 -n hello-hestia
# or: kubectl rollout restart deployment/web -n hello-hestia
```

Hestia waits for the rollout to finish and the Deployment to be ready again, then
runs a fresh job. List jobs again — there are now two, each tagged with a different
`runner.stakater.com/version`:

```sh
kubectl get jobs -n hello-hestia -l runner.stakater.com/name=web-smoke-test \
  --show-labels
```

No CI pipeline triggered that. Hestia reacted to the cluster. The same happens when
an autoscaler scales `web`, or anyone runs `rollout restart`. See
[How it works » When a job runs](how-it-works.md#when-a-job-runs) for the exact rules.

### Re-run by hand

Delete the job; Hestia recreates it:

```sh
kubectl delete job -n hello-hestia -l runner.stakater.com/name=web-smoke-test
```

---

## Variation A — Run on a schedule (cron)

Add `spec.schedule`. Hestia creates a CronJob (still gated on the workload being
ready before it's set up) instead of a one-shot Job:

```yaml
apiVersion: e2e.stakater.com/v1alpha1
kind: Runner
metadata:
  name: web-hourly-check
  namespace: hello-hestia
spec:
  schedule: "0 * * * *"       # top of every hour
  deadlineSeconds: 120
  workloadSelector:
    matchLabels:
      app: web
  template:
    spec:
      containers:
        - name: check
          image: curlimages/curl:8.11.1
          command: ["sh","-c","curl -fsS http://web.hello-hestia.svc.cluster.local"]
      restartPolicy: Never
```

---

## Variation B — Chain runners into a sequence

A Runner can wait for *another Runner* to finish successfully using
`runnerSelector`. Here `runner-2` runs only after `runner-1` has a successful job.

```yaml
apiVersion: e2e.stakater.com/v1alpha1
kind: Runner
metadata:
  name: runner-1
  namespace: hello-hestia
  labels:
    app: runner-1              # <-- runner-2 selects on this
spec:
  workloadSelector:
    matchLabels:
      app: web
  template:
    spec:
      containers:
        - name: step1
          image: busybox
          command: ["sh","-c","echo step-1 && sleep 5 && exit 0"]
      restartPolicy: Never
---
apiVersion: e2e.stakater.com/v1alpha1
kind: Runner
metadata:
  name: runner-2
  namespace: hello-hestia
spec:
  runnerSelector:             # <-- wait for runner-1 (by its labels)
    matchLabels:
      app: runner-1
  template:
    spec:
      containers:
        - name: step2
          image: busybox
          command: ["sh","-c","echo step-2 && exit 0"]
      restartPolicy: Never
```

> Chaining is **one level deep** — "wait for the selected Runner(s) to succeed". For
> complex DAGs, use a workflow engine (Argo Workflows, Tekton) and point Hestia at
> the whole thing if you want change-triggered execution.

---

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| No job ever appears | The target workload is missing `runner.stakater.com/enable: "true"`, or `workloadSelector` doesn't match its labels. |
| No job appears | The workload isn't *ready* (replicas not all available), so the readiness gate holds. Check `kubectl get runner ... -o yaml` → `status.watchedResources`. |
| Job never re-runs on restart | A pod restart that doesn't make the workload *not-ready* long enough won't trigger a run. A rollout always does. See [When a job runs](how-it-works.md#when-a-job-runs). |
| `Stalled=True`, reason `StableStateTimeout` | The workload has been not-ready longer than `stableStateTimeout` (default 30m). The message names which workloads. |

---

## Next steps

- [How it works](how-it-works.md) — the architecture and the exact re-run rules.
- [Comparison with alternatives](comparison-with-alternatives.md).
- [Hestia vs kube-health](hestia-vs-kube-health.md).
- [Argo CD health integration](../README.md#argo-cd-health).
