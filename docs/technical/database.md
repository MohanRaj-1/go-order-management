# Database

## Overview

The `internal/database` package initializes the application's PostgreSQL database handle and configures its connection pool. It uses Go's standard `database/sql` package with the `github.com/jackc/pgx/v5/stdlib` PostgreSQL driver.

During startup, `run()` in [`cmd/api/main.go`](../../cmd/api/main.go) calls `database.Open()` with the configured database URL and default pool configuration. If initialization fails, `run()` returns an error before starting the HTTP server, and `main()` logs the failure and exits with status `1`.

Database schema changes are managed separately through Goose SQL migrations in the `migrations/` directory. Migrations are applied explicitly rather than automatically during API startup.

## PostgreSQL Driver

[`internal/database/postgres.go`](../../internal/database/postgres.go) imports `github.com/jackc/pgx/v5/stdlib` with the blank identifier (`_`) to run its initialization code, which registers the driver with `database/sql`, without referencing the package directly. The call `sql.Open("pgx", connectionString)` uses the registered name `pgx` to select that driver and create a database handle for the supplied connection string.

`database/sql` provides the common database interface and manages the connection pool, while pgx's stdlib adapter provides PostgreSQL-specific connectivity through that interface. Creating the handle does not guarantee an established connection; connectivity is verified separately during initialization.

## Connection Initialization

`database.Open()` first validates the supplied pool configuration, then rejects an empty connection string. If these checks pass, it calls `sql.Open("pgx", connectionString)` to create a `*sql.DB` handle and applies the configured maximum open connections, maximum idle connections, connection lifetime, and connection idle time.

The function then calls `PingContext()` with the caller's context to verify connectivity before returning the handle with a `nil` error. If any initialization step fails, it returns a `nil` handle and an error identifying the failure. When the connectivity check fails, it also attempts to close the handle, joining any cleanup error with the connectivity error before returning.

## Connection Pool Configuration

`database.DefaultConfig()` supplies the pool settings used during application startup. `database.Open()` applies them to the `*sql.DB` handle through the corresponding `database/sql` setters.

| Setting                 | Default    | Purpose                                                                                        |
| ----------------------- | ---------- | ---------------------------------------------------------------------------------------------- |
| `MaxOpenConnections`    | 25         | Limits the total number of open connections, including connections in use and idle connections |
| `MaxIdleConnections`    | 10         | Limits the number of unused connections retained for reuse                                     |
| `ConnectionMaxLifetime` | 30 minutes | Limits how long a connection may be reused before it is retired                                |
| `ConnectionMaxIdleTime` | 5 minutes  | Limits how long a connection may remain idle before it is closed                               |

Lifetime and idle-time limits govern connection reuse and cleanup; they do not set query execution deadlines or interrupt an in-use connection when its lifetime expires.

`Config.Validate()` requires a positive maximum open count, a non-negative maximum idle count, and positive lifetime and idle-time durations. It also rejects `MaxIdleConnections` values greater than `MaxOpenConnections`: idle connections are part of the total open count, so their configured limit must fit within that total. A zero idle limit is valid and prevents the pool from retaining idle connections.

## Connectivity Check

`database.Open()` calls `db.PingContext(ctx)` to verify that PostgreSQL can be reached before returning the database handle. This check is necessary because successfully creating a handle with `sql.Open()` does not establish that a database connection is available.

The caller's context controls cancellation and the deadline for the connectivity check. During application startup, `run()` supplies a context with a five-second timeout. If PostgreSQL is unavailable or the context deadline expires, initialization returns an error wrapped with `ping database`, after attempting to close the handle; the HTTP server is not started.

This is a startup connectivity check, not continuous database monitoring. The `/health` endpoint returns its HTTP response without calling `PingContext()` or otherwise checking the database, so a successful health response does not confirm current database availability.

## Resource Cleanup

After successful database initialization, `run()` in `cmd/api/main.go` registers a deferred function that calls `db.Close()` to close the connection pool. During the normal shutdown path, this cleanup executes as `run()` returns, after HTTP shutdown processing, receipt of the server result, and the `HTTP server stopped` log message. If graceful HTTP shutdown fails, the application attempts forced closure before reaching database cleanup.

If `db.Close()` returns an error, the deferred function logs `Failed to close database connection` with an `error` attribute. It then wraps the error with `close database` and uses `errors.Join()` to combine it with the named return value `err`, preserving any existing server or shutdown error. A close failure therefore makes `run()` return an error even when HTTP shutdown otherwise succeeds; `main()` logs the resulting application failure and exits with status `1`.

## Database Schema

The initial database schema consists of five tables: `customers`, `products`, `inventory`, `orders`, and `order_items`.

### Tables and Relationships

| Table         | Purpose                                                                                             |
| ------------- | --------------------------------------------------------------------------------------------------- |
| `customers`   | Stores customer information                                                                         |
| `products`    | Stores product details, SKU, price, and active status                                               |
| `inventory`   | Tracks physical and reserved quantities for products                                                |
| `orders`      | Stores customer orders and their status                                                             |
| `order_items` | Stores the products and quantities included in each order, together with their purchase-time prices |

The relationships are:

- One customer can have multiple orders.
- One order can contain multiple order items.
- One product can appear in multiple order items.
- Each product can have at most one inventory record.

