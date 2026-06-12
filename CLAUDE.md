# Mail Tracker — CLAUDE.md

Internal tool for a team of fewer than 20 people to track Outlook email relevant to them.
Each user logs in with Microsoft, pulls their own mail, and configures their own filters and AI key.

## Stack

| Layer | Tech |
|---|---|
| Backend | Go + Gin, Clean Architecture |
| Database | PostgreSQL 16 |
| Frontend | Vue 3 + Vite + TypeScript |
| Auth | Microsoft Entra ID (OAuth 2.0 + OIDC, Confidential Client) |
| Infra | Docker Compose + Nginx, Azure VM |

## Directory structure

```
src/
├── docker-compose.yml          # local dev — bind mount + hot reload
├── docker-compose.prod.yml     # production — multi-stage build
├── .env / .env.example
├── backend/
│   ├── internal/
│   │   ├── domain/             # structs, interfaces, errors — zero deps
│   │   ├── app/                # use cases — orchestrate domain
│   │   ├── infra/              # implement interfaces: DB, Graph, AI, crypto
│   │   └── handler/            # Gin handlers + middleware
│   ├── migrations/             # golang-migrate SQL files
│   ├── Dockerfile
│   └── .air.toml               # hot reload config
├── frontend/
│   ├── src/
│   │   ├── stores/             # Pinia
│   │   ├── views/              # Login, Dashboard, Settings
│   │   └── components/
│   └── Dockerfile
└── nginx/
    ├── nginx.dev.conf
    └── nginx.prod.conf
```

## Dependency rule (STRICT — do not violate)

```
domain ← app ← infra / handler
```
- `domain`: imports nothing outside stdlib
- `app`: imports only `domain` — never `infra` or `handler`
- `infra`: implements interfaces defined in `domain`
- `handler`: calls use cases only — never calls DB or infra directly

## Auth — invariant rules

- OAuth flow: Authorization Code + PKCE (Confidential Client, uses `AZURE_CLIENT_SECRET`)
- Browser stores only `session_id` cookie — HttpOnly, Secure, SameSite=Lax
- Backend stores OAuth tokens in DB, encrypted AES-256 using `AES_KEY`
- Token auto-refresh when < 5 minutes remaining
- Refresh fail → set `needs_reauth=true` → return 401 `{"error":"reauth_required"}`
- Never send access token or refresh token to the browser

## AI tiers

| Tier | Condition | Action |
|---|---|---|
| 3 — BYOK | User has API key | Call OpenAI / OpenRouter |
| 2 — PA | User enables PA fallback | Call Power Automate webhook (POC) |
| 1 — No AI | Nothing configured | Set `summary_status = "no_key"` |

Fallback runs in Job 2 — graceful degradation only, no panics or errors.

## Background jobs — every 30 minutes

- **Job 1** — Sync mail: Graph API for all users with valid session (within 7 days)
- **Job 2** — AI summary: threads with `summary_status=pending`, run fallback chain
- After Job 1 completes → publish `sync_complete` to SSE broker → browser re-renders

## SSE Broker

```
SyncJob → Broker (map[userID]chan Event) → SSE Handler → Browser
```
Plain Go channels only — no WebSocket, no Redis.

## Thread branching — Phase 1

Sort messages by `receivedDateTime asc`. Phase 2 (`In-Reply-To` tree) only when user feedback demands it.

## Skills

Skills live in `.claude/skills/` — use them for repeatable tasks, do not rewrite inline:

| Skill | Invoke | Use when |
|---|---|---|
| `run-dev` | `/run-dev` | Start local Docker dev environment |
| `db-migrate` | `/db-migrate [name]` | Create or run DB migrations |
| `test` | `/test` | Run Go tests + Vitest |
| `deploy` | `/deploy` | Deploy to Azure VM (production) |
| `graph-api` | `/graph-api [topic]` | Look up Microsoft Graph API reference |
| `git-commit` | `/git-commit <task_id> <summary>` | Commit + push after completing a WBS task |

**Commit convention:** `<task_id>: <summary>` — e.g. `P01_03: add docker-compose.yml for local dev`
After commit → update task status in `wbs/tasks.md` to `done`.

## Behavior rules

**Before coding any task:**
1. Read `wbs/tasks.md` — understand current task
2. Read `docs/SPEC.md` — if task touches architecture
3. **Web search current docs** for any lib/framework being used:
   - Search "{lib} {version} changelog" or "{lib} latest docs 2026"
   - Never assume API signatures from training data — always verify
   - Especially: Go libs, Gin, golang-migrate, MSAL, Graph API, Vue 3, Vite

**While coding:**
- Go: `fmt.Errorf("context: %w", err)` for error wrapping, standard project layout
- Go: return errors, do not panic except at startup
- Vue: Composition API + TypeScript only — never Options API
- Never hardcode secrets, IDs, or URLs — always read from `.env`
- One use case per file in `app/`
- Keep DB queries in `infra/` — never in `app/` or `handler/`

**Do NOT:**
- Add new dependencies without checking if stdlib or existing deps already cover it
- Change architecture without asking first
- Combine multiple use cases into one file
- Store tokens or API keys in plaintext — always encrypt before saving to DB
- Use `var` for package-level mutable state — inject via constructor
- Write code using a lib/framework without searching its current docs first
- Assume function signatures, config format, or CLI flags from memory

## WBS

`wbs/tasks.md` — full task breakdown by phase P01–P10.

## Common commands

```bash
# Dev
docker compose up -d
docker compose logs -f backend
docker compose logs -f frontend

# Migration
docker compose exec backend ./migrate up
docker compose exec backend ./migrate create -ext sql -dir migrations -seq $NAME

# Test
docker compose exec backend go test ./...
docker compose exec frontend npx vitest run

# Deploy
docker compose -f docker-compose.prod.yml up -d --build
```

## Required `.env` variables

```
# Microsoft Entra ID (Confidential Client — Authorization Code + PKCE)
AZURE_CLIENT_ID=
AZURE_CLIENT_SECRET=       # required for Confidential Client flow
AZURE_TENANT_ID=
AZURE_REDIRECT_URI=https://domain.com/auth/callback

# Database
DB_HOST=postgres
DB_PORT=5432
DB_NAME=mailtracker
DB_USER=mailtracker
DB_PASSWORD=

# Security
AES_KEY=                   # 32-byte hex — generate: openssl rand -hex 32
SESSION_SECRET=            # random string — generate: openssl rand -base64 32

# App
APP_ENV=development        # development | production
BASE_URL=http://localhost
SYNC_INTERVAL_MINUTES=30
```