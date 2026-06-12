# Mail Tracker — Work Breakdown Structure

> Stack: Go + Gin | Vue 3 + Vite | PostgreSQL | Microsoft Entra ID | Docker + Nginx
> All implementation follows Clean Architecture: `domain ← app ← infra / handler`

## Status legend
`todo` | `in-progress` | `done` | `blocked` | `skip`

---

## P01 — Project Setup & Infrastructure

| task_id | status | summary |
|---|---|---|
| P01_01 | done | Initialize the Go module (`go.mod`), `internal/{domain,app,infra,handler}` directories, basic Makefile |
| P01_02 | done | Initialize the Vue 3 + Vite frontend (`pnpm create vite`), configure TypeScript + path aliases |
| P01_03 | done | `docker-compose.yml` for local dev: postgres + backend (air) + frontend (HMR) + nginx, bind mount source code |
| P01_04 | done | `docker-compose.prod.yml` for production: multi-stage Go + Vue build, no ports exposed except through Nginx |
| P01_05 | done | Nginx config: `nginx.dev.conf` (proxy → containers), `nginx.prod.conf` (SSL termination + serve static files) |
| P01_06 | done | `.env.example` with all required variables, config loader in Go (`viper` or env package) |
| P01_07 | done | Multi-stage `backend/Dockerfile` (build binary + minimal runtime), multi-stage `frontend/Dockerfile` (dev + build + nginx) |
| P01_08 | done | `.air.toml` for Go hot reload inside the container |

---

## P02 — Database Schema & Migration

| task_id | status | summary |
|---|---|---|
| P02_01 | done | Choose a migration tool (`golang-migrate`), set up the `migrate` binary in the backend container |
| P02_02 | done | Migration: `users` table — profile, `filter_settings` JSONB, `needs_reauth` flag |
| P02_03 | done | Migration: `sessions` table — random hex `session_id` → `user_id`, expires in 7 days |
| P02_04 | done | Migration: `user_tokens` table — encrypted `access_token` + `refresh_token`, `expires_at` |
| P02_05 | done | Migration: `user_api_keys` table — `ai_mode` (`byok`/`power_automate`), encrypted `api_key`, `model`, encrypted `pa_webhook_url` |
| P02_06 | done | Migration: `email_threads` table — Tier 1 fields + Tier 2 fields + `summary_status` enum (`pending`/`processing`/`done`/`failed`/`no_key`) |
| P02_07 | done | Indexes: `email_threads(user_id, received_at DESC)`, `sessions(session_id)`, `sessions(expires_at)` |

---

## P03 — Authentication

| task_id | status | summary |
|---|---|---|
| P03_01 | done | Configure the Microsoft Entra ID app registration (redirect URI, scopes: `openid profile email Mail.Read`) |
| P03_02 | done | `GET /auth/login` — build Microsoft authorization URL, redirect, store the state param in a cookie |
| P03_03 | done | `GET /auth/callback` — validate state, exchange code → access token + refresh token, upsert user |
| P03_04 | done | AES-256 encryption service (`internal/infra/crypto`) for token storage |
| P03_05 | done | Save tokens in `user_tokens` (encrypted), create a session record, set the `session_id` HttpOnly cookie |
| P03_06 | done | Session middleware — validate cookie → load user from DB, inject into context |
| P03_07 | done | Automatic token refresh: when the access token is close to expiry (<5 minutes), call the refresh endpoint and update the DB |
| P03_08 | done | `POST /auth/logout` — delete session record, clear cookie |
| P03_09 | done | `needs_reauth` flow: set flag when refresh token fails, return 401 with `{"error":"reauth_required"}`, frontend redirects to login |

---

## P04 — Domain & App Layer (Clean Architecture)

