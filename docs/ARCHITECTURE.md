# Architecture

## System Overview

```text
Browser ──▶ Next.js dashboard (:3000) ──▶ Go API (:8080) ──▶ PostgreSQL 18
                        │                         │
                 demo mode (no backend)     click buffer (in-memory)
                                                  │
                                        batch worker (1s / 500 events)
                                                  │
                                            click_events table
```
````

## Two Traffic Paths

1. **Dashboard traffic** — authenticated users reach the Go API through the Next.js frontend (CORS-restricted to a single origin).
2. **Redirect traffic** — visitors of short links hit `GET /{code}` directly on the Go server, bypassing the frontend entirely. One hop, no framework overhead.

## Request Lifecycle (mutation example)

```text
Request → security headers → CORS → rate limit → auth (JWT / API key)
→ membership gate (tenant routes) → audit middleware → handler
→ validation → service → PostgreSQL (parameterized SQL)
→ response envelope { success, data | error }
```

## Backend Package Layout

| Package                | Responsibility                                         |
| ---------------------- | ------------------------------------------------------ |
| `internal/server`      | Router, middleware chain, route wiring                 |
| `internal/auth`        | Register/login/refresh/logout, JWT, Argon2id, rotation |
| `internal/tenant`      | Workspaces, membership, roles, invitations             |
| `internal/link`        | Link CRUD, short codes, UTM merge, soft delete         |
| `internal/redirect`    | `GET /{code}` resolution + click enqueue               |
| `internal/clickbuffer` | In-memory click pipeline, batch flush worker           |
| `internal/analytics`   | SQL aggregations for the dashboard                     |
| `internal/apikey`      | API keys: create/list/revoke, public API auth          |
| `internal/publicapi`   | `/api/v1/public/*` handlers (scope-authorized)         |
| `internal/audit`       | Append-only action log (middleware-captured)           |
| `internal/ratelimit`   | Fixed-window limiter, 5 groups                         |
| `internal/logutil`     | Colored console + daily JSON log files                 |
| `internal/httpx`       | Response envelope, error codes, UserError contract     |
| `internal/shortid`     | Base62 generator (rejection sampling, no modulo bias)  |

## Frontend

Next.js App Router, TypeScript strict. Route groups: `(auth)` for login/register/invite, `(dashboard)` for the app. Data fetching via TanStack Query. Session tokens live in memory only — never localStorage — and are recovered on reload via a silent refresh against the HttpOnly cookie.

Entrance animations are pure CSS: content visibility never depends on JavaScript hydration.
