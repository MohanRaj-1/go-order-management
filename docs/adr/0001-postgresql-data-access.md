# ADR 0001: PostgreSQL Data Access

- Status: Accepted
- Date: 2026-11-03

## Context

The order-management application requires persistent storage for related product, order, order-item, and inventory data. Its requirements include preventing overselling and ensuring that inventory changes and order persistence succeed or fail together. A relational database provides a suitable foundation through relationships, constraints, and transactions; these mechanisms must be designed and used correctly to enforce the application's consistency rules.

PostgreSQL supports these needs as a persistent relational database and is already the database specified by the project's requirements and initial architecture. An in-memory store alone would not meet the persistence requirement across application restarts. Other relational databases could also meet the core requirements; the project does not currently identify a need for an additional database engine.

Database access needs a boundary separate from HTTP request handling so persistence details and transaction coordination can evolve without becoming coupled to transport code. The current `internal/database` package establishes shared connection infrastructure; domain persistence operations and their interfaces remain to be implemented. The Go integration should support explicit SQL and transaction control while keeping connection-pool management separate from application business rules.

## Decision

Use PostgreSQL as the application's database, accessed through Go's standard `database/sql` interface and the `github.com/jackc/pgx/v5/stdlib` driver. The application uses pgx through its `database/sql` adapter rather than calling pgx's native API directly.

Manage database connections through a shared `*sql.DB` handle, with pool configuration and initialization centralized in `internal/database`. During startup, `run()` calls `database.Open()` to create the handle, apply pool settings, and verify connectivity before starting the HTTP server. If initialization fails, application startup returns an error without starting the server.

## Alternatives Considered

- In-memory storage: simple to implement, but data would be lost when the application restarts. It does not meet the requirement for persistent order and inventory data.
- Another relational database: could provide the required relationships, constraints, and transactions. PostgreSQL already fits the project's requirements and architecture, and no current requirement justifies choosing a different database.
- pgx native API: would provide direct access to pgx functionality, but would tie database access more closely to its driver-specific interfaces. The stdlib adapter supports the current connection-management needs while retaining Go's standard `database/sql` interface.

## Consequences

### Benefits

- PostgreSQL provides relational modelling, constraints, and transactions as a foundation for persistent, consistent order and inventory data.
- `database/sql` supplies a standard Go database interface and manages connection pooling, with PostgreSQL connectivity provided by pgx's stdlib adapter.
- Centralized initialization and pool configuration keep connection management separate from HTTP handlers and provide one place to validate settings and verify startup connectivity.

### Trade-offs and Limitations

- The standard interface does not make database engines interchangeable. PostgreSQL-specific SQL and database behavior can still require changes when moving to another engine.
- Access through `database/sql` does not expose every capability available through pgx's native API. Requirements for those capabilities would need a separate integration decision.
- This decision establishes connection infrastructure, not the complete persistence layer. Schema migrations, repositories, and transaction workflows remain to be implemented, including the constraints and concurrency controls needed to enforce order and inventory consistency.

## Implementation

The decision is reflected in the following files:

| File | Responsibility |
| --- | --- |
| [`internal/database/postgres.go`](../../internal/database/postgres.go) | Registers the pgx driver, validates pool configuration, initializes the database handle, applies pool settings, and verifies connectivity |
| [`internal/database/postgres_test.go`](../../internal/database/postgres_test.go) | Tests pool defaults and validation, with a database initialization integration test that runs when `DATABASE_URL` is supplied |
| [`cmd/api/main.go`](../../cmd/api/main.go) | Initializes the database before HTTP startup and defers pool cleanup until `run()` returns |
| [`go.mod`](../../go.mod) | Declares the `github.com/jackc/pgx/v5` dependency that provides the stdlib adapter |

Connection lifecycle details and test instructions are documented in the [database technical guide](../technical/database.md).
