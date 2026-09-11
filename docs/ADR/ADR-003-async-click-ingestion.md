# ADR-003: Asynchronous buffered click ingestion

**Status**: Accepted

## Context

Redirects must never wait for analytics writes (PRD 9.5.4).

## Decision

Redirects enqueue click events into an in-memory channel; a worker flushes
batches (500 events or every second) inside one transaction that also bumps
`links.click_count`. Graceful shutdown drains the buffer before exit.

## Consequences

- Redirect latency is independent of database load.
- A full buffer drops events with a logged warning — never blocks a redirect.
- Ctrl+C during traffic loses zero clicks (verified in testing).
