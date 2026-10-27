# Go Order Management — Architecture

## 1. Overview

The application will begin as a modular monolith written in Go, backed by PostgreSQL.

The system will have clear boundaries between HTTP transport, application and domain logic, persistence, and application startup.

The initial goal is to keep the architecture understandable, testable, and extensible without introducing unnecessary complexity.

## 2. Architectural Style

The initial implementation will use a modular monolith.

- The application will run as one deployable unit.
- Modules will have clearly defined responsibilities.
- Modules will communicate through Go functions and interfaces where appropriate.
- Separate services will be considered only when a concrete requirement justifies them.

## 3. Proposed Project Structure

```text
cmd/
  api/
    main.go

internal/
  transport/
    http/

  order/
  product/

  platform/
    database/

docs/
  architecture.md
  requirements.md
  roadmap.md
```

This structure is a starting point. Packages and directories may be adjusted as implementation details become clearer.

## 4. Package Responsibilities

### cmd/api

- Load application configuration.
- Initialize infrastructure dependencies.
- Construct application services and HTTP handlers.
- Start the HTTP server.
- Coordinate graceful shutdown.

### internal/transport/http

- Register HTTP routes.
- Decode and validate request formats.
- Call application-level operations.
- Translate application results and errors into HTTP responses.

HTTP handlers should not own order business rules or independently coordinate inventory and order SQL operations.

### internal/order

- Own order-related business rules.
- Coordinate order creation and retrieval.
- Ensure order creation follows the required workflow.
- Coordinate with persistence abstractions to preserve order and inventory consistency.

### internal/product

- Own product-related behavior.
- Support product listing and retrieval.
- Provide the product information needed by order operations through appropriate application or persistence interfaces.

### internal/platform/database

- Establish and configure PostgreSQL connectivity.
- Provide shared database infrastructure.
- Support transaction and persistence setup as the design develops.

This package should not contain order-specific business rules.

## 5. Request Flow: Create an Order

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

The order operation coordinates the complete business transaction.

Small interfaces may be used where they help separate application logic from PostgreSQL-specific implementation details and enable unit testing.

All database operations participating in order creation must use the same transaction. The transaction ownership and repository interfaces will be designed explicitly during implementation.

PostgreSQL constraints and atomic conditional operations will help enforce critical data invariants.

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

Test business behavior independently using fakes or other controlled dependencies.

### Integration tests

Verify PostgreSQL queries, constraints, transactions, and persistence behavior against a real database.

### Concurrency tests

Verify that simultaneous order requests cannot reserve more inventory than is available.

### Failure tests

Verify rollback behavior, idempotent retries, and relevant error paths.

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

- Exact package and interface boundaries.
- Database schema and migration strategy.
- Transaction implementation and locking strategy.
- REST endpoints and response formats.
- Order status transitions.
- Idempotency record model and retention.
- Authentication and authorization.
- Observability and performance targets.

These decisions will be refined as the project progresses.