| task_id | status | summary |
|---|---|---|
| P04_01 | done | `domain/` — Go structs: `User`, `Session`, `UserToken`, `UserAPIKey`, `EmailThread`, `FilterSettings` |
| P04_02 | done | `domain/` — Interface definitions: `UserRepo`, `SessionRepo`, `TokenRepo`, `ThreadRepo`, `APIKeyRepo` |
| P04_03 | done | `domain/` — Interface definitions: `MailClient`, `AIClient`, `PAClient`, `Encryptor` |
| P04_04 | done | `app/` — Use case: `AuthUseCase` (login flow, logout, token refresh) |
| P04_05 | done | `app/` — Use case: `SyncUseCase` (pull mail, apply filters, save threads) |
| P04_06 | done | `app/` — Use case: `SummaryUseCase` (fallback chain: BYOK → PA → no_key) |
| P04_07 | done | `app/` — Use case: `ThreadUseCase` (list with filters, update status/notes, trigger re-summary) |
| P04_08 | done | `app/` — Use case: `UserUseCase` (get profile, update `display_name`, filter settings CRUD, API key CRUD) |

---

## P05 — Infrastructure Layer

| task_id | status | summary |
|---|---|---|
| P05_01 | done | PostgreSQL repositories: implement `UserRepo`, `SessionRepo`, `TokenRepo` (`pgx` or `sqlx`) |
| P05_02 | done | PostgreSQL repositories: implement `ThreadRepo`, `APIKeyRepo` — including JSONB filters and pagination |
| P05_03 | done | Microsoft Graph API client: `GET /me/mailFolders/Inbox/messages`, `$select` fields, `$filter`, `$orderby`, paging with `@odata.nextLink` |
| P05_04 | done | Graph API: parse `conversationId`, group messages into threads, sort `receivedDateTime asc` (Phase 1 strategy) |
| P05_05 | done | OpenAI client: call the Chat Completions API, parse JSON output (summary, priority, action_required) |
| P05_06 | done | OpenRouter client: same interface as OpenAI, different base URL + auth header |
| P05_07 | done | Power Automate client (POC): POST to webhook URL, parse Copilot JSON response |
| P05_08 | done | Retry + timeout logic for all external HTTP calls (3 retries with backoff, 30s timeout) |

---

## P06 — Background Jobs & SSE Broker

| task_id | status | summary |
|---|---|---|
| P06_01 | done | Job scheduler: 30-minute ticker, run Job 1 + Job 2 independently, structured logging |
| P06_02 | done | Job 1 — Mail sync: fetch active users (session within 7 days), use each user's OAuth token, pull Graph API, apply `filter_settings`, upsert `email_threads` |
| P06_03 | done | Job 1 — Filter logic: `to_me`, `cc_me`, `mention_email`, `mention_name`, `mention_aliases`, `exclude.senders`, `exclude.subject_keywords` |
| P06_04 | done | Job 2 — AI summary: fetch threads with `summary_status=pending`, run fallback chain (BYOK → PA → no_key), update status + fields |
| P06_05 | done | Job 2 — Concurrency: process multiple users in parallel (goroutines + semaphore), rate limit per AI provider |
| P06_06 | done | SSE Broker: `map[userID]chan Event`, publish/subscribe pattern, per-user channel cleanup on disconnect |
| P06_07 | done | SSE Broker: 30-second heartbeat goroutine to keep proxy connections alive, automatically clean up stale channels |
| P06_08 | done | Job 1 publishes a `sync_complete` event into the broker after sync finishes for each user |

---

## P07 — REST API Handlers