The inventory relationship is enforced through a unique constraint on `inventory.product_id`. The database does not require every product to have an inventory record; the application is responsible for creating both records together.

### Data Integrity

The schema uses primary keys, foreign keys, unique constraints, and check constraints to enforce data integrity.

- Primary keys use `BIGINT GENERATED ALWAYS AS IDENTITY`.
- Customer emails and product SKUs have unique expression indexes on `LOWER(email)` and `LOWER(sku)`, respectively, to prevent case-insensitive duplicates.
- Customer and product names, and product SKUs, must not be empty or consist only of whitespace.
- Product prices and order-item unit prices use `NUMERIC(10,2)` and cannot be negative.
- Product status is restricted to `ACTIVE` and `INACTIVE`.
- Order status is restricted to `PENDING`, `CONFIRMED`, `CANCELLED`, and `FULFILLED`.
- Inventory physical and reserved quantities cannot be negative, and reserved quantity cannot exceed physical quantity.
- Order-item quantity must be greater than zero.
- A product can appear only once in an order, enforced by `UNIQUE (order_id, product_id)`.

Foreign keys connecting customers to orders, products to inventory, and orders and products to order items use `ON DELETE RESTRICT`. This prevents deletion of referenced records and preserves order and inventory history.

Order items store `unit_price` as a snapshot of the product price at the time of purchase. Changes to a product's current price therefore do not change the price recorded on existing order items.

Order totals are calculated from order-item quantities and unit prices rather than stored as a separate value.

### Indexes

Primary key and unique constraints create their corresponding indexes automatically. Additional indexes are created for query patterns that are expected to be used frequently.

| Index                              | Purpose                                                                            |
| ---------------------------------- | ---------------------------------------------------------------------------------- |
| `customers_email_lower_uidx`       | Enforces case-insensitive customer email uniqueness                                |
| `products_sku_lower_uidx`          | Enforces case-insensitive product SKU uniqueness                                   |
| `orders_customer_id_idx`           | Supports retrieving orders by customer                                             |
| `order_items_order_product_unique` | Enforces product uniqueness within an order and supports retrieving items by order |

PostgreSQL does not automatically create indexes on referencing foreign-key columns. The initial schema adds indexes based on known query patterns rather than indexing every foreign key. Additional indexes can be introduced when query requirements and performance measurements justify them.

## Schema Migrations

Schema changes are managed using [Goose](https://github.com/pressly/goose), a database migration tool that applies versioned SQL migration files.

Migration files are stored in the repository's `migrations/` directory. Goose generates timestamp-based filenames, which establish the migration order.

The initial schema is created through five migrations:

| Migration            | Description                                                                                                      |
| -------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `create_customers`   | Creates the customers table and case-insensitive email index                                                     |
| `create_products`    | Creates the products table, its constraints, and case-insensitive SKU index                                      |
| `create_inventory`   | Creates the inventory table and its quantity constraints                                                         |
| `create_orders`      | Creates the orders table, customer foreign key, status constraint, and customer index                            |
| `create_order_items` | Creates the order items table, foreign keys, quantity and price constraints, and unique order-product constraint |

Each migration contains two sections:

- `-- +goose Up`: Applies the schema change.
- `-- +goose Down`: Reverses the schema change.

Migrations are applied in timestamp order. Their applied versions are recorded in the database by Goose, allowing it to identify pending migrations.

### Migration Commands

Run migration commands from the repository root.

Validate migration files:

```cmd
goose -dir migrations validate
```

Check migration status:

```cmd
goose -dir migrations postgres "<DATABASE_URL>" status
```

Apply pending migrations:

```cmd
goose -dir migrations postgres "<DATABASE_URL>" up
```

Roll back the most recently applied migration:

```cmd
goose -dir migrations postgres "<DATABASE_URL>" down
```

The database connection string must be supplied to the Goose CLI. The application's `.env` loading through `godotenv` applies only to the API process; it does not automatically load environment variables into a separate Goose process. Local migration commands must therefore supply the connection details through the shell environment or another appropriate local configuration method.

The Goose CLI is installed separately from the API application dependencies. The API does not automatically run migrations during startup.

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

PostgreSQL is the only configured database. The `internal/database` package currently handles connection initialization and pool configuration; database models, repositories, and application queries have not yet been implemented.

The initial schema establishes table relationships and database-level constraints, but some business rules require application-level enforcement:

- Product creation must create its inventory record in the same database transaction. The schema permits a product without an inventory record.
- Order creation must include at least one order item. The schema does not independently enforce this requirement.
- Order status transitions must be validated by the application. The database check constraint restricts permitted status values but does not enforce transition rules.
- Inventory reservation and fulfillment must use appropriate transactions and concurrency control to prevent overselling.
- The application must update `updated_at` when records change. The column defaults apply during insertion and do not automatically update the timestamp.
- Email and SKU normalization, including input validation, remains an application responsibility. The unique indexes provide case-insensitive duplicate protection.
- The application must validate monetary input where values with more than two fractional digits should be rejected rather than rounded by PostgreSQL's `NUMERIC(10,2)` type.

The application verifies database connectivity during startup but does not continuously monitor database health.
