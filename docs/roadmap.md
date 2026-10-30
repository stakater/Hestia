# Roadmap

> Status: Draft for review · Owner: Stakater Platform Engineering

Hestia is built story by story (see [docs/method.md](method.md)). This file is the
**single backlog**: every change traces back to a story here, and a story is built
only after it reaches 👍 **Approved**.

## How to read this

**House story format:**

> **Story `<code>`** — *As a `<role>`, I want `<capability>`, so that `<benefit>`.*
> **Done:** `<the observable, demo-able outcome that means this is finished>`
> **Status:** 📝 Draft | 👍 Approved | 🔨 In progress | ✅ Done
> **Plan:** `docs/planning/<slug>.md` (once Approved)

**Lifecycle gate** ([method](method.md#story-lifecycle)): 📝 Draft (written, *not*
agreed — killable) → 👍 Approved (a human said build it) → 🔨 In progress (one at a
time) → ✅ Done (Done line met, `make test` + `make lint` green, plan linked).

> The stories below are **Draft seeds** that illustrate the format and capture real,
> visible gaps in the repo. They are deliberately *not* Approved — treat them as a
> starting point to accept, re-scope, or kill, not a committed plan.

---

## Developer experience (DX)

**Story DX.1** — *As a contributor, I want `make verify` and a local
verify-on-edit hook, so that "done but unverified" can't exist between commits —
not just at the CI gate.*
**Done:** `make verify` chains `fmt vet build lint test`; a documented Claude Code
PostToolUse hook runs it after edits; [ADR-0004](adr/0004-testing-and-definition-of-done.md)
and [method.md](method.md) updated to cite the hook as the enforcement point.
**Status:** 📝 Draft

**Story DX.2** — *As a new operator user, I want the empty `OPERATOR.md` either
filled with operations guidance or removed, so that I'm not misled by a stub.*
**Done:** `OPERATOR.md` either contains real OLM/operations content (and is linked
from the docs index) or is deleted; no empty tracked docs remain.
**Status:** 📝 Draft

---

## Observability (OBS)

**Story OBS.1** — *As a platform operator, I want Prometheus metrics for Runner job
outcomes (runs, successes, failures, stalls), so that I can alert and graph
verification health alongside my other signals.*
**Done:** the controller exports counters/gauges for run outcomes via the existing
`config/prometheus` wiring; a sample dashboard/query is documented; metrics names
are covered by a test. Needs an ADR deciding the metric names and labels.
**Status:** 📝 Draft

---

## Triggering & engine (ENG)

**Story ENG.1** — *As a user watching many workload kinds, I want Hestia to watch
them via dynamic informers rather than a fixed `For`/`Watches` set, so that new
or CRD-based workload kinds can be targeted without a code change.*
**Done:** workload kinds are discovered/configured at runtime; the existing fixed
set keeps working; trigger-semantics tests still green. Blocked on an ADR (see
[adr/README.md](adr/README.md#candidates-not-yet-written), "Dynamic informers").
Relates to the `dynamic-informers` branch.
**Status:** 📝 Draft

**Story ENG.2** — *As a cluster admin, I want a documented, configurable retention
policy for completed Runner Jobs, so that I can trade re-run noise against keeping
an audit trail.*
**Done:** job retention behaviour is specified in an ADR and configurable on the
Runner; `how-it-works.md` documents it; regression test pins it. Relates to the
`fix/remove-job-ttl` history.
**Status:** 📝 Draft

---

## Documentation (DOC)

**Story DOC.1** — *As a new user, I want getting-started, concept, and comparison
docs plus a repo development method, so that I can adopt and contribute to Hestia
without git archaeology.*
**Done:** `docs/` contains getting-started, how-it-works, comparison guides, the
method, ADRs, and this roadmap; README links them.
**Status:** 🔨 In progress (this documentation effort)
