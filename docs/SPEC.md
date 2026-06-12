# Mail Tracker — Product Spec

## Goal
Internal tool for a team of fewer than 20 people to track Outlook email relevant to them.
Each user logs in, pulls mail, and configures their own filters and AI key.

---

## Stack
| Layer | Tech |
|---|---|
| Backend | Go + Gin |
| Database | PostgreSQL |
| Frontend | Vue 3 + Vite |
| Auth | Microsoft Entra ID (OAuth 2.0 + OIDC) |
| Host | Azure VM + Nginx |

---

## High-level architecture

```
Browser (Vue 3)
    │  session cookie (HttpOnly)
    ▼
Gin Backend (Go)
    │  OAuth token (server-side)
    ├──▶ Microsoft Graph API  (pull Outlook mail)
    ├──▶ OpenAI / OpenRouter  (AI summary)
    └──▶ PostgreSQL           (store threads, users, sessions)
```

**Auth principles:**
- The browser stores only the `session_id` cookie — it never sees the OAuth token
- The backend stores the access token + refresh token in the DB (AES-256 encrypted)
- Tokens refresh automatically on expiry, so the user does not need to log in again

---

## Thread Branching — the off-branch reply problem

Outlook allows users to reply to any message in a thread, not necessarily the latest one, which creates branches inside the same `conversationId`.

```
Mail 1 (Mon) — A sends
├── Mail 2 (Tue) — B replies to Mail 1
│   └── Mail 3 (Wed) — A replies to Mail 2  ← main thread
└── Mail 4 (Thu) — C replies to Mail 1      ← side branch
```

The Graph API still returns everything under one `conversationId`, but chronological order does not reflect the real conversation flow when branching exists.

**Handling strategy — 2 phases:**

```
Phase 1 (MVP): Sort by receivedDateTime asc
  → Simple, good enough for 90% of cases
  → Summary may be slightly off-context when the thread branches
  → Implement immediately

Phase 2 (if needed): Build a thread tree from the In-Reply-To header
  → Add internetMessageHeaders to the Graph API $select
  → Parse In-Reply-To → build a parent-child tree
  → Flatten in depth-first order → more accurate summary
  → Implement only when users complain
```

> Do not over-engineer from the start — implement Phase 2 only when there is real feedback.

---

## Background Jobs

Two jobs run independently every 30 minutes:

```
Job 1 — Sync Mail (all active users)
  Use each user's OAuth token
  → Pull new mail from the Graph API
  → Filter by user_filter_settings
  → Store raw threads in the DB
  Do not call AI

Job 2 — AI Summary (only users with an AI key)
  → Fetch threads that have not been summarized
  → Call the AI API with that user's key
  → Update summary, priority, action in the DB
```

"Active user" = has a valid session within the last 7 days.
The user does not need to be online — the server uses the refresh token to pull mail on their behalf.

---

## Feature tiers

| | Tier 1 — No AI | Tier 2 — Copilot via PA | Tier 3 — BYOK |
|---|---|---|---|
| View thread list | ✅ | ✅ | ✅ |
| Read full thread | ✅ | ✅ | ✅ |
| Filter by date, sender | ✅ | ✅ | ✅ |
| AI Summary | ❌ | ✅ Copilot (free) | ✅ User-selected model |
| Priority (High/Med/Low) | ❌ | ✅ | ✅ |
| Action required | ❌ | ✅ | ✅ |
| Update status | ❌ | ✅ | ✅ |
| Notes | ❌ | ✅ | ✅ |
| Cost | Free | Free (M365 license) | User pays by usage |
| Output reliability | — | ⚠️ POC | ✅ |

**Fallback chain in Job 2:**
```
Thread needs summary
  → User has BYOK (AI key)?  → Call OpenAI / OpenRouter   (Tier 3)
  → User enables PA fallback? → Call Power Automate       (Tier 2, POC)
  → Nothing configured        → summary_status = "no_key" (Tier 1)
```

> Tier 2 is a POC — implement it after confirming that PA Copilot output can be parsed as JSON.

---

## Database — table relationships

```
users ──────────────────────────────────────────┐
  │                                              │
  ├── (1:1) user_tokens      OAuth tokens        │
  ├── (1:N) sessions         Browser sessions    │
  ├── (1:N) user_api_keys    AI provider keys    │
  └── (1:N) email_threads    Mail threads        │
                                                 │
users.filter_settings (JSONB) ◀──────────────────┘
  Each user configures their own mail pull conditions
```

**Main tables:**

`users` — profile + filter config (JSONB) + `needs_reauth` flag

`sessions` — `session_id` (random hex) → `user_id`, expires in 7 days

`user_tokens` — access token + refresh token (encrypted), readable only by the backend

`user_api_keys` — AI provider key (encrypted), one active key per user
  - `ai_mode`: `byok` | `power_automate`
  - BYOK: store provider + api_key + model
  - PA: store `pa_webhook_url` (encrypted)

