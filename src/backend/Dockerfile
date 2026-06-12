# ── dev stage ─────────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS dev

RUN apk add --no-cache git curl

# air for hot reload
RUN go install github.com/air-verse/air@latest

# golang-migrate binary
RUN curl -sSL https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-amd64.tar.gz \
    | tar xvz migrate && mv migrate /usr/local/bin/

WORKDIR /app

# Allow Go to fetch and update go.sum on first build
ENV GONOSUMDB=*
ENV GOFLAGS=-mod=mod

COPY go.mod go.su[m] ./
RUN go mod download 2>/dev/null || true

EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# ── build stage ───────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS build

RUN apk add --no-cache git

WORKDIR /app
ENV GONOSUMDB=*

COPY go.mod go.su[m] ./
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./...

# ── prod stage ────────────────────────────────────────────────────────────────
FROM alpine:3.21 AS prod

RUN apk add --no-cache ca-certificates curl

# golang-migrate binary for production migrations
RUN curl -sSL https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-amd64.tar.gz \
    | tar xvz migrate && mv migrate /usr/local/bin/

WORKDIR /app
COPY --from=build /app/server .
COPY --from=build /app/migrations ./migrations

EXPOSE 8080
CMD ["./server"]
