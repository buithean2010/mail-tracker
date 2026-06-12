---
description: Start the local dev environment with Docker Compose (hot reload for Go + Vue)
when_to_use: When the user wants to run, start, or restart the local dev environment, or asks about ports, hot reload, or Docker setup
allowed-tools: Bash(docker compose up *), Bash(docker compose down *), Bash(docker compose logs *), Bash(docker compose ps *)
argument-hint: "[--build] [service-name]"
---

# Run Dev Environment

```bash
# First time: copy env
cp .env.example .env   # fill in CLIENT_ID, CLIENT_SECRET, AES_KEY, DB_PASSWORD

# Start all containers
docker compose up -d

# Watch logs in real time
docker compose logs -f backend
docker compose logs -f frontend

# Stop
docker compose down
```

**Ports:**
- Frontend (Vite HMR): http://localhost:5173
- Backend (Go/air): http://localhost:8080
- Postgres: localhost:5432
- App via Nginx: http://localhost

**Hot reload:**
- Go: `air` detects `.go` changes → rebuild → restart automatically
- Vue: Vite HMR injects updates into the browser automatically

**Troubleshoot:**
```bash
# Rebuild after changing Dockerfile or go.mod
docker compose up -d --build backend

# Reset DB clean
docker compose down -v && docker compose up -d
```
