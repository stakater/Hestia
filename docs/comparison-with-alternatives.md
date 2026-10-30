# Hestia vs. the alternatives

There are many ways to "run something after a deployment". This page is an honest
map of the landscape, what makes Hestia different, and — just as important — when
you should **not** reach for it.

> For the dedicated, in-depth comparison with Stakater's own
> **kube-health (KHI)**, see [Hestia vs kube-health](hestia-vs-kube-health.md).

## The one thing that makes Hestia different

Almost every other tool triggers on **time** (a cron) or on **a step in a pipeline
you control** (CI/CD, GitOps hooks). Hestia triggers on the **live readiness of your
workloads**, in-cluster, regardless of who or what caused the change.

That means Hestia catches deployment paths the pipeline never sees:

- an Argo CD / Flux sync that changed the app,
- an autoscaler scaling it,
- a `kubectl rollout restart` or `oc rollout latest` by an operator or a human,
- a ConfigMap/Secret-driven restart.

If your verification only needs to run as the last step of a pipeline you already
own, you probably don't need Hestia — a test step there is simpler and gives you
richer reporting. Hestia earns its keep when **"ready" can happen by many paths**.

## At a glance

| Tool | Trigger model | Lives where | Reports via | Best at |
|---|---|---|---|---|
| **Hestia** | Workload **readiness / rollout** (optional cron) | In-cluster operator | Runner status (kstatus), Argo CD health | Change-triggered post-deploy verification, whoever triggered the change |
| **Kubernetes CronJob** | Time only | In-cluster | You wire it up | Fixed-schedule batch work |
| **CI/CD post-deploy tests** (GitHub Actions, GitLab CI, Jenkins, Tekton) | A step in your pipeline | CI system | CI run / artifacts | Verifying a deploy *that went through that pipeline*; rich reporting |
| **Argo CD PostSync hooks** | Argo CD sync lifecycle | In-cluster (Argo-managed) | Argo CD sync result | Verification bound to an Argo sync |
| **Argo Rollouts analysis** | Progressive-delivery step | In-cluster | Rollout status / metrics | Metric-based canary/blue-green gating |
| **Testkube** | Webhook / schedule / manual / events | In-cluster test platform | Testkube dashboard/API | A full in-cluster test orchestration platform |
| **kuttl / chainsaw** | `kubectl`/CI invocation | CLI | Test report | Asserting operator/controller behaviour in CI |
| **Sonobuoy** | Manual / scheduled | In-cluster | Report bundle | Cluster conformance/diagnostics |
| **kube-health (KHI)** | Time (CronJob) | In-cluster (Helm) | ConfigMap + Prometheus Pushgateway + Argo health | Periodic synthetic health probing with metrics |

## How Hestia differs, tool by tool

### vs. Kubernetes CronJob
A CronJob fires on a clock and knows nothing about whether your app just changed or
is healthy. Hestia fires **on the change** and **only when ready**. (Hestia can
*also* run on a cron via `spec.schedule` — but then you're using it as a readiness-
gated CronJob, which is a thin slice of its value.)

### vs. CI/CD post-deploy test steps
This is the real alternative for most teams, and often the right one. A pipeline
test step is simple, gives you logs/artifacts/retries/notifications for free, and
needs no extra in-cluster component. Its structural limitation: it only runs for
deployments that **flow through that pipeline**. If an autoscaler, a GitOps
self-heal, or a human changes the cluster, the pipeline never knows — and nothing
re-verifies. Hestia closes that gap. Many teams run **both**: pipeline tests for the
release, Hestia for everything that happens to the workload afterward.

### vs. Argo CD PostSync hooks
PostSync hooks run a Job as part of an Argo sync — great, but only on a sync, and
only if Argo is your delivery tool. Hestia is **delivery-tool-agnostic** and keeps
reacting after the sync (to scaling, restarts, drift), reporting its own health that
Argo then reads via the shipped `health.lua`. Use PostSync hooks when "verify this
exact sync" is all you need; use Hestia when you want continuous, source-agnostic
verification.

### vs. Argo Rollouts analysis
Argo Rollouts gates a *progressive delivery* step on metrics/analysis before
promoting a canary. That's a different job: it decides *whether to proceed with a
rollout*. Hestia runs a *verification job after a workload is ready*. They compose —
Rollouts for promotion gating, Hestia for post-settle E2E.

### vs. Testkube
Testkube is a full test-orchestration platform (test CRDs, executors, a dashboard,
triggers including events). It overlaps with Hestia's "run a test in-cluster, triggered
by cluster events" idea and does far more around test management and reporting. Hestia
is deliberately small: one CRD, Job semantics, readiness-gated triggering, kstatus
health. Choose Testkube if you want a platform; choose Hestia if you want a focused,
GitOps-native "verify on every ready" primitive with almost no surface area.

### vs. kuttl / chainsaw
These assert controller/operator behaviour in CI by applying manifests and checking
outcomes. They're test *authoring/assertion* frameworks run from CI, not in-cluster
reactive runners. Different layer entirely.

### vs. Sonobuoy
Cluster-level conformance and diagnostics, run on demand. Not workload-scoped or
change-triggered. No overlap in practice.

## When to use Hestia

- You run **GitOps** (Argo CD/Flux) and deployments don't all pass through one
  pipeline you control.
- You want post-deploy verification to **re-fire on scale/restart/drift**, not only
  on the release commit.
- You want the verdict as a **first-class Kubernetes health status** (kstatus +
  Argo CD health), not buried in a CI log.
- You're on **OpenShift** and need `DeploymentConfig` support.

## When *not* to use Hestia

- All your deploys go through **one pipeline you own** — a post-deploy test step
  there is simpler and better-instrumented.
- You need **rich test reporting**, artifacts, flake-retries, or notifications out
  of the box — Hestia gives you Job semantics + status conditions; you build the
  rest inside the Job.
- You want **periodic synthetic monitoring with Prometheus metrics** — that's
  [kube-health (KHI)](hestia-vs-kube-health.md), not Hestia.
- You need **multi-step DAG workflows** — use Argo Workflows / Tekton (Hestia
  chaining is one level deep).
- You can't yet absorb an **alpha-stage** dependency — Hestia is `v1alpha1`.

## A note on re-run cost

Because Hestia re-runs on every path into "ready", a noisy workload (frequent
autoscaling, frequent restarts) can trigger frequent jobs. That's usually what you
want for verification, but keep the job cheap and idempotent, and set
`deadlineSeconds`. See [How it works » When a job runs](how-it-works.md#when-a-job-runs).
