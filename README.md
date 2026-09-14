# Personal Dashboard API

A Go backend for a personal dashboard made of user-configured widgets: weather,
news, stock price, and currency rate. Users register, log in, and manage their
widgets; `GET /dashboard` fetches live data for every widget from its external
API and returns the assembled result.

The dashboard assembly currently runs **sequentially**, on purpose, with timing
instrumented. That number is the baseline a goroutine-based version will be
measured against.

## Stack

| Concern     | Choice                                                         |
|-------------|----------------------------------------------------------------|
| Language    | Go 1.26                                                        |
| HTTP        | [Gin](https://github.com/gin-gonic/gin)                        |
| Database    | Postgres 16 (Docker), [pgx/v5](https://github.com/jackc/pgx) with hand-written SQL |
| Migrations  | [goose](https://github.com/pressly/goose), embedded and applied on startup |
| Auth        | JWT (HS256) + bcrypt                                           |
| Config      | Environment variables / `.env` into a typed struct             |
| Logging     | `log/slog`, JSON                                               |
| Outbound    | Standard `net/http` client                                     |

## Architecture

```
cmd/api/                 entrypoint: config → logger → db → migrations → server
internal/
  config/                typed Config loaded from env, validated at startup
  database/              pgx pool + embedded goose migrations
  domain/                core types and sentinel errors; depends on nothing
  repository/postgres/   SQL lives here and nowhere else
  service/               business rules: auth, widget validation, dashboard assembly
  auth/                  bcrypt + JWT primitives
  widget/                one HTTP client per external provider behind a Client interface
  api/                   Gin server, router, middleware, handlers, request/response DTOs
```

Request flow:

```
handler  →  service  →  repository  →  Postgres
                ↓
           widget clients  →  external APIs
```

- **Handlers** bind and validate the request shape, call one service method,
  and map the result or error to HTTP. No SQL, no business rules.
- **Services** hold the rules and depend on interfaces they declare themselves,
  so they are unit-tested with in-memory fakes.
- **Repositories** translate pgx errors into domain errors. Every widget query
  is scoped by owner, so a foreign widget looks identical to a missing one.
- **Widget clients** parse provider responses into small internal structs and
  return errors rather than panicking. A missing API key is reported per
  widget at fetch time, not as a startup failure.

Cross-cutting: every request carries an `X-Request-ID` that appears in logs
and error responses; errors share one envelope
`{"error": {"code", "message", "request_id"}}`; the server has read, write,
and idle timeouts and shuts down gracefully on SIGINT/SIGTERM.

## Widget types

| type       | provider                                       | config                              |
|------------|------------------------------------------------|-------------------------------------|
| `weather`  | OpenWeatherMap (key required)                  | `{"city": "London"}`                |
| `news`     | NewsAPI (key required)                         | `{"keyword": "golang", "limit": 5}` |
| `stock`    | Finnhub (key required, 60 calls/min free tier) | `{"ticker": "AAPL"}`                |
| `currency` | Frankfurter (no key)                           | `{"base": "USD", "target": "EUR"}`  |

Configs are validated per type on create and update (unknown fields rejected,
tickers and currency codes upper-cased, news limit defaults to 5 and caps at
20), and stored in normalized form.

## Running locally

Requires Go 1.26+ and Docker.

```bash
cp .env.example .env        # set JWT_SECRET (32+ chars) and any API keys you have
make db-up                  # starts Postgres and waits for it to be healthy
make run                    # applies migrations, listens on :8080
```

Other targets: `make test`, `make test-db` (includes Postgres integration
tests), `make vet`, `make migrate-status`, `make db-shell`, `make db-down`.

## API

All routes are under `/api/v1`. Protected routes take
`Authorization: Bearer <token>`.

| method | path             | auth | description                                       |
|--------|------------------|------|---------------------------------------------------|
| GET    | `/healthz`       | no   | 200 if the database answers, 503 otherwise         |
| POST   | `/auth/register` | no   | `{"email","password"}` → 201 `{"id","email"}`      |
| POST   | `/auth/login`    | no   | `{"email","password"}` → `{"token","token_type","expires_in"}` |
| POST   | `/auth/logout`   | no   | no-op; JWTs are stateless, the client discards it  |
| POST   | `/widgets`       | yes  | `{"type","config"}` → 201 widget                   |
| GET    | `/widgets`       | yes  | list your widgets (config only, no live data)      |
| GET    | `/widgets/:id`   | yes  | one widget                                         |
| PATCH  | `/widgets/:id`   | yes  | `{"config"}` → updated widget; type is fixed       |
| DELETE | `/widgets/:id`   | yes  | 204                                                |
| GET    | `/dashboard`     | yes  | widgets with live data, per-widget status, timing  |

### Example

```bash
B=localhost:8080/api/v1
curl -s $B/auth/register -d '{"email":"me@example.com","password":"hunter2hunter2"}'
TOKEN=$(curl -s $B/auth/login -d '{"email":"me@example.com","password":"hunter2hunter2"}' | jq -r .token)
curl -s $B/widgets -H "Authorization: Bearer $TOKEN" -d '{"type":"currency","config":{"base":"USD","target":"EUR"}}'
curl -s $B/dashboard -H "Authorization: Bearer $TOKEN" | jq
```

### Dashboard response

```json
{
  "mode": "sequential",
  "widget_count": 3,
  "elapsed_ms": 712,
  "widgets": [
    {
      "widget_id": 1, "type": "currency", "config": {"base": "USD", "target": "EUR"},
      "status": "ok", "data": {"base": "USD", "target": "EUR", "rate": 0.86266, "date": "2026-09-11"},
      "duration_ms": 164
    },
    {
      "widget_id": 3, "type": "stock", "config": {"ticker": "AAPL"},
      "status": "error", "error": "Finnhub API key is not configured",
      "duration_ms": 0
    }
  ]
}
```

Each external call runs under its own `context.WithTimeout`
(`EXTERNAL_TIMEOUT`, default 5s). A widget whose call fails or times out is
reported with `status: "error"` and a message; the other widgets still return
data. Partial success is the contract.

## Sequential vs concurrent timing

The sequential baseline is logged on every dashboard request:

```
{"level":"INFO","msg":"dashboard assembled","mode":"sequential","widgets":4,"elapsed_ms":2169}
```

Measured on a MacBook Air with all four providers configured, one widget each:

| widget   | provider       | duration |
|----------|----------------|---------:|
| weather  | OpenWeatherMap |   111 ms |
| news     | NewsAPI        |  1163 ms |
| stock    | Finnhub        |   333 ms |
| currency | Frankfurter    |   560 ms |
| **total (sequential)** | | **2169 ms** |

The total is the sum of the parts (2167 ms) plus overhead, which is the
defining property of the sequential approach: every widget added makes the
dashboard slower by that widget's full latency. The slowest single call here
was 1163 ms, so a concurrent version should land close to that number instead
of the sum.

_Concurrent implementation and its numbers: to be added._

## Tests

```bash
make test       # unit tests: services, widget clients, HTTP handlers (fakes, no DB)
make test-db    # adds repository integration tests against the Docker Postgres
```

- `service`: widget config validation table, auth register/login, dashboard
  partial failure and per-call timeout with stub clients.
- `widget`: each provider client against `httptest` servers, including non-2xx,
  rate-limit, cancelled-context and malformed-JSON paths.
- `api/handler`: register → login → CRUD → cross-user 404 → dashboard through
  the real router and middleware with in-memory fakes.
- `repository/postgres`: user and widget repositories against a real database,
  skipped unless `TEST_DATABASE_URL` is set.
