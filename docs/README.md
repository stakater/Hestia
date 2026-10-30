# Hestia Documentation

Hestia is a Kubernetes operator that **runs a verification Job every time your
workloads roll out, scale, or otherwise settle into a ready state** — and reports
the verdict as a Kubernetes-native health status that GitOps tools understand.

> In Greek mythology Hestia is the goddess of the hearth and of stability. The
> operator plays the same role for your apps: it confirms the system is *truly
> ready* before anything moves forward.

## Start here

| If you want to… | Read |
|---|---|
| Run your first Runner in ~5 minutes | [Getting started / Hello World](getting-started.md) |
| Understand how it actually works (and when a job reruns) | [How it works](how-it-works.md) |
| See how Hestia compares to CronJobs, CI/CD, Argo hooks, Testkube… | [Comparison with alternatives](comparison-with-alternatives.md) |
| Decide between Hestia and kube-health (KHI) | [Hestia vs kube-health](hestia-vs-kube-health.md) |
| Install for production via OLM | [README » Installation](../README.md#installation--deployment) |
| Contribute, and understand *how we build Hestia* | [The Hestia Method](method.md) |

## How we build Hestia (contributors)

Hestia follows a customized, BMAD-derived development method — the same one we run
in our other operator projects. If you're changing code, start here:

- [The Hestia Method](method.md) — vision → story → plan → red → green → verify →
  review, the lifecycle gate, and where each practice is enforced.
- [Roadmap & stories](roadmap.md) — the single backlog; every change traces to a story.
- [Architecture Decision Records](adr/) — immutable decisions and their rejected alternatives.
- [Planning-doc template](planning/TEMPLATE.md) — the context capsule each story gets.
- [CLAUDE.md](../CLAUDE.md) — repo invariants, architecture map, and the workflow in brief.

## The one-paragraph pitch

Most teams verify a deployment from inside their CI/CD pipeline — a test step that
runs after the deploy step. That works until something changes the cluster *outside*
that pipeline: a GitOps sync, an autoscaler scaling the app, a `kubectl rollout
restart`, a manual change. Hestia lives **in the cluster** and reacts to the live
state of your workloads, so a verification job fires on **every** path into "ready",
regardless of who caused it — and only when the workload is genuinely healthy, never
mid-rollout.

## What Hestia is — and isn't

**It is:** a change-triggered, readiness-gated Job runner with GitOps-readable
health (kstatus + an Argo CD health check). Good for post-deploy smoke/E2E
verification in clusters where deploys don't all flow through one pipeline you
control.

**It isn't:** a general job scheduler, a workflow/DAG engine (chaining is one level
deep), a test-reporting platform (you own reporting inside the Job), or a periodic
synthetic-monitoring tool (that's closer to [kube-health](hestia-vs-kube-health.md)).

> **Maturity:** `v1alpha1`, early (`v0.0.x`). The API can still change — pin
> versions and expect churn if you adopt it now.
