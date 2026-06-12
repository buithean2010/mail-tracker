---
description: Run backend Go tests and frontend Vitest tests
when_to_use: When the user wants to run tests, check coverage, or asks about unit/integration testing conventions
allowed-tools: Bash(docker compose exec backend go test *), Bash(docker compose exec frontend pnpm test *), Bash(go test *)
argument-hint: "[./path/... | --coverage | --watch]"
---

# Running Tests

## Backend (Go)

```bash
# All tests
docker compose exec backend go test ./...

# Tests with coverage
docker compose exec backend go test ./... -cover

# Unit tests only (no DB required)
docker compose exec backend go test ./internal/domain/... ./internal/app/...

# Integration tests (requires running DB)
docker compose exec backend go test ./internal/infra/... -tags integration

# Verbose output
docker compose exec backend go test ./... -v

# Run directly on host (requires Go installed)
cd backend && go test ./...
```

## Frontend (Vitest)

```bash
docker compose exec frontend pnpm test

# Watch mode
docker compose exec frontend pnpm test --watch

# Coverage
docker compose exec frontend pnpm test --coverage
```

## Test conventions
- Unit tests: mock all interfaces (domain + app layer)
- Integration tests: use a real test DB, do not mock repositories
- HTTP tests: use `httptest.NewRecorder()`, do not spin up a live server
