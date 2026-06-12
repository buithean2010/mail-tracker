---
description: Run database migrations with golang-migrate inside the backend container
when_to_use: When the user wants to apply, roll back, or create DB migrations, or asks about schema changes, tables, or migration versioning
allowed-tools: Bash(docker compose exec backend ./migrate *)
argument-hint: "[up|down|version|create <name>]"
---

# Database Migrations

Migration files: `backend/migrations/` (format: `000001_create_users.up.sql` / `000001_create_users.down.sql`)

```bash
# Apply all pending migrations
docker compose exec backend ./migrate up

# Rollback 1 step
docker compose exec backend ./migrate down 1

# Show current migration version
docker compose exec backend ./migrate version

# Create a new migration file
docker compose exec backend ./migrate create -ext sql -dir /app/migrations -seq $ARGUMENTS
# Example: /db-migrate add_email_threads
```

**Migration order:**
1. `users` (base table)
2. `sessions`
3. `user_tokens`
4. `user_api_keys`
5. `email_threads`
6. Indexes

**Reset DB (dev only):**
```bash
docker compose down -v   # DELETES all data
docker compose up -d
docker compose exec backend ./migrate up
```
