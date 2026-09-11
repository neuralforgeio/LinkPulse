# Database

PostgreSQL 16+ · migrations via [Goose](https://github.com/pressly/goose) in `backend/migrations/`.

## Schema

| Table | Purpose |
|---|---|
| `users` | Accounts; passwords stored as Argon2id hashes |
| `refresh_tokens` | Rotating sessions; hashes only, revocable |
| `tenants` | Workspaces; unique slugs; JSONB settings |
| `memberships` | User↔tenant with role (owner/admin/member/viewer) |
| `invitations` | One-time codes with expiry and max uses |
| `links` | Short links; globally unique `short_code`; soft delete |
| `click_events` | Append-only clicks; salted IP hashes; parsed UA |
| `api_keys` | SHA-256 key hashes + prefixes + scopes |
| `audit_logs` | Append-only action history; salted IP hashes |

## Migrations

```bash
cd backend
goose -dir migrations postgres "$DATABASE_URL" up     # apply
goose -dir migrations postgres "$DATABASE_URL" down   # rollback (tested)
````

## Conventions

- UUIDs are generated in Go (`google/uuid`), not by the database.
- Every query is parameterized — no string concatenation, ever.
- `click_events` and `audit_logs` are append-only.
- Link status (active / expired / max_clicks_reached / disabled) is **computed** from columns at read time, never stored as a state machine.
