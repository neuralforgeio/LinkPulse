# Evaluation

## What Was Verified

- **Functional**: 60+ curl-driven acceptance tests across 7 milestones —
  auth (including refresh rotation and reuse detection), multi-tenancy
  (roles, last-owner protection), link CRUD, the redirect engine, the
  click pipeline (graceful-shutdown flush proven with Ctrl+C), analytics
  aggregation, API keys (scopes + revocation), rate limiting (exact
  100/10 thresholds observed), and the audit log.
- **Unit tests**: `go test ./internal/...` covers the short-code generator
  (uniformity, uniqueness), URL validation (XSS payload rejection), UTM
  merge semantics, and the rate limiter (block, reset, per-key isolation).
- **Security**: parameterized SQL everywhere; salted IP hashing verified
  directly in psql; Argon2id hashes verified never to leave the database.

## Known Gaps (honest list)

- Password-protected link verification UI (backend redirects to
  `/protected/{code}`; the page is pending).
- Bulk import/export not implemented.
- Rate limiting is per-instance — horizontal scaling requires Redis.
- Analytics time buckets are UTC days.

The suites are reproducible via `backend/scripts/test-vars.ps1` plus the
documented curl sequences.
