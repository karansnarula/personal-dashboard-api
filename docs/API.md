# Personal Dashboard API — Reference

Contract for the backend, written for frontend work. Everything a client needs
is on this page: base URL, auth flow, every endpoint with request and response
bodies, the error envelope, validation rules, and the exact data shape each
widget type returns.

Source of truth is the Go code in this repo. If this document and the code
disagree, the code wins; please fix the document.

## Base URL and conventions

- Local development: `http://localhost:8080`
- All routes are prefixed with `/api/v1`
- Request and response bodies are JSON. Send `Content-Type: application/json`.
- Request bodies are limited to 1 MiB.
- **Unknown JSON fields are rejected** with `400 bad_request`. Send only the
  fields documented here.
- Timestamps are RFC 3339 strings with timezone, e.g. `2026-09-14T11:49:15.310832+07:00`.
- IDs are 64-bit integers, serialised as JSON numbers.
- Every response carries an `X-Request-ID` header. Error bodies include the
  same value. Include it when reporting a problem; it appears in server logs.
- CORS is enabled for the origins listed in the server's
  `CORS_ALLOWED_ORIGINS` (default: `http://localhost:5173` and
  `http://localhost:3000`). Allowed headers: `Authorization`, `Content-Type`,
  `X-Request-ID`. `X-Request-ID` is exposed to browser code.

## Authentication

Stateless JWT (HS256). Flow:

1. `POST /auth/register` once.
2. `POST /auth/login` to get a token. Default lifetime is 24 hours
   (`expires_in` is in seconds).
3. Send `Authorization: Bearer <token>` on every protected route.
4. On `401` the token is missing, malformed, or expired. Log in again. There
   is no refresh token.
5. `POST /auth/logout` is a no-op on the server. "Logging out" means the
   client discards its token.

Protected routes: everything under `/widgets` and `/dashboard`.

## Error envelope

Every error, from any endpoint, has this shape:

```json
{
  "error": {
    "code": "validation_error",
    "message": "stock widget requires a non-empty \"ticker\"",
    "request_id": "3c37bedb-1023-4f5b-9868-c85f2d861fb0"
  }
}
```

| HTTP | `code`               | When                                                             |
|------|----------------------|------------------------------------------------------------------|
| 400  | `bad_request`        | Body is empty, malformed JSON, wrong field type, unknown field, missing required field, non-numeric `:id`, body too large |
| 400  | `validation_error`   | Body parsed but a rule failed (email format, password length, widget type/config rules) |
| 401  | `unauthorized`       | Missing/invalid/expired token, or wrong email/password on login   |
| 404  | `not_found`          | Widget does not exist **or belongs to another user**; unknown route |
| 405  | `method_not_allowed` | Known route, wrong HTTP method                                    |
| 409  | `conflict`           | Email already registered                                          |
| 500  | `internal_error`     | Unexpected server failure; details are in server logs only        |
| 503  | `service_unavailable`| Health check could not reach the database                         |

`message` is human-readable and safe to show to the user for `400`, `401`,
`404`, and `409`. Branch logic on `code` and HTTP status, not on `message`
text.

---

## Endpoints

### GET /healthz

No auth. Use it to check the backend is up before rendering the app.

- `200` → `{"status": "ok"}`
- `503` → error envelope with `service_unavailable`

### POST /auth/register

No auth.

Request:

```json
{"email": "karan@example.com", "password": "hunter2hunter2"}
```

Rules: `email` must be a valid address (trimmed, lower-cased on the server);
`password` must be 8 to 72 bytes.

- `201`

  ```json
  {"id": 1, "email": "karan@example.com"}
  ```

- `400 validation_error` on a bad email or password
- `409 conflict` if the email is already registered

### POST /auth/login

No auth.

Request:

```json
{"email": "karan@example.com", "password": "hunter2hunter2"}
```

