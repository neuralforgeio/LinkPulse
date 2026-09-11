# Deployment

LinkPulse adapts to any hosting setup through environment variables alone —
no code changes between environments. Configure four variables, deploy, done.

## The Four Variables

| Side     | Variable                   | Controls                                                                                                  |
| -------- | -------------------------- | --------------------------------------------------------------------------------------------------------- |
| Backend  | `FRONTEND_ORIGIN`          | CORS allowlist (comma-separated); the last entry is the public frontend used for visitor-facing redirects |
| Backend  | `APP_BASE_URL`             | Base URL of generated `short_url` values                                                                  |
| Backend  | `COOKIE_SAMESITE`          | `lax` — frontend & API share a domain; `none` — different domains (forces `Secure`)                       |
| Frontend | `NEXT_PUBLIC_API_BASE_URL` | Where the dashboard sends API calls                                                                       |

> `NEXT_PUBLIC_*` variables are baked into the JS bundle at **build time** —
> changing them requires a redeploy. Backend variables apply on restart.

## Topologies

### 1. Local development

```env
# backend/.env
APP_BASE_URL=http://localhost:8080
FRONTEND_ORIGIN=http://localhost:3000
COOKIE_SAMESITE=lax
```

```env
# frontend/.env.local
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

### 2. Tunnel (ngrok / Cloudflare Tunnel)

Frontend on Vercel, backend tunneled from your machine:

```env
# backend/.env
APP_BASE_URL=https://your-name.ngrok.app
FRONTEND_ORIGIN=http://localhost:3000,https://linkpulse.vercel.app
COOKIE_SAMESITE=none
```

```env
# Vercel → Environment Variables
NEXT_PUBLIC_API_BASE_URL=https://your-name.ngrok.app
```

Use a **static tunnel domain** so the URL never rotates (see below).

### 3. Production — one domain (recommended)

Dashboard and API behind the same registrable domain
(`app.example.com` + `api.example.com` are both `example.com`):

```env
APP_BASE_URL=https://api.example.com
FRONTEND_ORIGIN=https://app.example.com
COOKIE_SAMESITE=lax
```

Cookies are first-party — works in every browser, forever.

### 4. Production — separate domains

```env
APP_BASE_URL=https://api.linkpulse.dev
FRONTEND_ORIGIN=https://linkpulse.vercel.app
COOKIE_SAMESITE=none
```

Note: cross-domain cookies are third-party cookies. Safari blocks them;
Chrome/Edge/Firefox currently allow `SameSite=None; Secure`. For
long-term reliability prefer topology 3.

## ngrok: Free Static Domain

Free ngrok accounts include one static domain — claim it once and the
tunnel URL never changes again:

1. **dashboard.ngrok.com → Domains → New Domain** (e.g. `your-name.ngrok.app`)
2. Start the tunnel with it:
   ```bash
   ngrok http 8080 --domain=your-name.ngrok.app
   ```
3. Set `APP_BASE_URL` (backend) and `NEXT_PUBLIC_API_BASE_URL` (Vercel) to
   that domain once — no more redeploying on every tunnel restart.

## What Never Changes

Same binary, same bundle, same migrations — every deployment difference
lives in the four variables above.
