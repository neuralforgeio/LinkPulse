# API Reference

Base URL: `http://localhost:8080/api/v1`. Every response uses one envelope:

```json
{ "success": true,  "data": { } }
{ "success": false, "error": { "code": "…", "message": "…", "details": [] } }
```

## Authentication

| Method                                 | Use                          |
| -------------------------------------- | ---------------------------- |
| `Authorization: Bearer <access_token>` | User session (15-minute JWT) |
| `Authorization: Bearer lp_live_…`      | Public API (API key)         |
| `lp_refresh_token` HttpOnly cookie     | Refresh / logout             |

## Error Codes

`VALIDATION_ERROR · BAD_REQUEST · CONFLICT · FORBIDDEN · NOT_FOUND ·
UNAUTHORIZED · INVALID_TOKEN · INVITE_EXPIRED · RATE_LIMITED · INTERNAL_ERROR`

Rate-limited responses carry `X-RateLimit-Limit / -Remaining / -Reset` and `Retry-After`.

## Auth

| Endpoint              | Notes                                                   |
| --------------------- | ------------------------------------------------------- |
| `POST /auth/register` | Creates user + default workspace (owner)                |
| `POST /auth/login`    | Returns the access token; sets the refresh cookie       |
| `POST /auth/refresh`  | Rotates the refresh token — reuse revokes every session |
| `POST /auth/logout`   | Revokes the session, clears the cookie                  |
| `GET /auth/me`        | User + workspaces + roles                               |

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"Pass1234"}'
```

## Workspaces

`POST /tenants` · `GET /tenants` · `GET|PATCH /tenants/{id}` ·
`GET /tenants/{id}/members` · `PATCH /tenants/{id}/members/{userId}` ·
`DELETE /tenants/{id}/members/{userId}` · `POST /tenants/{id}/leave` ·
`POST /tenants/{id}/invitations` (one-time code, 72h) · `POST /invitations/accept`

Roles: owner / admin / member / viewer — enforced server-side on every route.

## Links

`POST|GET /tenants/{id}/links` · `GET|PATCH|DELETE /tenants/{id}/links/{linkId}`

List filters: `page, page_size (≤100), search, status, tag, sort_by, sort_order, start_date, end_date`.

```bash
curl -X POST http://localhost:8080/api/v1/tenants/$TENANT/links \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"destination_url":"https://example.com","custom_code":"promo",
       "utm":{"source":"instagram","medium":"social"}}'
```

## Analytics

`GET /tenants/{id}/analytics/overview?start_date=&end_date=&granularity=day|week|month`

## API Keys

`POST|GET /tenants/{id}/api-keys` · `DELETE /tenants/{id}/api-keys/{keyId}`
The raw key appears exactly once — at creation.

## Public API (API keys only)

| Endpoint                                              | Scope            |
| ----------------------------------------------------- | ---------------- |
| `POST /public/links`                                  | `links:write`    |
| `GET /public/links` · `GET /public/links/{shortCode}` | `links:read`     |
| `PATCH · DELETE /public/links/{shortCode}`            | `links:write`    |
| `GET /public/links/{shortCode}/analytics`             | `analytics:read` |

```bash
curl http://localhost:8080/api/v1/public/links \
  -H "Authorization: Bearer lp_live_YOUR_KEY"
```

## Redirect · Audit · Health

`GET /{code}` → `302` (or `410` expired / disabled / limit-reached, `404` unknown) ·
`GET /tenants/{id}/audit-logs` — append-only history ·
`GET /healthz` · `GET /readyz`
