# ADR-0003 — kstatus conditions as the Runner's health, with a shipped Argo CD health check

- **Status:** Accepted
- **Date:** 2026-10-30 (retroactive — records the existing design)

## Context

A verification result is only useful if the surrounding platform can act on it.
GitOps tools (Argo CD) and `kubectl wait` need a standard, machine-readable notion
of a resource's health. Hestia-specific condition types alone (`ReconcileSuccess`,
`JobCompleted`) don't plug into those tools.

## Decision

Summarise the Runner as standard
[kstatus](https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus)
conditions — `Ready`, `Reconciling`, `Stalled` — derived in
[internal/status/runner_health.go](../../internal/status/runner_health.go), on top
of the existing domain conditions. Ship an Argo CD health check
([config/argocd/e2e.stakater.com/Runner/health.lua](../../config/argocd/e2e.stakater.com/Runner/health.lua))
that maps `Stalled=True` → Degraded, `Ready=True` → Healthy, else Progressing.
A `stableStateTimeout` (default `30m`) bounds how long watched workloads may stay
not-ready before the Runner reports `Stalled`.

## Consequences

- `kubectl wait --for=condition=Ready runner/...` works with no extra glue.
- Argo CD renders Runner health natively once the Lua check is configured, so a
  stalled rollout shows as Degraded in the GitOps UI.
- Hestia now owns a kstatus contract: the truth table
  ([README](../../README.md#stable-state-timeout)) is part of the public API and
  changes to it are API changes.
- Cost: health derivation is extra logic that must stay consistent with the
  trigger logic (ADR-0002).

## Alternatives rejected

- **Domain conditions only** — no standard health; every consumer writes its own
  interpretation.
- **A separate health CRD / metric only** — heavier, and still wouldn't satisfy
  `kubectl wait` or Argo CD's per-kind health model.
