# Finance Dashboard Backend

A backend API for a role-based finance dashboard system, built in Go. Supports financial record management, user role administration, and aggregated dashboard analytics with strict access control.

The primary goal of this project was to apply OOAD design principles — SOLID, GRASP, and structural patterns — in an idiomatic Go codebase, not just make things work.

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

Analysts interpret financial data — they do not originate it. Separating write authority from analytical access reduces the risk of accidental mutation and aligns with the principle of least privilege.

---

## Architecture

### Project Structure

```
finance-backend/
├── cmd/server/main.go           # Composition Root — only place concrete types are instantiated
├── internal/
│   ├── config/                  # Environment configuration, fail-fast on missing vars
│   ├── db/                      # Connection setup and SQL migrations
│   ├── middleware/              # Auth (JWT) and RBAC middleware
│   ├── apierr/                  # Centralised error types and response helpers
│   └── domain/
│       ├── user/                # model · repository · service · handler
│       ├── record/              # model · repository · service · handler
│       └── dashboard/           # service · handler
└── pkg/validator/               # Reusable, accumulating field validator
```

The structure is domain-driven rather than layer-driven. All code for a given concern lives together — easier to navigate and easier to extract into a separate service later if needed.

### Request Lifecycle

Every protected request goes through this sequence, in order:

```
Chi router matches path + method
  → Logger / Recoverer / RealIP / RequestID  (global)
  → middleware.Authenticate                  (parse JWT, inject claims, or 401)
  → middleware.RequireRoles                  (check claims.Role, or 403)
  → Handler                                  (decode input, call service)
  → Service                                  (validate, apply business rules, call repo)
  → Repository                               (execute SQL, return data)
  → Service                                  (assemble response)
  → Handler                                  (write JSON)
```

Steps 3 and 4 are the gatekeepers. The handler never runs if they fail. The service never runs if the handler can't decode input. Each layer has exactly one job.

---

## Design Decisions

### SOLID in Practice

**Single Responsibility** — each file in a domain package has one job: `model.go` owns data shapes and domain behaviour, `repository.go` owns database access, `service.go` owns business logic, `handler.go` owns HTTP decoding and response writing. No file crosses these boundaries.

**Open/Closed** — the RBAC middleware is open for extension without modification. Adding a new role to a route requires only passing it in:

```go
r.With(middleware.RequireRoles(user.RoleAdmin, user.RoleAuditor)).Get("/reports", handler.Reports)
```

`RequireRoles` itself never changes.

**Liskov Substitution** — any struct satisfying `user.Repository` or `record.Repository` can substitute for the Postgres implementation. This enables test doubles or in-memory stores with zero changes to the service layer.

**Interface Segregation** — the auth middleware doesn't import the user service package. It depends only on a single-method interface it defines itself:

```go
type TokenParser interface {
    ParseToken(token string) (*user.Claims, error)
}
```

The user service satisfies this implicitly. The middleware can't accidentally call `Register()` or `Login()` — it can only call `ParseToken`.

**Dependency Inversion** — services depend on repository interfaces, not concrete types. Concrete implementations are wired once in `main.go` and injected. The service layer has no import of `sqlx` or `lib/pq` — it is completely database-agnostic.

### GRASP in Practice

**Information Expert** — domain models own methods that reason about their own data:

```go
func (u *User) IsActive() bool         { return u.Status == StatusActive }
func (u *User) CanWrite() bool         { return u.Role == RoleAdmin }
func (r *Record) IsDeleted() bool      { return r.DeletedAt != nil }
func (r *Record) SignedAmount() float64 { ... }
```

Logic lives where the data lives.

**Controller** — HTTP handlers are GRASP Controllers: they receive HTTP requests, decode input, delegate to the service, and write the response. They contain zero business logic.

**Creator** — the service creates domain entity instances, not handlers or repositories. The entity that has the creation context is responsible for instantiation.

