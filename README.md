# Finance Dashboard Backend

A backend API for a role-based finance dashboard system, built in Go. Supports financial record management, user role administration, and aggregated dashboard analytics with strict access control.

---

## Quick Start

### Prerequisites
- Go 1.22+
- PostgreSQL 14+

### Setup

```bash
# 1. Clone and enter the project
git clone <repo-url>
cd finance-backend

# 2. Install dependencies
go mod tidy

# 3. Create the database
psql -U postgres -c "CREATE DATABASE finance_dashboard;"

# 4. Run the migration
psql -U postgres -d finance_dashboard -f internal/db/migrations/001_init_schema.sql

# 5. Configure environment
cp .env.example .env
# Edit .env with your DB password and a strong JWT secret

# 6. Run the server
go run ./cmd/server
```

Server starts on `http://localhost:8080`

---

## API Reference

### Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/auth/register` | None | Register a new user (default role: viewer) |
| POST | `/api/v1/auth/login` | None | Login and receive a JWT |

All protected endpoints require the header:
```
Authorization: Bearer <token>
```

### Users *(Admin only)*

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users` | List all users |
| GET | `/api/v1/users/:id` | Get a single user |
| PUT | `/api/v1/users/:id` | Update name, role, or status |
| PATCH | `/api/v1/users/:id/status` | Activate or deactivate a user |

### Financial Records

| Method | Endpoint | Roles | Description |
|--------|----------|-------|-------------|
| GET | `/api/v1/records` | All | List records (filterable, paginated) |
| GET | `/api/v1/records/:id` | All | Get a single record |
| POST | `/api/v1/records` | Admin | Create a record |
| PUT | `/api/v1/records/:id` | Admin | Update a record |
| DELETE | `/api/v1/records/:id` | Admin | Soft-delete a record |

**Query parameters for `GET /records`:**
```
?type=income|expense
?category=salaries
?from=2025-01-01
?to=2025-12-31
?page=1
?limit=20
```

### Dashboard *(Analyst and Admin)*

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/dashboard/summary` | Total income, expenses, net balance |
| GET | `/api/v1/dashboard/by-category` | Totals grouped by category and type |
| GET | `/api/v1/dashboard/trends?months=12` | Monthly income/expense breakdown |
| GET | `/api/v1/dashboard/recent?limit=10` | Most recent financial activity |

### Other

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check (no auth) |

---

## Role Permission Matrix

| Action | Viewer | Analyst | Admin |
|--------|--------|---------|-------|
| View records | ✅ | ✅ | ✅ |
| Filter records | ✅ | ✅ | ✅ |
| View dashboard summary | ❌ | ✅ | ✅ |
| View trends & analytics | ❌ | ✅ | ✅ |
| Create records | ❌ | ❌ | ✅ |
| Update records | ❌ | ❌ | ✅ |
| Delete records | ❌ | ❌ | ✅ |
| Manage users | ❌ | ❌ | ✅ |

**Design rationale:** Analysts interpret financial data — they do not originate it. Separating write authority from analytical access reduces the risk of accidental mutation and aligns with the principle of least privilege.

---

## Architecture & Design Decisions

### Project Structure

```
finance-backend/
├── cmd/server/main.go           # Composition Root — only place concrete types are instantiated
├── internal/
│   ├── config/                  # Environment configuration
│   ├── db/                      # Connection setup and SQL migrations
│   ├── middleware/              # Auth and RBAC middleware
│   ├── apierr/                  # Centralised error types and response helpers
│   └── domain/
│       ├── user/                # model · repository · service · handler
│       ├── record/              # model · repository · service · handler
│       └── dashboard/           # service · handler
└── pkg/validator/               # Reusable input validation
```

The structure is **domain-driven** rather than layer-driven. Grouping by feature (`user/`, `record/`, `dashboard/`) means all code related to a concern lives together — easier to navigate, easier to own, and easier to extract into a separate service later if needed.

---

### SOLID Principles

#### Single Responsibility
Each file in a domain package has exactly one job:
- `model.go` — data shapes and domain behaviour
- `repository.go` — database access only
- `service.go` — business logic only
- `handler.go` — HTTP decoding and response writing only

No file crosses these boundaries.

#### Open/Closed
The RBAC middleware is open for extension without modification:
```go
// Adding a new role to a route requires only passing it in — no changes to RequireRoles itself
r.With(middleware.RequireRoles(user.RoleAdmin, user.RoleAuditor)).Get("/reports", handler.Reports)
```

#### Liskov Substitution
Any struct that satisfies `user.Repository` or `record.Repository` can substitute for the Postgres implementation. This enables test doubles, in-memory implementations, or alternative database backends with zero changes to the service layer.

#### Interface Segregation
Rather than one large repository interface, capabilities are split:

```go
// user/repository.go
type UserReader interface {
    GetByID(id string) (*User, error)
    GetByEmail(email string) (*User, error)
    List() ([]*User, error)
}

type UserWriter interface {
    Create(u *User) (*User, error)
    Update(u *User) (*User, error)
}

// Repository composes both — used by the service
type Repository interface {
    UserReader
    UserWriter
}
```

The auth middleware depends only on `TokenParser` — a single-method interface carved from the user service. It never sees the full service.

