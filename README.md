# Finance Dashboard Backend

A backend API for a role-based finance dashboard, built in Go. Supports financial record management, user role administration, and aggregated dashboard analytics with strict access control.

---

## Quick Start

**Prerequisites:** Go 1.22+, PostgreSQL 14+

```bash
git clone <repo-url>
cd finance-backend

go mod tidy

psql -U postgres -c "CREATE DATABASE finance_dashboard;"
psql -U postgres -d finance_dashboard -f internal/db/migrations/001_init_schema.sql

cp .env.example .env
# Set DB_PASSWORD and JWT_SECRET in .env

go run ./cmd/server
```

Server starts on `http://localhost:8080`.

---

## API Reference

### Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | /api/v1/auth/register | None | Register a new user (default role: viewer) |
| POST | /api/v1/auth/login | None | Login and receive a JWT |

All protected endpoints require: `Authorization: Bearer <token>`

### Users (Admin only)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/users | List all users |
| GET | /api/v1/users/:id | Get a single user |
| PUT | /api/v1/users/:id | Update name, role, or status |
| PATCH | /api/v1/users/:id/status | Activate or deactivate a user |

### Financial Records

| Method | Endpoint | Roles | Description |
|--------|----------|-------|-------------|
| GET | /api/v1/records | All | List records (filterable, paginated) |
| GET | /api/v1/records/:id | All | Get a single record |
| POST | /api/v1/records | Admin | Create a record |
| PUT | /api/v1/records/:id | Admin | Update a record |
| DELETE | /api/v1/records/:id | Admin | Soft-delete a record |

Query parameters for `GET /records`: `?type=income|expense`, `?category=salaries`, `?from=2025-01-01`, `?to=2025-12-31`, `?page=1`, `?limit=20`

### Dashboard (Analyst and Admin)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/dashboard/summary | Total income, expenses, net balance |
| GET | /api/v1/dashboard/by-category | Totals grouped by category and type |
| GET | /api/v1/dashboard/trends?months=12 | Monthly income/expense breakdown |
| GET | /api/v1/dashboard/recent?limit=10 | Most recent financial activity |

### Role Permission Matrix

| Action | Viewer | Analyst | Admin |
|--------|--------|---------|-------|
| View records | Yes | Yes | Yes |
| Filter records | Yes | Yes | Yes |
| View dashboard summary | No | Yes | Yes |
| View trends and analytics | No | Yes | Yes |
| Create records | No | No | Yes |
| Update records | No | No | Yes |
| Delete records | No | No | Yes |
| Manage users | No | No | Yes |

Analysts interpret financial data — they do not originate it. Separating write authority from analytical access reduces the risk of accidental mutation and follows the principle of least privilege.

---

## Project Structure

```
finance-backend/
├── cmd/server/main.go           # Composition root — dependency wiring happens here
├── internal/
│   ├── config/                  # Environment configuration, fail-fast on missing vars
│   ├── db/                      # Connection setup and SQL migrations
│   ├── middleware/              # JWT auth and RBAC middleware
│   ├── apierr/                  # Centralised error types and response helpers
│   └── domain/
│       ├── user/                # model · repository · service · handler
│       ├── record/              # model · repository · service · handler
│       └── dashboard/           # service · handler
└── pkg/validator/               # Reusable accumulating field validator
```

Structured by domain rather than layer — all code for a given concern lives together, easier to navigate and easier to extract later if needed.

---

## Technical Decisions

**Chi over Gin/Echo** — stays close to Go's standard `net/http`. Middleware composability is excellent and the HTTP layer is easy to reason about without framework abstractions getting in the way.

**Layered architecture with interface-based dependencies** — handlers talk to services, services talk to repositories, each through interfaces rather than concrete types. Concrete implementations are wired once in `main.go`. The service layer has no import of `sqlx` or `lib/pq` — swapping the database requires implementing the interface, nothing else changes.

**Auth and RBAC as middleware** — access control is enforced before handlers run. Adding or changing permissions on a route is a one-line change in the router. Business logic never needs to check roles.

**JWT (stateless auth)** — no session store needed. The token carries the user's role so RBAC decisions require no database round-trip per request. Tradeoff: tokens can't be revoked before expiry. A Redis-backed blacklist or refresh token pattern would address this in production.

**sqlx with raw SQL over an ORM** — dashboard aggregation queries are written exactly as needed with `GROUP BY` and `SUM`. No translation layer, no impedance mismatch. Query behaviour is explicit and reviewable in the code.

**`NUMERIC(15,2)` for money** — `FLOAT` cannot represent decimal fractions exactly in binary, causing rounding errors in financial calculations. Not optional here.

**UUIDs as primary keys** — sequential integer IDs are enumerable. UUIDs make unauthorised resource discovery significantly harder.

**Soft deletes** — financial records are historical data. `deleted_at IS NULL` is the base condition on every query. Data is never permanently removed.

**Parameterised dynamic queries** — record filtering builds `WHERE` clauses dynamically using `$N` positional parameters throughout. Values are never string-interpolated, so SQL injection is prevented by construction.

**Connection pool tuning** — `MaxOpenConns`, `MaxIdleConns`, and `ConnMaxLifetime` are explicitly set rather than left at defaults.

---

## Security

- Passwords hashed with bcrypt. The hash field is tagged `json:"-"` and cannot appear in any API response.
- Login errors are intentionally vague ("invalid email or password") to prevent user enumeration.
- No bootstrap admin endpoint — the first admin is created directly in the database.
- New users default to `viewer`. An admin must explicitly promote them.
- A user cannot deactivate their own account, preventing accidental lockout.

---

## Known Limitations

- JWT tokens cannot be revoked before expiry. Refresh tokens or a Redis blacklist would fix this.
- Offset pagination degrades on very large tables. Cursor-based is the natural next step.
- No integration tests yet. The repository abstraction makes adding them straightforward — implement the interface against a test database, run migrations, test the full request lifecycle.
- No graceful shutdown on SIGTERM.
- No structured logging (JSON log output via `slog` or `zap` would be the next addition).