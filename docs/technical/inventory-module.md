# Inventory Module

## 1. Overview

The inventory module manages physical stock quantities and reservations associated with products. It supports inventory retrieval, physical quantity adjustments, stock reservation, and reservation release.

The module includes a domain model, validation rules, a repository contract, a PostgreSQL implementation, and a service that enforces inventory business rules.

Inventory operations maintain `0 <= reserved_quantity <= physical_quantity`. Available quantity is calculated as `available_quantity = physical_quantity - reserved_quantity`.

Initial inventory creation is coordinated by the Product service. Product creation and its corresponding inventory record are inserted within the same database transaction.

## 2. Database Schema

The `inventory` table stores inventory quantities for products.

| Column              | Type          | Description                               |
| ------------------- | ------------- | ----------------------------------------- |
| `id`                | `BIGINT`      | Database-generated inventory identifier   |
| `product_id`        | `BIGINT`      | Identifier of the associated product      |
| `physical_quantity` | `INTEGER`     | Physical inventory quantity               |
| `reserved_quantity` | `INTEGER`     | Quantity reserved from physical inventory |
| `created_at`        | `TIMESTAMPTZ` | Record creation timestamp                 |
| `updated_at`        | `TIMESTAMPTZ` | Record update timestamp                   |

### Default Values

Both inventory quantities default to zero:

- `physical_quantity`: `0`
- `reserved_quantity`: `0`

The timestamp columns default to `CURRENT_TIMESTAMP`.

The repository inserts only the product ID. PostgreSQL supplies the initial quantities and timestamps through these defaults.

Inventory update queries explicitly set `updated_at` to `CURRENT_TIMESTAMP`. The schema has no trigger for automatic timestamp updates.

## 3. Database Constraints

The migration defines database-level constraints to protect inventory data integrity.

### Product Foreign Key

The `product_id` column references `products(id)`.

The foreign key uses `ON DELETE RESTRICT`, preventing deletion of a product while an inventory record references it.

### Unique Product Relationship

The `inventory_product_unique` constraint enforces uniqueness on `product_id`.

This means each product can have at most one inventory record.

The unique constraint also provides an index for lookups and uniqueness enforcement on `product_id`.

### Nonnegative Quantities

Two check constraints enforce the following rules:

- `physical_quantity >= 0`
- `reserved_quantity >= 0`

These prevent negative physical and reserved inventory quantities.

### Reserved Quantity Constraint

The `inventory_reserved_not_exceed_physical` constraint enforces:

`reserved_quantity <= physical_quantity`

This prevents the database from accepting an inventory state in which reserved quantity exceeds physical quantity.

## 4. Repository

The inventory repository exposes five operations.

```go
type Repository interface {
    Create(ctx context.Context, productID int64) error
    GetByProductID(ctx context.Context, productID int64) (*Inventory, error)
    AdjustPhysicalQuantity(ctx context.Context, productID int64, delta int32) error
    Reserve(ctx context.Context, productID int64, quantity int32) error
    Release(ctx context.Context, productID int64, quantity int32) error
}
```

### PostgreSQL Repository

`PostgresRepository` implements the repository interface and uses the shared `database.DBTX` abstraction.

This allows it to execute database operations through either a connection pool or an existing transaction.

A compile-time interface assertion verifies that `PostgresRepository` implements `Repository`.

| Operation | Responsibility |
|---|---|
| `Create` | Creates the initial inventory record |
| `GetByProductID` | Retrieves inventory for a product and maps `sql.ErrNoRows` to `ErrNotFound` |
| `AdjustPhysicalQuantity` | Changes physical stock by a signed delta |
| `Reserve` | Increases reserved quantity using available stock |
| `Release` | Decreases reserved quantity |

### Create Operation

The `Create()` method inserts a record into the `inventory` table using the supplied product ID.

The INSERT statement specifies only the `product_id` column:

`INSERT INTO inventory (product_id) VALUES ($1)`

The database supplies the initial quantity and timestamp values through column defaults.

If the database operation fails, the repository wraps the error with the associated product ID while preserving the original error through `%w`.

The method does not perform explicit product ID validation. Referential integrity and the other database constraints are enforced by PostgreSQL.

### Conditional Updates

Each quantity change applies the stock checks in the `WHERE` clause of the same `UPDATE` that writes the new quantity. An update that affects zero rows triggers a follow-up lookup to distinguish missing inventory from a rejected change. The outcome of that lookup is used only to report an error. For adjustments and reservations, the SQL uses `bigint` arithmetic and checks the resulting quantity against PostgreSQL's `INTEGER` maximum before storing it.

## 5. Transaction Coordination

The Product service coordinates initial inventory creation.

During product creation:

1. The Product service starts a database transaction.
2. It constructs the product repository using the transaction.
3. It constructs the inventory repository using the same transaction.
4. It inserts the product and obtains its generated ID.
5. It creates the corresponding inventory record using that ID.
6. The transaction manager commits if both operations succeed.

If inventory creation fails, the service returns the error from the transaction callback. The transaction manager rolls back the transaction, including the preceding product insertion.

Using the same transaction ensures that product creation and initial inventory creation are atomic.

### Inventory Operation Boundaries

Subsequent retrieval, physical quantity adjustments, reservations, and releases run through the Inventory service independently of the product creation transaction. Each quantity change uses an atomic conditional PostgreSQL update. PostgreSQL's row-level locking and the conditions on each update protect the quantities when requests target the same product concurrently.

## 6. Migration Management