#### Dependency Inversion
Services depend on repository **interfaces**, not concrete types. Concrete implementations are wired once in `main.go` and injected. The service layer has no import of `sqlx` or `lib/pq` — it is completely database-agnostic.

---

### GRASP Principles

#### Information Expert
Domain models own methods that reason about their own data:
```go
func (u *User) IsActive() bool        { return u.Status == StatusActive }
func (u *User) CanWrite() bool        { return u.Role == RoleAdmin }
func (r *Record) IsDeleted() bool     { return r.DeletedAt != nil }
func (r *Record) SignedAmount() float64 { ... }
```
Logic is placed where the data lives — not scattered across services or handlers.

#### Controller (GRASP)
HTTP handlers are GRASP Controllers: they receive system-level events (HTTP requests), decode input, delegate to the service, and write the response. They contain zero business logic.

#### Creator
The `record.Service` creates `Record` instances — not handlers, not repositories. The entity that has the creation data and context is responsible for instantiation.

#### Pure Fabrication
`dashboard.Service` is a textbook Pure Fabrication. It has no real-world counterpart in the domain but exists to cleanly house aggregation logic. It queries the database directly with `GROUP BY` and `SUM` — pulling all records into Go to aggregate would be wasteful and unscalable.

#### Low Coupling / High Cohesion
Achieved through interface-based dependency injection. Each package depends only on abstractions it defines, not on other packages' concrete types.

---

### Creational Patterns

#### Factory Functions
Every type is constructed via a `NewXxx()` function that enforces dependency provision:
```go
userRepo    := user.NewPostgresRepository(database)
userService := user.NewService(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours)
userHandler := user.NewHandler(userService)
```
Raw struct literals are never used for types with dependencies — callers cannot accidentally forget a required field.

#### Repository Pattern
All persistence is hidden behind the `Repository` interface. The service layer is completely unaware of SQL, connection pooling, or Postgres-specific behaviour. Swapping Postgres for SQLite (or an in-memory store for tests) requires implementing the interface — nothing else changes.

#### Composition Root
`cmd/server/main.go` is the single location where concrete types are instantiated and the dependency graph is assembled. No global state, no `init()` side effects, no service locator — just explicit top-to-bottom wiring. Every dependency is visible and traceable from this one file.

---

### Technology Choices

#### Go + Chi
Chi is a lightweight, idiomatic router with excellent middleware composability. Chosen over larger frameworks (Gin, Echo) to keep the code close to Go's standard `net/http` — making the HTTP layer easy to reason about without framework magic.

#### sqlx over an ORM
Raw SQL with `sqlx` struct scanning was chosen deliberately:
- Aggregation queries (dashboard) are written exactly as needed — no ORM translation layer
- Query behaviour is explicit and reviewable in the code
- No impedance mismatch between Go structs and SQL results for complex joins or computed fields
- Tradeoff: more boilerplate than GORM for simple CRUD — acceptable given the clarity benefit

#### NUMERIC(15,2) for money
`FLOAT` and `DOUBLE` cannot represent decimal fractions like `0.1` exactly in binary, causing rounding errors in financial calculations (e.g. `0.1 + 0.2 ≠ 0.3`). `NUMERIC(15,2)` stores exact decimal values — mandatory for any financial application.

#### UUIDs as primary keys
Sequential integer IDs are enumerable — a client can guess valid IDs by incrementing. UUIDs provide no ordering information, making unauthorised resource discovery significantly harder.

#### Soft Deletes
Financial records are historical data. Hard deletes would permanently destroy audit trails. `deleted_at IS NULL` filters keep deleted records invisible to queries while preserving the data — standard practice in financial systems.

#### JWT (stateless auth)
Stateless tokens eliminate the need for a session store. The token carries the user's role, so the RBAC middleware can make access decisions without a database round-trip on every request. Tradeoff: tokens cannot be individually revoked before expiry — acceptable for this use case. A token blacklist (Redis) would be the next step for production.

#### Offset Pagination
Chosen over cursor-based pagination for simplicity. For a finance dashboard with controlled write frequency, the consistency tradeoff (possible duplicates if records are inserted mid-pagination) is not a practical concern. Cursor-based pagination would be preferred for high-frequency, real-time data feeds.

---

## Assumptions

1. **New users default to `viewer` role.** An admin must explicitly promote them. This is the most conservative and secure default.
2. **Analysts cannot create records.** Analysts interpret data; they do not originate it. This enforces separation of duties.
3. **Amounts must be positive.** Record type (`income` / `expense`) carries the sign semantics. Negative amounts would be ambiguous.
4. **Deleted records are excluded from all dashboard calculations.** Soft-deleted data is considered logically non-existent for reporting purposes.
5. **A user cannot deactivate their own account.** Prevents accidental lockout of the last admin.
6. **The first admin must be created directly in the database.** There is no bootstrap admin endpoint, which would be a security risk.

---

## Potential Improvements

- **Graceful shutdown** — `signal.NotifyContext` to drain in-flight requests on SIGTERM
- **Structured logging** — `slog` (stdlib) or `zap` for JSON log output
- **Rate limiting** — `chi/middleware.Throttle` or token bucket per IP
- **Unit tests** — service layer tests with repository mock implementing the interface
- **Integration tests** — spin up a test DB, run migrations, test full request lifecycle
- **Swagger/OpenAPI** — auto-generated from handler annotations