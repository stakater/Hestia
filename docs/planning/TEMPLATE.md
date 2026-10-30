# <Story code> — <short title>

<!--
The planning doc is a CONTEXT CAPSULE (see docs/method.md): a cold session with no
memory and no conversation history must be able to execute this story from this
file alone, without re-scouting the codebase. Fill every section; write "none"
rather than deleting one — an explicit "none" is information, an absent section is
a question. Delete this comment block in the real doc.
-->

Story: [`docs/roadmap.md`](../roadmap.md) <Story code> (status must be 👍 Approved
before this doc is implemented). Driver: <the user report, review finding, or need
that prompted it — cite ids/links>.

## Why (one paragraph)

<What breaks or is missing today, for whom, and why now.>

## Governing constraints

<The ADRs and CLAUDE.md invariants this change must respect, by number, with one
clause each on how they bite here. E.g. "ADR-0002: a job runs once per runVersion —
the new field must feed the readiness map, not spawn a second job path.">

## Design

<The change at seam level: which packages and files (exact paths under
api/v1alpha1, internal/controller, internal/resources, internal/status). New
spec fields or types shown as snippets. Wiring: what SetupWithManager / the
reconcilers connect. State explicitly what is NOT changing. Add a Mermaid
sequence or flow diagram if the control flow is non-obvious.>

## Tests (red first)

<The failing tests to write BEFORE implementing: test file, seam, and the
behaviour each pins — including the negative/regression cases that must stay
green. Trigger-semantics changes MUST add cases to
internal/controller/rollout_trigger_test.go and/or stable_state_timeout_test.go.
If a real cluster bug motivated the story, reproduce it as an envtest fixture.>

## Out of scope

<What this story deliberately does not do, and where that work lives instead (a
follow-up story code, an ADR to write first, or "accepted limitation" with the
honest consequence stated).>

## Done means

<Restate the roadmap Done line, plus the mechanics: make test + make lint green;
`make manifests` run if the CRD changed; docs updated (name which — README "When a
job runs", docs/how-it-works.md, an ADR); regression test named if this is a fix.>
