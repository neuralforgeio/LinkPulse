<div align="center">

<!-- ![Logo](frontend/app/icon.svg) -->
<img src="frontend/app/icon.svg" width="23%">

# 🔗 LinkPulse

**Self-hosted, multi-tenant URL shortener with real click analytics.**

![Version](https://img.shields.io/badge/version-1.2.0-2563eb)
![License](https://img.shields.io/badge/license-MIT-22c55e)
![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go)
![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=nextdotjs)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?logo=postgresql)

_Shorten links. Track every click. Own your data._

[Features](#-features) · [Architecture](#-architecture) · [Quick Start](#-quick-start) · [API](#-api-overview)

</div>

---

## Overview

LinkPulse is a production-grade URL shortener built as a modular monolith: a **Go** backend, a **Next.js** dashboard, and **PostgreSQL**. Every redirect is recorded asynchronously — referrers, devices, browsers, campaigns, unique-visitor estimates — without ever storing a raw IP address.

Built to be owned: self-hosted, MIT licensed, zero vendor lock-in.

## ✨ Features

- **Email OTP login** — two-step login with a 6-digit code (Resend, with dev-log fallback)
- **Password reset via email OTP** — one-time codes, 10-minute expiry, session revocation
- **Short links** — custom aliases, random base62 codes, expiry, click limits, password protection, UTM builder
- **Link detail page** — per-link overview, analytics charts, QR code, and inline settings
- **Bulk operations** — create up to 100 links per request, import/export CSV (500 rows)
- **Async click tracking** — redirects never wait for analytics: in-memory buffer, batch inserts, graceful-shutdown flush
- **Analytics** — clicks over time, unique visitor estimates (salted IP hashes), top referrers / devices / browsers / OS / campaigns
- **Multi-tenant workspaces** — invite codes, four roles (owner / admin / member / viewer) enforced server-side on every route
- **Public REST API** — scoped API keys (`links:read`, `links:write`, `analytics:read`), keys hashed at rest, shown exactly once
- **Security** — Argon2id password hashing, JWT access tokens + rotating refresh tokens with reuse detection, rate limiting, append-only audit log, parameterized SQL everywhere
- **Dashboard** — dark / light / system themes, fully responsive, QR codes, CSV-ready data
- **Demo mode** — explore the full dashboard with realistic sample data at `/demo`, no account or backend needed

## 🏗 Architecture

```text
Browser ──▶ Next.js dashboard (:3000) ──▶ Go API (:8080) ──▶ PostgreSQL 18
                        │                         │
                 demo mode (no backend)     click buffer (in-memory)
                                                  │
                                        batch worker (1s / 500 events)
                                                  │
                                            click_events table
```

- **Redirects bypass the dashboard entirely** — visitors hit the Go server directly (`GET /{code}` → 302)
- **Clicks are buffered**, never blocking a redirect; the buffer is flushed on every graceful shutdown — zero clicks lost
- **Modular monolith** — one deployable backend with clear package boundaries (`internal/auth`, `internal/link`, `internal/analytics`, `internal/apikey`, `internal/audit`, …)

## 🧰 Tech Stack

| Layer    | Technology                                                                                      |
| -------- | ----------------------------------------------------------------------------------------------- |
| Backend  | Go 1.27 · chi · pgx/v5 · slog                                                                   |
| Frontend | Next.js 16 (App Router) · TypeScript · Tailwind CSS · TanStack Query · Recharts · framer-motion |
| Database | PostgreSQL 16+ · Goose migrations                                                               |
| Auth     | JWT access tokens · rotating refresh tokens · Argon2id                                          |

## 🚀 Quick Start

**Prerequisites:** Go 1.27+, Node.js 20+, PostgreSQL 16+, Goose CLI.

```bash
# 1. Backend
cd backend
cp .env.example .env                # set DATABASE_URL + JWT_SECRET
goose -dir migrations postgres "postgres://user:pass@localhost:5432/linkpulse?sslmode=disable" up
go mod tidy
go run ./cmd/server                 # → http://localhost:8080

# 2. Frontend
cd ../frontend
cp .env.example .env.local
npm install
npm run dev                         # → http://localhost:3000
```

## ⚙️ Environment Variables

### Backend (`backend/.env`)

| Variable                                                                   | Default                 | Description                                                           |
| -------------------------------------------------------------------------- | ----------------------- | --------------------------------------------------------------------- |
| `DATABASE_URL`                                                             | _required_              | PostgreSQL connection string                                          |
| `JWT_SECRET`                                                               | _required_              | Token signing secret                                                  |
| `APP_PORT`                                                                 | `8080`                  | HTTP port                                                             |
| `FRONTEND_ORIGIN`                                                          | `http://localhost:3000` | Allowed CORS origin                                                   |
| `CLICK_SALT`                                                               | random per boot         | Salt for IP hashing                                                   |
| `CLICK_BUFFER_SIZE` / `CLICK_FLUSH_INTERVAL_MS` / `CLICK_FLUSH_BATCH_SIZE` | 5000 / 1000 / 500       | Click pipeline tuning                                                 |
| `RATE_LIMIT_ENABLED` + `RATE_LIMIT_*_PER_MINUTE`                           | see `.env.example`      | Redirect 100 · login 10 · register 5 · public API 300 · dashboard 600 |
| `LOG_DIR` / `LOG_FORMAT`                                                   | `logs` / `pretty`       | Daily JSON log files + colored console                                |

### Frontend (`frontend/.env.local`)

| Variable                   | Default                 | Description                   |
| -------------------------- | ----------------------- | ----------------------------- |
| `NEXT_PUBLIC_API_BASE_URL` | `http://localhost:8080` | Backend base URL              |
| `NEXT_PUBLIC_DEMO_MODE`    | `false`                 | Mock data — no backend needed |

## 📡 API Overview

| Area       | Endpoints                                                                       |
| ---------- | ------------------------------------------------------------------------------- |
| Auth       | `POST /api/v1/auth/register · login · refresh · logout` · `GET /me`             |
| Workspaces | `POST/GET /tenants` · members & roles · invitations · leave                     |
| Links      | `POST/GET/PATCH/DELETE /tenants/{id}/links` — search, filters, sort, pagination |
| Links      | `POST /tenants/{id}/links/bulk` (100 max) · `POST /links/import` · `GET /links/export?format=csv\|json` |
| Analytics  | `GET /tenants/{id}/analytics/overview` — time series + top rankings             |
| Analytics  | `GET /tenants/{id}/links/{id}/analytics` · `GET /tenants/{id}/links/{id}/clicks` |
| API keys   | `POST/GET/DELETE /tenants/{id}/api-keys` — scoped, one-time reveal              |
| Public API | `/api/v1/public/links` — Bearer `lp_live_…`, scope-enforced                     |
| Redirect   | `GET /{code}` → `302` · `410` for expired / disabled / limit-reached            |
| Audit      | `GET /tenants/{id}/audit-logs` — append-only history                            |
| Health     | `GET /healthz` · `GET /readyz`                                                  |

## 🧪 Testing

```bash
cd backend
go test ./internal/...    # shortid, URL validation, UTM merge, rate limiter
```

## 🗺 Roadmap

- [v] Core: auth · multi-tenancy · links · redirect · analytics · API keys · audit · rate limiting
- [v] Password-protected links with 6-digit OTP + email reset flow
- [v] Email OTP login (Resend)
- [v] Settings (General, Profile, Members, API Keys, Audit)
- [v] Bulk import/export CSV + link detail page + demo mode
- [v] Per-link analytics with recent clicks feed
- [] Redis-backed distributed rate limiting

## ⚠️ Known Limitations

- Analytics time buckets are UTC days
- Rate limiting is per-instance (in-memory); use `RATE_LIMIT_ENABLED=false` to disable
- Unique visitors are estimates derived from salted IP hashes
- Bulk create is transactional per item, not per batch — a partial failure leaves successful rows in place

## 📄 License

MIT — see [LICENSE](LICENSE).
