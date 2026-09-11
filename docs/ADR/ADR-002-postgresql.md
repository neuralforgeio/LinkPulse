# ADR-002: PostgreSQL as the primary database

**Status**: Accepted

## Context

Click analytics needs aggregation (window functions, FILTER), tenancy needs
relational integrity, and the deploy target is a single host.

## Decision

PostgreSQL 16+ with Goose migrations.

## Consequences

- JSONB settings, TEXT[] tags, and `COUNT(DISTINCT (ip_hash, day))` come free.
- No extra infrastructure; development runs on a native install without Docker.
