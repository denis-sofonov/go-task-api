# Reference Backend — Go (1.26+)

> **Status: work in progress.** This README describes the target state; most of
> it is not built yet. Today the repository contains the project skeleton: a
> `net/http` server with `GET /health`, plus `make run`, `build`, `test` and
> `lint`.

A task-management API (users, projects, tasks) built with Go, Huma, PostgreSQL
and sqlc. It is a sibling of the FastAPI implementation and the full-stack
[Nuxt](https://github.com/denis-sofonov/ts-nuxt-task-app) and
[Next.js](https://github.com/denis-sofonov/ts-next-task-app) takes: the same
domain, different stacks.

> A user owns projects; a project contains tasks. You can only see and change
> your own data.

## Tech stack

| Concern            | Choice                                                  |
| ------------------ | ------------------------------------------------------- |
| Language / runtime | Go 1.26+, single static binary                          |
| HTTP               | `net/http` (Go 1.22+ routing) + [Huma v2](https://huma.rocks) |
| Database           | PostgreSQL 17, [pgx v5](https://github.com/jackc/pgx) pool |
| Queries            | [sqlc](https://sqlc.dev) — type-safe Go generated from SQL |
| Migrations         | [goose](https://github.com/pressly/goose) (plain SQL, embedded) |
| Auth               | Bearer JWT access + revocable, rotating refresh tokens  |
| Passwords          | Argon2id (`golang.org/x/crypto/argon2`)                 |
| Background jobs    | [River](https://riverqueue.com) (Postgres-backed queue) |
| Cache / rate limit | Redis ([go-redis v9](https://github.com/redis/go-redis)) |
| Logging            | `log/slog` (JSON)                                       |
| Config             | Environment variables ([caarlos0/env](https://github.com/caarlos0/env)) |
| Tooling            | golangci-lint, `go test` + testcontainers-go, Make      |

## Features

- Registration, login, `me`, token refresh (with rotation) and logout.
- Email verification and password reset via short-lived signed tokens.
- CRUD for projects and tasks nested under a project.
- Ownership authorization on every resource (`403` for others', `404` if absent).
- Pagination, case-insensitive search, status filtering and whitelisted sorting.
- Per-endpoint rate limiting on authentication.
- One error envelope for every failure.
- Background email dispatch, Redis caching, JSON logs and a DB-aware health check.
- OpenAPI 3.1 generated from Go types, interactive docs and an exported `openapi.json`.
- Graceful shutdown: in-flight requests and jobs finish before the process exits.

## Design decisions & trade-offs

- **Huma on `net/http`, not Fiber or Gin.** Huma is the closest Go analogue of
  FastAPI: handlers take and return typed structs, and validation plus the
  OpenAPI schema are derived from those types. It sits on the standard
  `net/http` router, so all ecosystem middleware works. Fiber is fast but built
  on `fasthttp`, which is not compatible with `net/http`. For a JSON API backed
  by Postgres the database dominates latency, so that speed buys nothing here.
- **sqlc instead of an ORM.** Queries are written as plain SQL and sqlc
  generates typed Go functions from them at build time. You get compile-time
  checking with no reflection or query builder at runtime, and the SQL you read
  is the SQL that runs. The cost is that dynamic queries such as optional
  filters and sorting take some care. They are handled with a fixed whitelist
  and `CASE`/`COALESCE` patterns rather than string building.
- **Package by feature, not by layer.** `internal/auth`, `internal/project` and
  `internal/task` each hold their own handler, service and storage code. This is
  the idiomatic Go layout: a package is a unit of meaning, and `internal/` keeps
  everything private to this module.
- **Explicit dependency injection in `main`.** There is no DI container. `main`
  builds the pool, services and handlers and passes them in as arguments. Go
  favours wiring you can read top to bottom over magic.
- **Interfaces at the consumer.** A service declares the small interface it
  needs (for example `Mailer`) instead of depending on a concrete type. Tests
  then substitute fakes without a mocking framework.
- **JWT access + revocable refresh tokens (hybrid).** Access tokens are
  stateless and short-lived (15 min), so authorizing a request never touches the
  database. Refresh tokens are stored hashed (SHA-256) and are revocable. Each
  refresh rotates them, so reuse is detectable, and logout and password reset
  revoke them. This matches the sibling implementations.
- **River over a Redis queue.** River keeps jobs in Postgres. A job can be
  enqueued in the same transaction as the data change, so "user created but
  welcome email lost" cannot happen. Redis is still used for the cache and rate
  limiting, where losing data is harmless.
- **Errors are values.** Every layer returns `error`. Domain errors
  (`ErrNotFound`, `ErrForbidden` and so on) are mapped to the HTTP error envelope
  in one place, and unexpected errors become `500` without leaking internals.
- **Real Postgres in tests.** Integration tests start PostgreSQL with
  testcontainers-go instead of mocking the database, because sqlc queries are
  only proven by running them.

## Requirements

- Go 1.26+
- Docker (for PostgreSQL and Redis)
- `make` (optional; every target is a plain command)

## Getting started

> Only `make run`, `build`, `test` and `lint` exist today. The other targets
> below (`tools`, `migrate`, `seed`, `worker`, `openapi`, `check`) are planned.

### Local development

```bash
cp .env.example .env
docker compose up -d db redis     # PostgreSQL on :5434, Redis on :6379
make tools                        # install sqlc, goose, golangci-lint
make migrate                      # apply migrations
make seed                         # optional: load demo data
make run                          # go run ./cmd/api
```

The API is then at `http://localhost:8080` and the docs at `http://localhost:8080/docs`.
Run the background worker in a second terminal:

```bash
make worker                       # go run ./cmd/worker
```

Demo user (after seeding): `demo@example.com` / `password123`.

### Docker (full stack, one command)

```bash
JWT_SECRET_KEY=$(openssl rand -base64 48) \
  docker compose -f docker-compose.prod.yml up --build
```

This starts PostgreSQL, Redis, the migration job, the API, the worker and
nginx. The API is served at `http://localhost:8081`.

## API reference

All endpoints are versioned under `/api/v1` except `/health`. Protected routes
require `Authorization: Bearer <access_token>`.

### Authentication

| Method | Endpoint                               | Description                     |
| ------ | -------------------------------------- | ------------------------------- |
| POST   | `/api/v1/auth/register`                | Create an account               |
| POST   | `/api/v1/auth/login`                   | Exchange credentials for tokens |
| GET    | `/api/v1/auth/me`                      | Current user                    |
| POST   | `/api/v1/auth/refresh`                 | Rotate the token pair           |
| POST   | `/api/v1/auth/logout`                  | Revoke refresh token(s)         |
| POST   | `/api/v1/auth/verify-email/request`    | Send a verification email       |
| POST   | `/api/v1/auth/verify-email/confirm`    | Confirm an email address        |
| POST   | `/api/v1/auth/password-reset/request`  | Send a reset email              |
| POST   | `/api/v1/auth/password-reset/confirm`  | Set a new password              |

### Projects

| Method | Endpoint                | Description                |
| ------ | ----------------------- | -------------------------- |
| GET    | `/api/v1/projects`      | List your projects (paged) |
| POST   | `/api/v1/projects`      | Create a project           |
| GET    | `/api/v1/projects/{id}` | Get a project              |
| PATCH  | `/api/v1/projects/{id}` | Update a project           |
| DELETE | `/api/v1/projects/{id}` | Delete a project           |

Query parameters: `page`, `size`, `q` (search by name), `sort`
(`created_at`, `updated_at`, `name`; prefix `-` for descending).

### Tasks

| Method | Endpoint                                   | Description        |
| ------ | ------------------------------------------ | ------------------ |
| GET    | `/api/v1/projects/{project_id}/tasks`      | List tasks (paged) |
| POST   | `/api/v1/projects/{project_id}/tasks`      | Create a task      |
| GET    | `/api/v1/projects/{project_id}/tasks/{id}` | Get a task         |
| PATCH  | `/api/v1/projects/{project_id}/tasks/{id}` | Update a task      |
| DELETE | `/api/v1/projects/{project_id}/tasks/{id}` | Delete a task      |

Query parameters: `page`, `size`, `q` (search by title), `status`
(`todo`, `in_progress`, `done`), `sort`
(`created_at`, `updated_at`, `title`, `status`).

### Stats & system

| Method | Endpoint        | Description                       |
| ------ | --------------- | --------------------------------- |
| GET    | `/api/v1/stats` | Counts of your projects and tasks |
| GET    | `/health`       | Liveness + database readiness     |

### Error envelope

Every failure has the same shape:

```json
{ "error": { "code": "not_found", "message": "Project not found", "details": [] } }
```

Codes: `bad_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`,
`validation_error`, `rate_limit_exceeded`, `internal_error`.

### Example

```bash
# Register and log in
curl -s -X POST localhost:8080/api/v1/auth/register \
  -H 'content-type: application/json' \
  -d '{"email":"ada@example.com","name":"Ada","password":"s3cret-pass"}'

TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"ada@example.com","password":"s3cret-pass"}' | jq -r .access_token)

# Create a project, then a task in it
PROJECT_ID=$(curl -s -X POST localhost:8080/api/v1/projects \
  -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"name":"Website redesign"}' | jq -r .id)

curl -s -X POST localhost:8080/api/v1/projects/$PROJECT_ID/tasks \
  -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"title":"Draft the new layout"}'
```

## Operations

- **Background jobs**: emails are dispatched through River and processed by
  `cmd/worker`.
- **Scheduled cleanup**: a periodic River job purges expired and revoked
  refresh tokens every night.
- **Cache**: `/api/v1/stats` uses Redis cache-aside with a short TTL.
- **Logging**: `slog` writes one JSON access line per request with method,
  path, status, duration and a correlated `X-Request-ID`.
- **Health**: `/health` pings the database and returns `503` if it is down.
- **Shutdown**: on `SIGINT`/`SIGTERM` the server stops accepting connections and
  waits up to `SHUTDOWN_TIMEOUT` for in-flight requests to finish.

### Configuration notes

- **CORS** is denied by default. Set `CORS_ORIGINS` (comma-separated) to allow
  browser clients.
- **Rate limiting** keys on the client IP. Behind the bundled nginx, set
  `TRUST_FORWARDED_FOR=true` so the real client IP is used instead of the proxy's.
- **Secrets**: in `production` the app refuses to start with the default or a
  weak (`< 32` char) `JWT_SECRET_KEY`.

## Generated API docs

Huma serves the schema and interactive docs:

- Docs UI: `/docs`
- Raw schema: `/openapi.json`, `/openapi.yaml`

A committed copy will live at `openapi.json` (planned). Regenerate it without
running the server:

```bash
make openapi
```

## Quality checks

```bash
make check      # go vet, golangci-lint, go test -race with coverage
```

CI runs the same gate on every push and pull request. It applies the
migrations, checks that `sqlc generate` leaves no diff and enforces a 70%
minimum coverage.

## Project structure

```
cmd/
  api/            # HTTP server entrypoint: wiring + graceful shutdown
  worker/         # River worker entrypoint
  seed/           # demo data loader
internal/
  config/         # env config + validation
  platform/       # infrastructure: postgres pool, redis, logger
  httpx/          # server, middleware (request id, logging, recover, CORS), error envelope
  auth/           # registration, login, JWT, refresh tokens, password hashing
  project/        # project handlers, service, ownership checks
  task/           # task handlers, service
  stats/          # stats endpoint + cache
  jobs/           # River job definitions (emails, cleanup)
  db/             # sqlc-generated code (do not edit)
  pagination/     # page/size/sort parsing shared by list endpoints
db/
  migrations/     # goose SQL migrations (embedded into the binary)
  queries/        # SQL that sqlc turns into Go
deploy/           # nginx configuration
```

## License

[MIT](LICENSE)
