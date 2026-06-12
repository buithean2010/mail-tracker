.PHONY: init dev down logs-backend logs-frontend migrate-up migrate-down test deploy

# Run once after cloning to generate go.sum and install frontend deps
init:
	cd src/backend && go mod tidy
	cd src/frontend && pnpm install

dev:
	docker compose -f src/docker-compose.yml up -d

build:
	docker compose -f src/docker-compose.yml up -d --build

down:
	docker compose -f src/docker-compose.yml down

logs-backend:
	docker compose -f src/docker-compose.yml logs -f backend

logs-frontend:
	docker compose -f src/docker-compose.yml logs -f frontend

migrate-up:
	docker compose -f src/docker-compose.yml exec backend migrate -path ./migrations -database "$$(docker compose -f src/docker-compose.yml exec -T backend printenv DB_URL 2>/dev/null || echo 'postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=disable')" up

migrate-down:
	docker compose -f src/docker-compose.yml exec backend ./migrate down 1

test:
	docker compose -f src/docker-compose.yml exec backend go test ./...
	docker compose -f src/docker-compose.yml exec frontend pnpm test

deploy:
	docker compose -f src/docker-compose.prod.yml up -d --build
