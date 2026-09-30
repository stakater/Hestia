# Stable-state timeout for Runner workloads

Date: 2026-09-30
Status: Approved design, pending spec review

## Goal

When the workloads selected by a Runner's `workloadSelector` start progressing
(e.g. a rollout) but do not all become ready within `stableStateTimeout`, the
Runner reports a failed state that Argo CD shows as **Degraded**.

Success criteria:

- A Runner whose watched workloads stay not-ready longer than the timeout gets
  `Stalled=True` with reason `StableStateTimeout` and a message naming the
  not-ready workloads.
- Argo CD shows that Runner as Degraded once the shipped Lua health check is
  configured.
- If the workloads later become ready, the Stall clears and the job runs as
  normal.
- Existing condition types (`ReconcileSuccess`, `JobCompleted`) and Runner
  chaining via `runnerSelector` keep working unchanged.

## Out of scope

- Upstream contribution of the health check to `argoproj/argo-cd`
  `resource_customizations` (the fixtures are written in that format so it can
  follow later).
- Timeouts for upstream Runners selected via `runnerSelector`.
- Removing or renaming `ReconcileSuccess` / `JobCompleted`.
- Changing the existing `Conditions.UpdateCondition` timestamp behaviour.

## 1. API changes (`api/v1alpha1/runner_types.go`)

`RunnerSpec`:

```go
// How long watched workloads may stay not-ready before the Runner reports Stalled.
// "0s" disables the timeout.
// +kubebuilder:default="30m"
StableStateTimeout *metav1.Duration `json:"stableStateTimeout,omitempty"`
```

`RunnerStatus`:

```go
ObservedGeneration int64        `json:"observedGeneration,omitempty"`
ProgressingSince   *metav1.Time `json:"progressingSince,omitempty"`
```

The default is applied by CRD structural defaulting, including to existing
Runners when they are read. This changes behaviour for existing Runners: any
Runner that waits through a rollout longer than 30m now reports Stalled.
`0s` disables the timeout.

## 2. Timeout clock (`RunnerReconciler`)

After collecting matched workloads, compute:

`stable = len(workloads) > 0 && every workload is ready`

A selector matching zero workloads counts as not stable.

| State | Action |
|---|---|
| not stable, `progressingSince == nil` | set `progressingSince = now` |
| not stable, `now - progressingSince >= timeout` (timeout > 0) | `Stalled=True` reason `StableStateTimeout`, message lists not-ready workloads |
| not stable, within timeout (or timeout disabled) | `Reconciling=True` reason `Progressing`; if timeout > 0 return `RequeueAfter: timeout - elapsed` |
| stable | `progressingSince = nil`; timeout Stall cleared |

`RequeueAfter` guarantees the timeout fires even when a stuck rollout emits no
further events.

When a timed-out rollout later becomes stable, the job runs normally (no
cycle is skipped).

The clock is driven by `workloadSelector` only. When `workloadSelector` is nil
there is no clock and no `StableStateTimeout` condition.

## 3. Status conditions

`RunnerReconciler` is the single writer of the kstatus conditions `Ready`,
`Reconciling` and `Stalled`, plus `status.observedGeneration`. It already
reconciles on every Runner status change (the `For(&Runner{})` watch), so it
sees `JobCompleted` updates written by `JobRunnerReconciler`.

Derivation:

| Workloads | `JobCompleted` reason | Ready | Reconciling | Stalled |
|---|---|---|---|---|
| progressing, within timeout | any | False | True `Progressing` | False |
| progressing, timed out | any | False | False | True `StableStateTimeout` |
| stable | Pending / JobNotFound / running | False | True `WaitingForJob` | False |
| stable | Successful | True | False | False |
| stable | Failed | False | False | True `JobFailed` |

For scheduled Runners the same table applies, using the CronJob's last job
status that `JobRunnerReconciler` already writes into `JobCompleted`.

Conditions are set with `k8s.io/apimachinery/pkg/api/meta.SetStatusCondition`,
which only changes `LastTransitionTime` when the status changes. The
derivation lives in a pure function
(new file `internal/status/runner_health.go`) taking the workload
readiness, `JobCompleted` condition, timeout, `progressingSince` and `now`,
and returning the conditions, the new `progressingSince` and the requeue delay.

`ReconcileSuccess` and `JobCompleted` are unchanged; `IsRunnerReady` and
`runnerReadyHandler` keep reading `JobCompleted`.

## 4. Watch predicate change

`readyPredicateFn.UpdateFunc` changes from "new object is ready" to "new object
is ready **or** readiness changed between old and new object". This lets the
controller observe ready → not-ready transitions and start the clock.

Accepted behaviour change: a rollout of a watched workload now reliably writes
`false` then `true` into the job-config ConfigMap readiness map, changing its
`ResourceVersion`, so every rollout triggers a fresh job run once it
stabilises. Previously this only happened if another reconcile coincided with
the rollout.

## 5. Argo CD health check

Ship `config/argocd/runner-health.lua`:

1. `observedGeneration < metadata.generation` → Progressing ("Waiting for
   status update")
2. `Stalled=True` → Degraded with the condition message
3. `Ready=True` → Healthy
4. otherwise → Progressing with the `Reconciling` message if present

Document the `argocd-cm` entry
`resource.customizations.health.e2e.stakater.com_Runner` in `README.md`
/ `OPERATOR.md`.

## 6. Testing

- **Unit (table-driven)**: the pure derivation function with an injected
  clock, covering every row of the tables in sections 2 and 3, including
  timeout disabled (`0s`) and zero matched workloads.
- **envtest**: Runner with a short timeout and a Deployment that never becomes
  ready → `Stalled=True/StableStateTimeout`; make it ready → `Reconciling` →
  `Ready` after the job succeeds.
- **Lua**: fixture YAMLs (healthy, progressing, degraded-timeout,
  degraded-job-failed, stale-generation) plus a Go test that runs the script
  with `gopher-lua` against each fixture, in the upstream
  `resource_customizations` test layout.
- Update e2e snapshots for the new status fields.