The inventory table is created by:

`20261110085818_create_inventory.sql`

The migration is managed using Goose.

The migration's `Up` section creates the inventory table and its constraints. Its `Down` section drops the inventory table.

Migrations are applied separately from API startup.

## 7. Domain Model and Service

### Inventory Model

| Field | Type | Description |
|---|---|---|
| `ID` | `int64` | Inventory identifier |
| `ProductID` | `int64` | Associated product identifier |
| `PhysicalQuantity` | `int32` | Total physical stock |
| `ReservedQuantity` | `int32` | Stock held in reserve |
| `CreatedAt` | `time.Time` | Record creation timestamp |
| `UpdatedAt` | `time.Time` | Last update timestamp |

### Available Quantity

`AvailableQuantity()` derives available stock from `PhysicalQuantity - ReservedQuantity` each time it is called, so availability always reflects the two quantities on the model.

### Service Responsibilities

The Inventory service exposes `GetInventory`, `AdjustPhysicalQuantity`, `ReserveStock`, and `ReleaseStock`. It validates product IDs and operation quantities before calling the repository. It wraps repository errors with operation context while preserving them for `errors.Is()` checks.

### Business Rules

- Product IDs must be positive.
- A physical quantity adjustment requires a nonzero delta. The resulting quantity must be at least the reserved quantity and between `0` and `2,147,483,647`.
- A reservation requires a positive quantity and sufficient available stock.
- A release requires a positive quantity and sufficient reserved stock.

Together, these rules preserve `0 <= ReservedQuantity <= PhysicalQuantity`.

## 8. Concurrency and Data Integrity

The conditional `UPDATE` for a reservation checks `physical_quantity - reserved_quantity` against the requested amount. A physical adjustment checks the resulting physical quantity against zero, reserved stock, and the integer limit. A release checks that the reserved quantity covers the amount being released.

PostgreSQL uses row-level locking to coordinate writes to the same inventory row. Under Read Committed isolation, if the first writer commits an update, a waiting `UPDATE` re-evaluates its `WHERE` clause against the newly committed row version. A second reservation therefore succeeds only if enough stock remains. [PostgreSQL transaction isolation documentation](https://www.postgresql.org/docs/current/transaction-iso.html#XACT-READ-COMMITTED)

The stock checks and quantity changes run in one statement, avoiding a race between an application read and a subsequent write. The schema constraints also enforce the quantity invariant for writes to the table.

## 9. HTTP API

`InventoryHandler` delegates retrieval and quantity operations through the `InventoryService` interface.

### Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/products/{id}/inventory` | Retrieves product inventory |
| PATCH | `/products/{id}/inventory/physical-quantity` | Adjusts physical quantity |
| POST | `/products/{id}/inventory/reservations` | Reserves stock |
| POST | `/products/{id}/inventory/reservations/release` | Releases reserved stock |

### Request Handling

The handler parses the `id` path parameter as a positive base-10 `int64`. Physical adjustments use `adjustPhysicalQuantityRequest` with a signed `int32` field named `delta`. Reservations and releases use `stockQuantityRequest` with a positive `int32` field named `quantity`.

For quantity changes, the handler limits the request body to 1 MiB, rejects unknown JSON fields, and requires a single JSON value.

### Response Handling

Successful retrieval returns `200 OK` with an `inventoryResponse` JSON object containing `id`, `product_id`, `physical_quantity`, `reserved_quantity`, `available_quantity`, `created_at`, and `updated_at`. Successful adjustments, reservations, and releases return `204 No Content`.

### Error Handling

| Condition | HTTP status |
|---|---|
| Invalid product ID, adjustment, quantity, or request JSON | `400 Bad Request` |
| Inventory not found | `404 Not Found` |
| Insufficient physical, available, or reserved stock | `409 Conflict` |
| Oversized request body | `413 Request Entity Too Large` |
| Unexpected internal error | `500 Internal Server Error` |

Errors are returned as JSON with an `error` field. Unexpected internal errors receive a generic message that keeps database details private.

## 10. Testing

### Repository Tests

PostgreSQL integration tests cover inventory creation, successful retrieval and missing rows, physical quantity increases and decreases, invalid adjustments, and integer boundaries. They also cover successful reservations and releases, requests at the exact available or reserved quantity, and insufficient stock.

### Concurrency Tests

Repository tests exercise competing reservations and a higher-concurrency scenario with multiple goroutines. Both verify that concurrent reservations stay within available stock.

### Service and HTTP Tests

Service tests verify input validation, repository delegation, and
error propagation.

HTTP handler tests cover inventory retrieval, physical quantity
adjustments, reservations, and reservation releases. They also verify
request validation, successful responses, and error mapping.

Router tests verify that all four Inventory endpoints are correctly
registered and delegate requests to the Inventory service.

### Test Database

PostgreSQL integration tests use `TEST_DATABASE_URL` and skip when it is unset.

## 11. Current Scope

The Inventory module currently supports the following operations and guarantees.

- Atomic initial inventory creation during product creation.
- Inventory retrieval by product ID and calculation of available quantity.
- Physical quantity adjustments, stock reservations, and reservation releases.
- Business validation, PostgreSQL conditional updates, and database integrity constraints.
- Concurrency-safe individual inventory updates.
- HTTP endpoints for inventory operations.
- Service, HTTP handler, and PostgreSQL integration tests, including reservation concurrency tests.

Order creation, order cancellation, automatic stock deduction, and coordination between orders and inventory remain outside the current implementation.