- `200`

  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 86400
  }
  ```

- `401 unauthorized` with message `invalid email or password` for both an
  unknown email and a wrong password (deliberately indistinguishable).

### POST /auth/logout

No auth, no body.

- `200` → `{"message": "logged out; discard the token on the client"}`

---

### Widget object

Returned by every `/widgets` endpoint.

```json
{
  "id": 3,
  "user_id": 1,
  "type": "stock",
  "config": {"ticker": "AAPL"},
  "created_at": "2026-09-14T11:49:15.310832+07:00",
  "updated_at": "2026-09-14T11:49:15.310832+07:00"
}
```

`config` is always the **normalized** form (trimmed, upper-cased where
relevant, defaults filled in), which may differ from what was sent.

### Widget types and config rules

| `type`     | `config` fields                 | Rules                                                                 |
|------------|---------------------------------|-----------------------------------------------------------------------|
| `weather`  | `city` (string)                 | required, non-empty after trimming                                    |
| `news`     | `keyword` (string), `limit` (int, optional) | `keyword` required; `limit` defaults to 5, must be 1–20      |
| `stock`    | `ticker` (string)               | required; upper-cased; must match `^[A-Z0-9.\-]{1,12}$`               |
| `currency` | `base` (string), `target` (string) | both required, upper-cased, exactly 3 letters, must differ         |

Extra fields inside `config` are rejected with `400 validation_error`.

### POST /widgets

Auth required.

Request:

```json
{"type": "currency", "config": {"base": "usd", "target": "thb"}}
```

- `201` → widget object (note `config` comes back as `{"base":"USD","target":"THB"}`)
- `400 bad_request` if `type` or `config` is missing
- `400 validation_error` on an unknown `type` or a config that breaks a rule

### GET /widgets

Auth required. Returns the caller's widgets in creation order, config only,
no live data. Empty list is `[]`, never `null`.

- `200`

  ```json
  {"widgets": [ {…widget…}, {…widget…} ]}
  ```

### GET /widgets/:id

Auth required.

- `200` → widget object
- `400 bad_request` if `:id` is not a positive integer
- `404 not_found` if it doesn't exist or belongs to another user

### PATCH /widgets/:id

Auth required. Only `config` can change. The widget's `type` is fixed at
creation; the new config is validated against that type.

Request:

```json
{"config": {"city": "Tokyo"}}
```

- `200` → updated widget object with a new `updated_at`
- `400 validation_error` if the config doesn't fit the widget's type
- `404 not_found`

### DELETE /widgets/:id

Auth required.

- `204` with empty body
- `404 not_found`

---

### GET /dashboard

Auth required. Loads the caller's widgets and fetches live data for each one
from its external provider. **This call is slow by design** (currently
sequential; roughly the sum of every provider's latency, about 2 s with four
widgets). Show a loading state.

- `200`

  ```json
  {
    "mode": "sequential",
    "widget_count": 4,
    "elapsed_ms": 2169,
    "widgets": [
      {
        "widget_id": 1,
        "type": "weather",
        "config": {"city": "Bangkok"},
        "status": "ok",
        "data": {"city": "Bangkok", "temp_c": 25.94, "condition": "Clouds", "description": "overcast clouds"},
        "duration_ms": 111
      },
      {
        "widget_id": 3,
        "type": "stock",
        "config": {"ticker": "AAPL"},
        "status": "error",
        "error": "Finnhub rate limit exceeded (HTTP 429)",
        "duration_ms": 333
      }
    ]
  }
  ```

Per-widget fields:

| field         | notes                                                                 |
|---------------|-----------------------------------------------------------------------|
| `widget_id`   | matches `id` from `/widgets`                                          |
| `type`        | widget type                                                           |
| `config`      | the stored config                                                     |
| `status`      | `"ok"` or `"error"`                                                   |
| `data`        | present only when `status` is `ok`; shape depends on `type` (below)   |
| `error`       | present only when `status` is `error`; human-readable, safe to show   |
| `duration_ms` | time spent on that widget's external call                             |

Top-level `elapsed_ms` is wall-clock time for the whole assembly; `mode`
identifies the fetch strategy (`sequential` today; a concurrent mode will be
added and the value will change).

**Partial success is the contract.** One provider failing, timing out (5 s per
call), or being rate-limited never fails the request. Render `ok` widgets
normally and `error` widgets as "unavailable" with the message. Widgets are
returned in the same order as `GET /widgets`.

Free-tier providers rate-limit aggressively (Finnhub: 60 calls/min). Avoid
polling the dashboard on a tight interval.

### `data` shapes by widget type

`weather`:

```json
{"city": "Bangkok", "temp_c": 25.94, "condition": "Clouds", "description": "overcast clouds"}
```

`condition` is a short category (`Clear`, `Clouds`, `Rain`, …); `description`
is longer text.

`news`:

```json
{
  "keyword": "golang",
  "articles": [
    {"headline": "…", "source": "Kitploit.com", "url": "https://…", "published_at": "2026-09-14T03:00:00Z"}
  ]
}
```

`articles` has at most `config.limit` entries and may be empty.

`stock`:

```json
{"ticker": "AAPL", "price": 334.45, "change": 2.18, "percent_change": 0.6561, "previous_close": 332.27}
```

`change` and `percent_change` are relative to `previous_close` and can be
negative.

`currency`:

```json
{"base": "USD", "target": "THB", "rate": 33.25, "date": "2026-09-14"}
```

`rate` is how many `target` units one `base` unit buys. `date` is the
provider's quote date (business days only).

---

## Quick start for a client

```
POST /api/v1/auth/register   {"email","password"}        → 201
POST /api/v1/auth/login      {"email","password"}        → 200 {token}
POST /api/v1/widgets         {"type","config"}  + Bearer → 201
GET  /api/v1/dashboard                          + Bearer → 200
```

## Known gaps relevant to a frontend

- **CORS origins are a fixed allow-list.** If the React app runs on a port
  other than 5173 or 3000, add it to `CORS_ALLOWED_ORIGINS` in the backend's
  `.env`.
- **No refresh tokens.** Tokens last 24 h; re-login on `401`.
- **No pagination** on `GET /widgets`. Fine for a personal dashboard.
- **No `GET /auth/me`** endpoint. If the UI needs the current user's email
  after a reload, store it from the register/login response, or ask for the
  endpoint.
- Dashboard is sequential and slow; a concurrent version is planned and will
  keep the same response shape.
