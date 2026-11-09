# Database

## Overview

The `internal/database` package initializes the application's PostgreSQL database handle and configures its connection pool. It uses Go's standard `database/sql` package with the `github.com/jackc/pgx/v5/stdlib` PostgreSQL driver.

During startup, `run()` in [`cmd/api/main.go`](../../cmd/api/main.go) calls `database.Open()` with the configured database URL and default pool configuration. If initialization fails, `run()` returns an error before starting the HTTP server, and `main()` logs the failure and exits with status `1`.

## PostgreSQL Driver

[`internal/database/postgres.go`](../../internal/database/postgres.go) imports `github.com/jackc/pgx/v5/stdlib` with the blank identifier (`_`) to run its initialization code, which registers the driver with `database/sql`, without referencing the package directly. The call `sql.Open("pgx", connectionString)` uses the registered name `pgx` to select that driver and create a database handle for the supplied connection string.

`database/sql` provides the common database interface and manages the connection pool, while pgx's stdlib adapter provides PostgreSQL-specific connectivity through that interface. Creating the handle does not guarantee an established connection; connectivity is verified separately during initialization.

## Connection Initialization

`database.Open()` first validates the supplied pool configuration, then rejects an empty connection string. If these checks pass, it calls `sql.Open("pgx", connectionString)` to create a `*sql.DB` handle and applies the configured maximum open connections, maximum idle connections, connection lifetime, and connection idle time.

The function then calls `PingContext()` with the caller's context to verify connectivity before returning the handle with a `nil` error. If any initialization step fails, it returns a `nil` handle and an error identifying the failure. When the connectivity check fails, it also attempts to close the handle, joining any cleanup error with the connectivity error before returning.

## Connection Pool Configuration

`database.DefaultConfig()` supplies the pool settings used during application startup. `database.Open()` applies them to the `*sql.DB` handle through the corresponding `database/sql` setters.

| Setting | Default | Purpose |
| --- | --- | --- |
| `MaxOpenConnections` | 25 | Limits the total number of open connections, including connections in use and idle connections |
| `MaxIdleConnections` | 10 | Limits the number of unused connections retained for reuse |
| `ConnectionMaxLifetime` | 30 minutes | Limits how long a connection may be reused before it is retired |
| `ConnectionMaxIdleTime` | 5 minutes | Limits how long a connection may remain idle before it is closed |

Lifetime and idle-time limits govern connection reuse and cleanup; they do not set query execution deadlines or interrupt an in-use connection when its lifetime expires.

`Config.Validate()` requires a positive maximum open count, a non-negative maximum idle count, and positive lifetime and idle-time durations. It also rejects `MaxIdleConnections` values greater than `MaxOpenConnections`: idle connections are part of the total open count, so their configured limit must fit within that total. A zero idle limit is valid and prevents the pool from retaining idle connections.

## Connectivity Check

`database.Open()` calls `db.PingContext(ctx)` to verify that PostgreSQL can be reached before returning the database handle. This check is necessary because successfully creating a handle with `sql.Open()` does not establish that a database connection is available.

The caller's context controls cancellation and the deadline for the connectivity check. During application startup, `run()` supplies a context with a five-second timeout. If PostgreSQL is unavailable or the context deadline expires, initialization returns an error wrapped with `ping database`, after attempting to close the handle; the HTTP server is not started.

This is a startup connectivity check, not continuous database monitoring. The `/health` endpoint returns its HTTP response without calling `PingContext()` or otherwise checking the database, so a successful health response does not confirm current database availability.

## Resource Cleanup

After successful database initialization, `run()` in `cmd/api/main.go` registers a deferred function that calls `db.Close()` to close the connection pool. During the normal shutdown path, this cleanup executes as `run()` returns, after HTTP shutdown processing, receipt of the server result, and the `HTTP server stopped` log message. If graceful HTTP shutdown fails, the application attempts forced closure before reaching database cleanup.

If `db.Close()` returns an error, the deferred function logs `Failed to close database connection` with an `error` attribute. It then wraps the error with `close database` and uses `errors.Join()` to combine it with the named return value `err`, preserving any existing server or shutdown error. A close failure therefore makes `run()` return an error even when HTTP shutdown otherwise succeeds; `main()` logs the resulting application failure and exits with status `1`.

## Testing

Tests are implemented in [`internal/database/postgres_test.go`](../../internal/database/postgres_test.go). The configuration unit tests run without a PostgreSQL instance:

- `TestConfigValidate` uses table-driven cases to check that the default configuration and a zero idle-connection limit are accepted. It checks rejection of zero maximum open connections, negative maximum idle connections, an idle limit exceeding the open limit, and zero connection lifetime or idle-time durations. These cases assert whether an error occurs, rather than its exact message.
- `TestDefaultConfig` checks all four default pool values: 25 maximum open connections, 10 maximum idle connections, a 30-minute connection lifetime, and a 5-minute connection idle time.

`TestOpen` is an integration test against the PostgreSQL instance specified by `DATABASE_URL`. It is skipped when that variable is unset or empty; otherwise, it calls `Open()` with the default pool configuration and then performs an additional `PingContext()` on the returned handle. Both operations share a five-second context, and the test defers closing the handle. PostgreSQL must be running and accessible with the supplied credentials for this test to succeed.

Run the database package tests from the repository root:

```cmd
go test ./internal/database -count=1 -v
```

Without `DATABASE_URL`, this command runs the unit tests and reports the integration test as skipped. Local environment configuration is described in the [local development setup guide](../development/local-setup.md#configure-the-environment).

The package tests do not invoke `run()` or load `.env` automatically. Supply `DATABASE_URL` and any required credentials to the test process environment to enable the integration test; URL credentials require the same percent-encoding described in the setup guide.

## Current Limitations

PostgreSQL is the only configured database, and the database package currently handles connection initialization and pool configuration. Schema migrations, database models, repositories, and application queries are not yet implemented. The application verifies connectivity during startup but does not continuously monitor database health.
