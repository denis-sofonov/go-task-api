# Reference Backend — Go (1.26+)

A task-management API for learning Go and its ecosystem. The project covers
HTTP handlers and middleware, PostgreSQL queries and migrations, Redis rate
limiting, Kafka events, and monitoring with Prometheus and Grafana.

It shares a domain with the [Nuxt](https://github.com/denis-sofonov/ts-nuxt-task-app)
and [Next.js](https://github.com/denis-sofonov/ts-next-task-app) implementations.

## Tech stack

| Area | Technologies |
| --- | --- |
| API | Go 1.26, standard library `net/http` and `ServeMux` |
| Database | PostgreSQL 17, pgx, sqlc, Goose |
| Authentication | Bearer JWT, bcrypt, Redis for login rate limiting |
| Events | Apache Kafka, kafka-go |
| Observability | JSON logs with `log/slog`, Prometheus, Grafana |
| Local development | Docker Compose, Make, golangci-lint |

## Getting started

With Docker Compose, start the full stack from the repository root:

```bash
docker compose up -d --build
curl -fsS http://localhost:8080/health
```

Compose starts PostgreSQL, Redis, Kafka, the migration job, the API, the Kafka
worker, Prometheus, and Grafana. The API is available at
[localhost:8080](http://localhost:8080/health). The health endpoint checks
PostgreSQL and returns `503` if the database is unavailable.

### Running Go on the host

Install Go 1.26+ and Make. Copy the example environment file if you do not
already have a local `.env`:

```bash
cp .env.example .env
docker compose up -d --wait db redis kafka
make migrate
make run
```

Run `make worker` in a second terminal to consume Kafka events. Make loads
configuration from `.env`. PostgreSQL is exposed on port `5434`, Redis on
`6379`, and Kafka on `9092`.

## API reference

Protected routes require `Authorization: Bearer <access_token>`. Tasks belong
directly to the signed-in user. Looking up, updating, or deleting a task owned
by another user returns `404`.

| Method | Path | Authentication | Purpose |
| --- | --- | --- | --- |
| POST | `/auth/register` | Public | Create an account with `email`, `name`, and `password` |
| POST | `/auth/login` | Public | Exchange `email` and `password` for an access token |
| GET | `/me` | Bearer token | Get the current user |
| GET | `/tasks` | Bearer token | List your tasks |
| POST | `/tasks` | Bearer token | Create a task with `title` and optional `description` |
| GET | `/tasks/{id}` | Bearer token | Get a task |
| PATCH | `/tasks/{id}` | Bearer token | Update `title`, `description`, or `done` |
| DELETE | `/tasks/{id}` | Bearer token | Delete a task |
| GET | `/health` | Public | Check PostgreSQL connectivity |
| GET | `/metrics` | Public | Expose Prometheus metrics |

JSON errors returned by the application handlers have the shape
`{"error":"message"}`. Login is limited to five requests per minute per IP
when Redis is available.

### Create your first task

This example uses `curl` and `jq`:

```bash
curl -fsS http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","name":"Ada","password":"s3cret-pass"}'

TOKEN=$(curl -fsS http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"s3cret-pass"}' | jq -er '.access_token')

curl -fsS http://localhost:8080/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Draft the new layout"}'
```

## Logs and metrics

The API uses `log/slog` to write JSON to standard output. HTTP access logs
include `method`, `path`, `route`, `status`, and `duration_ms`:

```bash
docker compose logs --tail=20 api
```

The full stack includes a preconfigured Grafana data source and dashboard.
To start the monitoring stack with the API and its dependencies, run
`docker compose up -d --build grafana`.

1. Open [Prometheus Targets](http://localhost:9090/targets) and look for the
   `task-api` target in state `UP`. Prometheus reads `api:8080/metrics` every
   15 seconds.
2. Open the [Task API dashboard](http://localhost:3000/d/task-api/task-api).
   On the first launch, sign in as `admin` with password `admin`.
3. Generate sample traffic from your terminal:

   ```bash
   for i in 1 2 3 4 5 6; do
     curl -fsS http://localhost:8080/health
     sleep 5
   done
   ```

   Wait for at least two scrapes after the first request. The dashboard shows
   requests per second and p95 request duration by route. p95 is the estimated
   time within which 95% of requests finish.

The API exports `http_requests_total`, `http_request_duration_seconds`, and
standard Go runtime and process metrics. The HTTP metrics use route patterns
such as `GET /tasks/{id}` to group requests for different task IDs. They exclude
scrapes of `/metrics`.

## How it fits together

- `cmd/api` wires the standard library HTTP server, handlers, and middleware.
  On `SIGINT` or `SIGTERM`, the server allows in-flight requests up to
  `SHUTDOWN_TIMEOUT` to finish.
- `internal/auth` handles registration, bcrypt password hashing, JWTs, and
  Redis-backed login rate limiting. Access tokens default to one hour.
- `internal/task` handles CRUD. SQL queries in `db/queries` enforce task
  ownership, and sqlc generates typed code in `internal/db`. The database
  uses PostgreSQL with a pgx connection pool.
- `cmd/migrate` runs Goose with the embedded SQL in `db/migrations`.
- When a task is created, the API attempts to publish `task.created` to the
  `task-events` Kafka topic before responding. A failed publish is logged and
  the task stays saved. `cmd/worker` reads the events and logs their payloads.
- `deploy/prometheus` holds the scrape configuration. `deploy/grafana` holds
  the data source provisioning and dashboard.

## Development commands

GitHub Actions runs `go vet ./...` and `make test-race` on pushes to
`main` and on pull requests.

| Command | Purpose |
| --- | --- |
| `make test` | Run `go test` with coverage |
| `make test-race` | Run `go test` with coverage and the race detector |
| `make lint` | Run `golangci-lint` when installed |
| `make migrate-status` | Show applied and pending migrations |
| `make tools` | Install the version of sqlc pinned in the Makefile |
| `make generate` | Regenerate Go code from SQL |

## Roadmap

Planned work includes projects and nested tasks, refresh token rotation,
email verification and password reset, pagination and filtering, and Huma
with OpenAPI documentation.

## License

[MIT](LICENSE)
