# ADR-0004 — Testing & Definition of Done

- **Status:** Accepted
- **Date:** 2026-10-30 (retroactive — records the existing practice)

## Context

The method (`docs/method.md`) requires test-first development and a green,
verifiable Definition of Done. That requirement needs a concrete, enforced
mechanism, not an honour system.

## Decision

- **Test-first with envtest.** Controller behaviour is pinned with
  controller-runtime's envtest (`suite_test.go` + the `_test.go` suites in
  `internal/`), table-driven where it fits. New behaviour starts with a failing
  test; trigger-semantics changes extend `rollout_trigger_test.go` /
  `stable_state_timeout_test.go`.
- **Definition of Done.** A coding change is done only when **`make test` and
  `make lint` are green**, output actually seen. `make test` runs
  `manifests generate fmt vet` then the envtest suites; `make lint` runs
  golangci-lint (`.golangci.yml`).
- **CI is the enforcement gate.** `.github/workflows/pull_request.yaml` runs the
  shared Stakater operator workflow with `ENABLE_LINTING` and `ENABLE_UNIT_TESTS`,
  so every PR to `main`/`release` must pass lint + unit tests.

## Consequences

- "Done but unverified" cannot survive a PR — CI blocks it.
- Honest gap: enforcement is at the **CI gate**, not on every local edit. Unlike
  kube-ward there is no `make verify` target and no PostToolUse hook, so a local
  working tree can be temporarily unverified between commits. Closing this is
  Draft story **DX.1** in [docs/roadmap.md](../roadmap.md).
- `paths-ignore` means docs-only PRs skip the operator CI; that is intentional and
  keeps docs changes cheap.

## Alternatives rejected

- **Manual review as the only gate** — not reproducible; lets unverified changes
  merge.
- **Claiming a `make verify` + hook today** — would be dishonest; the target does
  not exist yet. Recorded as a gap and a story instead.
