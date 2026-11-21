# Inventory Module

## 1. Overview

The inventory module provides the persistence operation for creating an inventory record associated with a product.

The current implementation consists of a narrow repository interface, a PostgreSQL repository, and a database migration defining inventory storage and integrity constraints.

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

The inventory module defines a narrow repository interface:

`Create(ctx context.Context, productID int64) error`

The interface currently exposes only the inventory creation operation.

### PostgreSQL Repository

`PostgresRepository` implements the repository interface and uses the shared `database.DBTX` abstraction.

This allows it to execute database operations through either a connection pool or an existing transaction.

A compile-time interface assertion verifies that `PostgresRepository` implements `Repository`.

### Create Operation

The `Create()` method inserts a record into the `inventory` table using the supplied product ID.

The INSERT statement specifies only the `product_id` column:

`INSERT INTO inventory (product_id) VALUES ($1)`

The database supplies the initial quantity and timestamp values through column defaults.

If the database operation fails, the repository wraps the error with the associated product ID while preserving the original error through `%w`.

The method does not perform explicit product ID validation. Referential integrity and the other database constraints are enforced by PostgreSQL.

## 5. Transaction Coordination

Inventory creation is currently coordinated by the Product service rather than through an independent inventory service.

During product creation:

1. The Product service starts a database transaction.
2. It constructs the product repository using the transaction.
3. It constructs the inventory repository using the same transaction.
4. It inserts the product and obtains its generated ID.
5. It creates the corresponding inventory record using that ID.
6. The transaction manager commits if both operations succeed.

If inventory creation fails, the service returns the error from the transaction callback. The transaction manager rolls back the transaction, including the preceding product insertion.

Using the same transaction ensures that product creation and initial inventory creation are atomic.

## 6. Migration Management

The inventory table is created by:

`20261110085818_create_inventory.sql`

The migration is managed using Goose.

The migration's `Up` section creates the inventory table and its constraints. Its `Down` section drops the inventory table.

Migrations are applied separately from API startup.

## 7. Current Scope and Limitations

The current inventory implementation supports:

- PostgreSQL persistence.
- Inventory record creation for a product.
- Database-generated initial quantities and timestamps.
- One inventory record per product.
- Product foreign-key enforcement.
- Nonnegative physical and reserved quantities.
- Enforcement that reserved quantity cannot exceed physical quantity.
- Atomic initial inventory creation as part of product creation.

The following operations and capabilities are not implemented in the current inventory repository:

- Inventory retrieval.
- Physical quantity updates.
- Inventory reservation and release.
- Stock deduction.
- Inventory adjustment workflows.
- Concurrency control for simultaneous inventory operations.

The `updated_at` column has a default value for insertion, but the migration does not define automatic timestamp updates. Future update operations will need to handle this explicitly.

Inventory reservation, stock management, and concurrency behavior remain future implementation work.
