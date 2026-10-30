# ADR-0001 — Two controllers coordinating through a job-config ConfigMap handshake

- **Status:** Accepted
- **Date:** 2026-10-30 (retroactive — records the existing design)

## Context

Hestia must decide *whether* a verification job should run (based on the live
readiness of many watched workloads) and separately *run and track* that job.
Mixing both concerns in one reconcile loop couples "observe the world" with
"manage a Job/CronJob" and makes the trigger logic hard to test in isolation.

## Decision

Split the work across **two controllers** that communicate through a single
per-Runner **job-config ConfigMap** (named after the Runner):

- `RunnerReconciler` ([internal/controller/runner_controller.go](../../internal/controller/runner_controller.go))
  watches the `Runner`, the workload kinds, and other Runners; it writes the
  readiness/generation data into the ConfigMap and computes Runner health.
- `JobRunnerReconciler` ([internal/controller/job_runner_controller.go](../../internal/controller/job_runner_controller.go))
  watches that ConfigMap and owns the `Job`/`CronJob` lifecycle.

The ConfigMap is the explicit, inspectable contract between them.

## Consequences

- The trigger decision is reduced to data in a ConfigMap, which is easy to assert
  in envtest and easy to debug by `kubectl get configmap`.
- Each controller stays small and single-purpose.
- Cost: an extra Kubernetes object per Runner, and a two-hop latency (workload
  event → ConfigMap write → Job creation).
- The ConfigMap is owned by the Runner (controller reference), so it is garbage
  collected with the Runner.

## Alternatives rejected

- **One reconciler doing everything** — simpler object graph, but entangles
  observation and Job management and makes the trigger untestable without a real
  Job controller in the loop.
- **Status-subresource-only coordination** — the job-run decision would live in
  Runner `.status`, conflating desired/observed state and making the handshake
  invisible to operators debugging a cluster.
