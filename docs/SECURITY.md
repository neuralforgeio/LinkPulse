# Security

## Authentication

- Passwords: **Argon2id** (OWASP parameters); never stored or logged in plaintext.
- Access: 15-minute **JWT** (HS256, algorithm pinned on verification).
- Refresh: 30-day **rotating** tokens stored as SHA-256 hashes; replaying a rotated token revokes every session of that user.
- Cookies: `HttpOnly`, `SameSite=Lax`, `Secure` outside development.

## Privacy

Visitor IPs are never stored raw — click events and audit logs keep only
salted SHA-256 hashes (`CLICK_SALT`). Unique-visitor metrics are derived
from `(ip_hash, day)` pairs.

## API Keys

`lp_live_` + 32 base62 chars; only the SHA-256 hash is stored; shown exactly
once at creation; scope-enforced (`links:read`, `links:write`, `analytics:read`).

## Rate Limiting (PRD 9.9)

| Group | Limit | Key |
|---|---|---|
| Redirect | 100/min | IP |
| Login | 10/min | IP (failed attempts count) |
| Register | 5/min | IP |
| Public API | 300/min | API key |
| Dashboard | 600/min | user |

## Audit

Append-only action log covering every mutation — session and API-key traffic
alike. Entries can be read, never edited or deleted.

## Headers

`X-Content-Type-Options: nosniff` · `X-Frame-Options: DENY` ·
`Referrer-Policy: strict-origin-when-cross-origin` are set by the API.
HSTS and CSP are enforced at the hosting edge in production.

## SQL Injection

All queries are parameterized. Dynamic filters (links, analytics, audit)
compose only whitelisted fragments — user input always travels as bound
parameters.