`email_threads` — 1 row = 1 conversation thread
  - Tier 1 fields: subject, from, dates, message_count, raw_messages (JSONB)
  - Tier 2 fields: summary, priority, action_*, status, notes
  - `summary_status`: `pending` → `processing` → `done` / `failed` / `no_key`

---

## User Filter Settings (JSONB)

Users configure this in Settings, stored in `users.filter_settings`:

```json
{
  "my_email": "you@company.com",
  "my_name": "Nguyen Van A",
  "my_name_aliases": ["Van A", "Yasu"],
  "pull_conditions": {
    "to_me": true,
    "cc_me": true,
    "mention_email": true,
    "mention_name": true,
    "mention_aliases": true
  },
  "exclude": {
    "senders": ["noreply@", "newsletter@"],
    "subject_keywords": ["OOO", "Auto-Reply", "unsubscribe"]
  }
}
```

---

## API Endpoints

```
Auth
  GET  /auth/login          Redirect to Microsoft login
  GET  /auth/callback       Exchange code → set session cookie
  POST /auth/logout         Clear session

Threads
  GET  /threads             List threads (filter: status, priority, date, search)
  PATCH /threads/:id        Update status + notes (Tier 2)
  POST /threads/:id/resummary  Trigger AI summary again

Users
  GET  /users/me            Profile + settings
  PATCH /users/me           Update display_name
  GET  /users/me/filter     Get filter settings
  PATCH /users/me/filter    Update filter settings

API Keys
  POST   /users/me/api-key  Save AI config — BYOK or PA webhook
    BYOK body: { "ai_mode": "byok", "provider": "openai", "api_key": "sk-...", "model": "gpt-4o-mini" }
    PA body  : { "ai_mode": "power_automate", "pa_webhook_url": "https://prod-xx..." }
  DELETE /users/me/api-key  Delete AI config

Mail
  POST /mail/sync           Trigger sync immediately (background)
  GET  /mail/sync/:job_id   Check sync job status

Realtime
  GET  /events              SSE stream — push sync_complete when the job finishes
```

---

## Frontend screens

**Login** — one "Login with Microsoft" button, auto-redirect if a session already exists

**Dashboard** — thread table, filter bar, sync button
- Tier 1: Date / From / Subject / Message count columns
- Tier 2: add Summary / Priority / Action / Status (inline dropdown)
- Warning banner if there is no AI key or reauth is required
- Auto re-render when the `sync_complete` SSE event arrives (no reload needed)

**Thread Detail** (slide-in drawer) — read the full thread, re-summarize, add notes, deep link to Outlook

**Settings**
- Filter config: Form ↔ JSON tab (real-time sync)
- AI config: choose one of two options (see screen below)
- Logout

```
AI Summary Configuration

  Choose a method:
  ◉ BYOK — Use my API key
  ○ Copilot via Power Automate (free, uses M365 license)

  -- When BYOK is selected ----------------------
  Provider : [OpenAI ▼] [OpenRouter ▼]
  API Key  : [••••••••••••••••]
  Model    : [gpt-4o-mini]
             Suggested: gpt-4o-mini / deepseek/deepseek-chat
  [Test] [Save]

  -- When Copilot via PA is selected ------------
  Power Automate Webhook URL:
  [https://prod-xx.westus.logic.azure.com/...]
  Tip: create a flow in Power Automate, then paste the URL here
  [Test] [Save]

  Warning: Copilot via PA is an experimental feature (POC)
           Output may be less stable than BYOK
```

---

## Clean Architecture (Go)

```
internal/
  domain/       Pure Go structs, interfaces, errors — zero dependencies
  app/          Use cases — orchestrate domain + call interfaces
  infra/        Implement interfaces: DB, Graph API, AI API, OAuth
  handler/      Gin HTTP handlers + middleware
```

Dependency rule: `domain ← app ← infra / handler`
Handlers do not call the DB directly — they only call use cases.

---

## Suggested implementation order

```
1. Auth flow (OAuth login → session → token storage)
2. DB schema + migration
3. Job 1: pull mail + filter
4. Job 2: AI summary
5. REST API endpoints
6. Vue frontend
7. Deploy Nginx + systemd
```

## Realtime Update — Server-Sent Events (SSE)

The frontend does not poll and does not require manual refresh — the backend pushes when new data is available.

```
Sync job completes for user X
  → Broker publishes an event into user X's channel
  → SSE handler pushes the event to the browser
  → Vue receives it → calls GET /threads → re-renders automatically
```

**Broker pattern:**
```
SyncJob ──publish──▶ Broker ──▶ SSE Handler ──▶ Browser
                    (per-user Go channel)
```

