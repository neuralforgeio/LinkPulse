# ADR-005: In-memory rate limiting (Redis optional)

**Status**: Accepted

## Context

Rate limits are mandatory (PRD 9.9) but the system must run without Redis.

## Decision

A fixed-window in-memory limiter behind a `Middleware` interface. Five groups:
redirect (100/min/IP), login (10/min/IP), register (5/min/IP), public API
(300/min/key), dashboard (600/min/user). `RATE_LIMIT_ENABLED=false` disables all.

## Consequences

- Zero extra infrastructure for single-instance deployments.
- Horizontal scaling swaps the implementation for Redis — the interface stays.
- Security headers are set by the API; HSTS/CSP belong to the hosting edge.