| task_id | status | summary |
|---|---|---|
| P07_01 | done | Auth handlers: `GET /auth/login`, `GET /auth/callback`, `POST /auth/logout` |
| P07_02 | done | Thread handlers: `GET /threads` (filters: status, priority, date, search), `PATCH /threads/:id`, `POST /threads/:id/resummary` |
| P07_03 | done | User handlers: `GET /users/me`, `PATCH /users/me`, `GET /users/me/filter`, `PATCH /users/me/filter` |
| P07_04 | done | API key handlers: `POST /users/me/api-key` (BYOK + PA body validation), `DELETE /users/me/api-key` |
| P07_05 | done | Mail handlers: `POST /mail/sync` (trigger a background sync job for the current user), `GET /mail/sync/:job_id` |
| P07_06 | done | SSE handler: `GET /events` — upgrade connection, register broker channel, stream events, clean up on disconnect |
| P07_07 | done | Middleware: CORS (allow only frontend origin), request logging, error recovery, rate limiting |
| P07_08 | done | AI key test endpoint: `POST /users/me/api-key/test` — run a sample AI call and return the result without saving |

---

## P08 — Frontend (Vue 3)

| task_id | status | summary |
|---|---|---|
| P08_01 | todo | App shell: Vue Router (login, dashboard, settings), Pinia store setup, axios instance with credentials |
| P08_02 | todo | Login page: one "Login with Microsoft" button, auto-redirect if session is valid, handle `reauth_required` error |
| P08_03 | todo | Dashboard: thread list component, tier-based columns (Tier 1 vs Tier 2), loading + empty states |
| P08_04 | todo | Dashboard: filter bar (date range, status, priority, sender, search text), debounced input |
| P08_05 | todo | Dashboard: sync button → `POST /mail/sync` → polling or wait for SSE `sync_complete` → refresh list |
| P08_06 | todo | Dashboard: warning banner for `no_ai_key` or `needs_reauth` (from `GET /users/me`) |
| P08_07 | todo | Thread detail: slide-in drawer, render full thread messages, re-summarize button, notes input, deep link to Outlook |
| P08_08 | todo | Settings — Filter config: form editor ↔ raw JSON tab (two-way, real-time sync), validate JSON |
| P08_09 | todo | Settings — AI config: BYOK vs PA radio, conditional form fields, test button, save |
| P08_10 | todo | SSE integration: `EventSource('/events', {withCredentials: true})`, `sync_complete` → `fetchThreads()`, heartbeat listener |
| P08_11 | todo | Tier-aware UI: hide/show columns + actions based on `summary_status` and the user's AI config |
| P08_12 | todo | Inline status dropdown on the dashboard (Tier 2): call `PATCH /threads/:id` immediately on change |

---

## P09 — Testing

| task_id | status | summary |
|---|---|---|
| P09_01 | todo | Unit tests for `domain/` structs + validation logic |
| P09_02 | todo | Unit tests for `app/` use cases with mocked interfaces |
| P09_03 | todo | Integration tests for `infra/` repositories (use a test DB, no mocks) |
| P09_04 | todo | Integration tests for the Graph API client (mock HTTP server) |
| P09_05 | todo | API handler tests: `httptest`, verify response codes + body |
| P09_06 | todo | Frontend: Vitest unit tests for Pinia stores + utility functions |

---

## P10 — Deployment

| task_id | status | summary |
|---|---|---|
| P10_01 | todo | Azure VM setup: install Docker + Docker Compose, firewall rules (80, 443), non-root user |
| P10_02 | todo | SSL certificate: Let's Encrypt with Certbot, auto-renew cron, mount cert into the Nginx container |
| P10_03 | todo | Production `.env` with secrets (DB password, AES key, OAuth client secret, session secret) |
| P10_04 | todo | Deploy workflow: `docker compose -f docker-compose.prod.yml up -d --build`, zero-downtime strategy |
| P10_05 | todo | Basic monitoring: Docker health checks for all containers, access via `docker compose logs` |

---

## Suggested implementation order

```
P01 (setup) → P02 (DB) → P03 (auth) → P04+P05 (domain+infra) → P06 (jobs) → P07 (API) → P08 (frontend) → P09 (test) → P10 (deploy)
```

> **Phase 2 — Thread branching** (implement when users provide feedback): add `internetMessageHeaders` to Graph API `$select`, parse `In-Reply-To`, build a parent-child tree, flatten depth-first before summarizing.