- Each user has a dedicated channel in the broker (`map[userID]chan Event`)
- User opens the app → connects to `GET /events` → keep-alive connection
- User closes the tab → context cancels → broker cleans up the channel automatically
- No WebSocket or Redis required — plain Go channels are enough
- Heartbeat every 30 seconds to keep the connection alive through proxies

**Frontend:**
```javascript
const es = new EventSource('/events', { withCredentials: true })
es.addEventListener('sync_complete', () => fetchThreads())
es.addEventListener('heartbeat', () => {}) // keep the connection alive
```

**Additional endpoint in the API list:**
```
GET /events    SSE stream — push 'sync_complete' when a job finishes
```

---

## Infrastructure — Docker

4 separate containers, orchestrated with Docker Compose.

```
docker-compose.yml
  ├── postgres      PostgreSQL 16
  ├── backend       Go + Gin
  ├── frontend      Vue 3 + Vite
  └── nginx         Reverse proxy
```

**Routing via Nginx:**
```
https://domain.com/        → frontend container :5173
https://domain.com/api/*   → backend container  :8080
https://domain.com/auth/*  → backend container  :8080
https://domain.com/events  → backend container  :8080 (SSE)
```

---

### Hot reload in local development

**Backend (Go):** use `air` — watch for file changes → recompile automatically → restart
```
local: edit Go code → air detects it → rebuilds inside the container → auto reload
```

**Frontend (Vue):** Vite dev server has built-in HMR
```
local: edit a `.vue` file → Vite HMR → browser updates without a full reload
```

**How it works — bind mount:**

Bind mount maps a physical host directory directly into the container:
```
Local machine: /home/user/mail-tracker/backend/   ← you edit files here
                       │ bind mount
Container    : /app/                              ← container reads from here
```
No copying, no syncing — it is the same physical directory seen from both sides.
Edit `.go` files in VS Code on your machine → `air` inside the container sees them immediately → recompiles.
Edit `.vue` files in VS Code on your machine → Vite HMR inside the container sees them immediately → browser updates.

In `docker-compose.yml`, it looks like this:
```yaml
backend:
  volumes:
    - ./backend:/app          # bind mount — source code

frontend:
  volumes:
    - ./frontend:/app         # bind mount — source code

postgres:
  volumes:
    - postgres_data:/var/lib/postgresql/data   # named volume — NOT a bind mount

volumes:
  postgres_data:              # managed by Docker
```

---

### Bind mount vs Named volume

| | Bind mount | Named volume |
|---|---|---|
| Used for | Source code | PostgreSQL data |
| Host directory | Chosen by you (`./backend`) | Managed by Docker (`/var/lib/docker/volumes/`) |
| Container restart | Preserved | Preserved |
| Container removal | Preserved | Preserved |
| `docker compose down` | Preserved | Preserved |
| `docker compose down -v` | Preserved | ⚠️ Data lost |

PostgreSQL should not use a bind mount because it can run into permission conflicts between the host user and the postgres user inside the container.

**Rule:** use `docker compose down -v` only when you want to fully reset the DB.

---

### Two Docker Compose environments

**Local dev** (`docker-compose.yml`):
```
backend:
  - Mount source code from the host into the container
  - Run air (hot reload)
  - Expose port 8080 to the host for debugging

frontend:
  - Mount source code from the host into the container
  - Run the Vite dev server (HMR)
  - Expose port 5173 to the host

postgres:
  - Expose port 5432 to the host for DB clients (DBeaver, TablePlus...)

nginx:
  - Proxy /api/* → backend
  - Proxy /* → frontend dev server
```

**Production** (`docker-compose.prod.yml`):
```
backend:
  - Build a Go binary (multi-stage build)
  - Do not mount source code
  - Do not expose ports publicly (only Nginx can access it)

frontend:
  - Build static files (`vite build`)
  - Nginx serves static files directly

postgres:
  - Do not expose ports publicly

nginx:
  - SSL termination (cert mounted in)
  - Serve frontend static files
  - Proxy /api/* → backend
```

---

### Daily workflow

```bash
# First-time setup
cp .env.example .env        # fill in config
docker compose up -d        # start all containers

# Regular development
# → Edit Go or Vue code on the host machine → hot reload happens automatically in containers

# View logs
docker compose logs -f backend
docker compose logs -f frontend

# Run migrations
docker compose exec backend ./migrate up

# Deploy production
docker compose -f docker-compose.prod.yml up -d --build
```

---

### Docker file structure

```
src/
├── docker-compose.yml          # local dev
├── docker-compose.prod.yml     # production
├── .env                        # do not commit
├── .env.example
├── backend/
│   ├── Dockerfile              # multi-stage: build + run
│   ├── .air.toml               # air config for hot reload
│   └── ...
├── frontend/
│   ├── Dockerfile              # multi-stage: dev + build + nginx
│   └── ...
└── nginx/
    ├── nginx.dev.conf          # local dev config
    └── nginx.prod.conf         # production config (SSL)
```