**Pure Fabrication** — `dashboard.Service` has no real-world counterpart in the domain but exists to cleanly house aggregation logic. It queries the database directly with `GROUP BY` and `SUM`. Pulling all records into Go to aggregate would be wasteful and unscalable, but forcing it through a standard repository interface would also be wrong — the abstraction would be leaky. Pure Fabrication is the right call here.

### Composition Root

`cmd/server/main.go` is the single location where concrete types are instantiated and the dependency graph is assembled. No global state, no `init()` side effects, no service locator — just explicit top-to-bottom wiring. If you want to understand what depends on what, this is the only file you need.

```go
userRepo    := user.NewPostgresRepository(database)
userService := user.NewService(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours)
userHandler := user.NewHandler(userService)
```

The full dependency graph is visible and traceable from one place.

### Technology Choices

**Chi over Gin/Echo** — Chi is a lightweight, idiomatic router with excellent middleware composability. Staying close to `net/http` means the HTTP layer is easy to reason about without framework magic. Middleware chains map directly to the layered concerns above.

**sqlx over an ORM** — raw SQL with struct scanning was a deliberate choice. Aggregation queries for the dashboard are written exactly as needed, query behaviour is explicit and reviewable in the code, and there's no impedance mismatch for complex computed fields. The tradeoff is more boilerplate for simple CRUD — acceptable given the clarity benefit.

**`NUMERIC(15,2)` for money** — `FLOAT` and `DOUBLE` cannot represent decimal fractions exactly in binary. `NUMERIC(15,2)` stores exact decimal values. This is not optional in a financial application.

**UUIDs as primary keys** — sequential integer IDs are enumerable. A client can guess valid IDs by incrementing. UUIDs provide no ordering information, making unauthorised resource discovery significantly harder.

**Soft deletes** — financial records are historical data. Hard deletes permanently destroy audit trails. `deleted_at IS NULL` filters keep deleted records invisible to queries while preserving the data. Standard practice in financial systems.

**JWT (stateless auth)** — stateless tokens eliminate the need for a session store. The token carries the user's role, so RBAC decisions require no database round-trip per request. Tradeoff: tokens cannot be individually revoked before expiry. A token blacklist backed by Redis would be the production next step.

**Offset pagination** — chosen over cursor-based for simplicity. For a finance dashboard with controlled write frequency, the consistency tradeoff (possible duplicates if records are inserted mid-pagination) is not a practical concern. Cursor-based pagination is the better choice for high-frequency real-time feeds.

---

## Security Notes

- New users default to `viewer` role. An admin must explicitly promote them — the most conservative secure default.
- No bootstrap admin endpoint exists. The first admin is created directly in the database. An admin registration endpoint would be a security risk.
- Passwords are hashed with `bcrypt` at `DefaultCost`. The plaintext password never touches the database. The hash field has `json:"-"` and cannot appear in any API response.
- Login errors are intentionally vague ("invalid email or password") to prevent user enumeration.
- A user cannot deactivate their own account. Prevents accidental lockout of the last admin.
- SQL injection is prevented by construction: all dynamic query clauses use `$N` positional parameters. Values are never string-interpolated into queries.

---

## Known Limitations and Potential Improvements

- JWT tokens cannot be revoked before expiry. A Redis-backed token blacklist or a refresh token pattern would address this for production.
- Offset pagination degrades on very large tables. Cursor-based pagination is the natural upgrade.
- No graceful shutdown. `signal.NotifyContext` to drain in-flight requests on SIGTERM would be the first production concern.
- No structured logging. `slog` (stdlib) or `zap` for JSON log output.
- No rate limiting. `chi/middleware.Throttle` or a token bucket per IP.
- The service layer is fully testable with mock repositories (the interface makes this straightforward), but unit tests are not yet written.
- No integration tests against a live database. The repository abstraction makes this easy to add: spin up a test DB, run migrations, test the full request lifecycle.

---

## Assumptions

- Amounts must be positive. Record type (`income` / `expense`) carries the sign semantics. A negative amount would be ambiguous.
- Soft-deleted records are excluded from all dashboard calculations. Logically deleted data is not real for reporting purposes.
- Analysts cannot create records. Analysts interpret data; they do not originate it. This enforces separation of duties.