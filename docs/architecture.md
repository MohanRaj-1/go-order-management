# Go Order Management — Architecture

## 1. Overview

The application is a modular monolith written in Go, backed by PostgreSQL.

It separates HTTP transport, application and domain logic, persistence, and application startup into cohesive packages with defined responsibilities.

The architecture aims to remain understandable, testable, and extensible without introducing unnecessary complexity.

## 2. Architectural Style

The application uses a modular monolith.

- The application runs as one deployable unit.
- Modules have clearly defined responsibilities.
- Modules communicate through Go functions and interfaces where appropriate.
- Separate services will be considered only when a concrete requirement justifies them.

## 3. Project Structure

```text
cmd/
  api/
    main.go

internal/
  config/
  customer/
  database/
  http/
  inventory/
  logger/
  order/       (planned)
  product/

migrations/
docs/
```

## 4. Package Responsibilities

### cmd/api

- Load application configuration.
- Initialize infrastructure dependencies.
- Construct application services and HTTP handlers.
- Start the HTTP server.
- Coordinate graceful shutdown.

### internal/http

- Register HTTP routes.
- Decode and validate request formats.
- Call application-level operations.
- Translate application results and errors into HTTP responses.

HTTP handlers should not own order business rules or independently coordinate inventory and order SQL operations.

### internal/customer

Owns customer registration and lookup, including input validation and persistence through a repository interface. The PostgreSQL implementation stores and retrieves customers, while the service defines the rules callers rely on. See the [customer module guide](technical/customer-module.md).

### internal/order (planned)

- Will own order-related business rules.
- Will coordinate order creation and retrieval.
- Will ensure order creation follows the required workflow.
- Will coordinate with persistence abstractions to preserve order and inventory consistency.

### internal/product

- Owns the product domain model and input validation.
- Defines the product repository contract.
- Implements product persistence through PostgreSQL.
- Coordinates product creation and initial inventory creation within a single database transaction.

### internal/inventory

- Defines the inventory domain model and repository contract.
- Implements inventory persistence through PostgreSQL.
- Provides operations for retrieving inventory, adjusting physical
  quantity, reserving stock, and releasing reservations.
- Creates the initial inventory record when requested by the Product
  service as part of product creation.
- Uses conditional database updates to protect inventory quantity
  invariants during concurrent operations.
- Relies on database constraints to enforce inventory quantity and
  product relationship invariants.

### internal/database

- Establishes and configures PostgreSQL connectivity.
- Configures the database connection pool.
- Defines the shared DBTX interface for database operations.
- Provides transaction management through TransactionManager.
- Keeps transaction infrastructure independent of module-specific business rules.

## 5. Planned Request Flow: Create an Order

1. The HTTP handler receives an order-creation request.
2. The handler decodes the request and validates its structure.
3. The order application operation validates the relevant business rules.
4. A database transaction begins.
5. Inventory is reserved only when sufficient stock is available.
6. The order and its items are persisted.
7. The transaction commits if all required operations succeed.
8. On failure, the transaction is rolled back where possible.
9. The handler returns an appropriate HTTP response.

Inventory reservation and order persistence must participate in the same transaction so that a failed order creation does not leave an unintended inventory deduction.

The implementation must also protect against concurrent requests overselling inventory.

## 6. Persistence and Transaction Boundaries

The application uses PostgreSQL through Go's `database/sql` package and the pgx standard-library adapter.

Repositories depend on the `database.DBTX` interface, which is implemented by both `*sql.DB` and `*sql.Tx`. This allows repository operations to execute through either the connection pool or an existing transaction.

The application service coordinates the business operation, while the transaction manager owns transaction lifecycle management.

Product creation demonstrates this approach:

1. The product service validates the product input.
2. The transaction manager begins a database transaction.
3. The product repository creates the product.
4. The inventory repository creates its initial inventory record using the same transaction.
5. The transaction commits only when both operations succeed.
6. If either operation fails, the transaction is rolled back.

The transaction manager owns transaction lifecycle management but does not contain module-specific business rules.

Order creation will follow the same principle: all persistence operations participating in one business transaction must use the same transaction.

## 7. Error Handling and Idempotency

- Invalid requests must be rejected clearly.
- Insufficient inventory must not create an order.
- Database failures must not leave partially committed order data.
- Retries must be bounded and used only when appropriate.
- Order creation must support idempotency keys to prevent duplicate logical orders.
- Uncertain transaction outcomes must be handled without blindly repeating operations that could duplicate effects.

Detailed retry policies and idempotency persistence will be designed during implementation.

## 8. Testing Strategy

### Unit tests

Test business behavior independently using fakes or other controlled
dependencies.

### Integration tests

Verify PostgreSQL queries, constraints, transactions, and persistence
behavior against a real database.

### Concurrency tests

Verify that simultaneous inventory operations cannot reserve more
stock than is available.

Order-specific concurrency tests will be introduced when the Order
module is implemented. They will verify that concurrent order requests
cannot oversell inventory.

### Failure tests

Verify transaction rollback behavior and relevant error paths.

Idempotency and retry tests will be introduced alongside the
corresponding Order functionality.

## 9. Architectural Principles

- Keep responsibilities cohesive and boundaries clear.
- Prefer simple designs over unnecessary abstractions.
- Keep transport-specific concerns separate from business operations.
- Protect critical invariants at the database level where appropriate.
- Make failures and cancellation behavior explicit.
- Use tests to verify correctness rather than relying on assumptions.
- Introduce new infrastructure or separate services only when justified.

## 10. Open Design Decisions

The following decisions remain open:

- Order-specific package and interface boundaries.
- Order-level transaction coordination and inventory concurrency strategy.
- REST endpoints and response formats for order operations.
- Order status transitions.
- Idempotency record model and retention.
- Authentication and authorization.
- Observability and performance targets.

These decisions will be refined as the project progresses.
