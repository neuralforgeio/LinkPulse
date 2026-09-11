# ADR-004: JWT access tokens + rotating refresh tokens

**Status**: Accepted

## Context

Sessions must be revocable, stolen refresh tokens must be detectable, and the
dashboard needs fast per-request auth.

## Decision

15-minute stateless JWTs for requests; 30-day refresh tokens stored as SHA-256
hashes with rotation. Replaying a rotated token revokes every session of the
user. Refresh tokens travel only in HttpOnly cookies.

## Consequences

- Stateless verification per request; revocation only at the refresh layer.
- Token theft collapses the whole session family — the attacker gets nothing.
