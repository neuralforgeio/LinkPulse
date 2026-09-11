# ADR-001: Modular monolith instead of microservices

**Status**: Accepted

## Context

LinkPulse needs auth, tenancy, links, analytics, and a public API — deployable on a laptop.

## Decision

A single Go binary with strict internal package boundaries (`internal/*`).

## Consequences

- One process to run, monitor, and deploy.
- Boundaries are enforced by import rules, not network hops.
- Extraction into services remains possible — every package owns its SQL.
