# ADR-0002 — Readiness-gated triggering; a job runs once per `runVersion`

- **Status:** Accepted
- **Date:** 2026-10-30 (retroactive — records the existing design)

## Context

Hestia's purpose is to verify a workload *after it changes and is genuinely
healthy* — not on a timer, and never mid-rollout. Two problems follow: (1) how to
avoid running while anything is not-ready, and (2) how to run **exactly once** per
meaningful change, without re-running on every reconcile or operator restart.

## Decision

- **Readiness gate.** `JobRunnerReconciler` refuses to run while any entry in the
  job-config ConfigMap is `"false"` ([job_runner_controller.go](../../internal/controller/job_runner_controller.go)).
  A workload's readiness, "at least one workload matched" (`workloadMinimum`), and
  upstream-Runner success (`runnersCountMinimum`) are all encoded as such entries.
- **Run identity = `runVersion`.** The ConfigMap carries a `runVersion` that bumps
  **only when some other entry changed** (`nextRunVersion` / `runChanged` in
  [job_config.go](../../internal/resources/job_config.go)). A Job is created once
  per `runVersion` and labelled `runner.stakater.com/version`; an existing Job for
  the current version is reused, never duplicated ([job.go](../../internal/resources/job.go)).
- Rollout-capable workloads also contribute a `<key>.generation` entry, so a
  rollout that finishes before the workload is ever observed not-ready still
  changes the content and triggers a fresh run.

## Consequences

- Re-runs are precise and explainable: a rollout, a scale, a selector change, or a
  Runner change each alter the ConfigMap content → new `runVersion` → one new Job.
- An operator restart or idempotent resync rewrites identical content → no re-run.
- A pod crash *without* a rollout re-runs only if the operator observes the
  workload go not-ready — an honest limitation, documented in the README and
  [docs/how-it-works.md](../how-it-works.md#when-a-job-runs).
- This logic is load-bearing and easy to break subtly; it is covered by
  `rollout_trigger_test.go` and must change together with those tests and the docs.

## Alternatives rejected

- **Hash of the whole spec/status** — opaque; couldn't express "bump only on a
  real change" cleanly and made debugging a spurious re-run hard.
- **Time-based (cron-only) triggering** — can't express "verify this change"; cron
  remains available via `spec.schedule` as an *additional* mode, not the default.
- **Re-run on every reconcile** — correct-ish but wasteful and noisy against the
  application under test.
