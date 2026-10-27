# Go Order Management — Requirements

## 1. Purpose

Build a Go-based order management backend to develop practical skills in backend engineering, database consistency, concurrency, resilience, and distributed systems.

The initial implementation will use a modular monolith backed by PostgreSQL.

## 2. Scope

The initial system will support:

- Listing products and retrieving product details.
- Tracking product inventory.
- Creating orders containing one or more items.
- Retrieving orders and their status.
- Preventing inventory from becoming negative or being oversold.
- Handling failures without leaving inconsistent order and inventory data.

Payment processing and real customer data are outside the initial scope.

## 3. Core Domain Concepts

### Product

Represents an item available for purchase.

Initial attributes:

- Product ID
- Name
- Price
- Available stock

### Order

Represents a customer's request to purchase products.

Initial attributes:

- Order ID
- Order items
- Order status
- Creation timestamp

### Order Item

Represents a product and quantity within an order.

Initial attributes:

- Product ID
- Quantity
- Unit price at the time of ordering

These attributes are an initial proposal and will be refined during domain and database design.

## 4. Functional Requirements

### Product Management

- List available products.
- Retrieve a product by ID.
- Return a meaningful error for an unknown product.

### Order Creation

- Accept one or more order items.
- Require valid product IDs and positive integer quantities.
- Reject requests that contain no items or invalid quantities.
- Verify and reserve sufficient inventory.
- Persist the order and its items.
- Return the created order ID and result.

### Order Retrieval

- Retrieve an order by ID.
- Return its items, status, and relevant order details.
- Return a meaningful error when the order does not exist.

## 5. Inventory and Data Consistency

- Inventory must never become negative through order creation.
- Concurrent requests must not reserve more stock than is available.
- Inventory reservation and order persistence must be coordinated in a database transaction.
- If order persistence fails, the inventory change must be rolled back.
- Database constraints and atomic operations must protect critical data invariants.

## 6. Failure Handling

- Invalid requests must be rejected before inventory is modified.
- Insufficient inventory must result in a clear rejection.
- Database failures must not leave a partially committed order.
- Errors must be handled explicitly and returned using appropriate API responses.
- Retry behavior must be bounded and used only when appropriate.
- Uncertain transaction outcomes must not trigger blind retries that could duplicate effects.

## 7. Idempotency

Order creation will support an idempotency mechanism.

- A client supplies an idempotency key for a logical order-creation operation.
- Repeated requests using the same key must not create duplicate orders or reserve inventory twice.
- PostgreSQL must enforce key uniqueness.
- Key registration, order creation, and inventory reservation must be coordinated safely.
- Reusing a key with a different request payload must be handled explicitly.
- A retry of a completed operation should return the previously recorded result.

The exact persistence model, concurrent request handling, and retention policy will be defined during implementation.

## 8. Initial Non-Functional Requirements

- Maintainable Go code with clear package responsibilities.
- Safe behavior under concurrent requests.
- Explicit request validation and error handling.
- Automated unit and integration tests for critical behavior.
- Structured logging and useful operational diagnostics.
- Documentation of important design decisions and trade-offs.

Performance targets and service-level objectives will be defined after the initial design and baseline measurements.

## 9. Acceptance Criteria

The initial order workflow will be considered correct when:

- Valid orders can be created and retrieved.
- Invalid requests are rejected.
- Insufficient inventory prevents order creation.
- Concurrent orders cannot oversell inventory.
- Failed order persistence does not leave an unintended inventory deduction.
- Repeated requests with the same idempotency key do not create duplicate orders.
- Tests verify the important success, failure, and concurrency scenarios.

## 10. Open Design Questions

The following decisions remain for the architecture and implementation stages:

- Exact REST endpoint paths and response formats.
- Order status model and allowed transitions.
- Database schema, indexes, and migration strategy.
- Transaction isolation and locking strategy.
- Idempotency record design and retention.
- Authentication and authorization requirements.
- Whether and where separate services are justified.

These decisions will be documented as the project develops.
