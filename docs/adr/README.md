# Architecture Decision Records

Each ADR records one decision: its context, what we decided, the consequences
(including costs), and the alternatives we rejected and why.

**Convention:** ADRs are **immutable once accepted**. Do not edit a decided ADR to
reflect a changed mind — write a new ADR that *supersedes* it, and update the
status of the old one. This keeps the *evolution* of decisions legible.

Format: Title · Status · Date · Context · Decision · Consequences · Alternatives.

These initial ADRs are **retroactive**: they record decisions the codebase already
embodies, so the *why* is captured before the code is changed again (see
[docs/method.md](../method.md)).

## Index

| # | Decision | Status |
|---|----------|--------|
| [0001](0001-two-controllers-configmap-handshake.md) | Two controllers coordinating through a job-config ConfigMap handshake | Accepted |
| [0002](0002-readiness-gated-run-once-per-runversion.md) | Readiness-gated triggering; a job runs once per `runVersion` | Accepted |
| [0003](0003-kstatus-health-and-argocd-lua.md) | kstatus conditions as the Runner's health, with a shipped Argo CD health check | Accepted |
| [0004](0004-testing-and-definition-of-done.md) | Testing & Definition of Done — envtest, `make test` + `make lint`, CI gate | Accepted |

## Candidates not yet written

Recorded so they aren't lost; write them when they become contentious:

- **Dynamic informers vs. static Watches** — whether to watch workload kinds
  dynamically (cf. the `dynamic-informers` branch) instead of the fixed
  `For`/`Watches` set in `SetupWithManager`.
- **Job retention / TTL policy** — how long completed Jobs linger (cf. the
  `fix/remove-job-ttl` history); the trade-off between re-run noise and audit trail.
- **`make verify` + PostToolUse hook** — moving the DoD from the CI gate to every
  local edit, as in kube-ward (tracked as roadmap story DX.1).
