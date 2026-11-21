# Product Module

## 1. Overview

The product module manages product creation and coordinates the creation of the corresponding initial inventory record.

It contains the product domain model, validation rules, repository contract, PostgreSQL persistence implementation, and service responsible for coordinating the creation workflow.

Product creation and initial inventory creation are performed within a single database transaction. This ensures that a product is not persisted without its corresponding inventory record.

## 2. Domain Model

The `Product` structure represents a product in the application.

| Field         | Type            | Description                                               |
| ------------- | --------------- | --------------------------------------------------------- |
| `ID`          | `int64`         | Database-generated product identifier                     |
| `SKU`         | `string`        | Product stock keeping unit                                |
| `Name`        | `string`        | Product name                                              |
| `Description` | `string`        | Product description                                       |
| `Price`       | `int64`         | Product price represented in minor currency units (paise) |
| `Status`      | `ProductStatus` | Current product status                                    |
| `CreatedAt`   | `time.Time`     | Record creation timestamp                                 |
| `UpdatedAt`   | `time.Time`     | Record update timestamp                                   |

### Product Status

`ProductStatus` is a string-based type representing the product's lifecycle status.

The currently defined values are:

- `ProductStatusActive`
- `ProductStatusInactive`

New products are created with `ProductStatusActive` by default.

## 3. Monetary Representation

Product prices are represented as `int64` values in paise rather than floating-point numbers.

For example:

- `12550` represents ₹125.50.
- `10000` represents ₹100.00.

This avoids floating-point precision problems when handling monetary values.

The maximum supported price is `9,999,999,999` paise, corresponding to the database column's `NUMERIC(10,2)` precision.

The PostgreSQL repository converts the integer representation into `pgtype.Numeric` with an exponent of `-2`. This preserves the intended decimal value without using floating-point conversion.

## 4. Product Construction and Validation

`NewProduct()` is responsible for constructing a valid product.

It trims leading and trailing whitespace from the SKU, name, and description before creating the product.

The following validations are applied:

| Field | Validation                                                    |
| ----- | ------------------------------------------------------------- |
| SKU   | Must not be empty and must not exceed 50 Unicode code points  |
| Name  | Must not be empty and must not exceed 150 Unicode code points |
| Price | Must be between 0 and 9,999,999,999 paise                     |

A zero price is valid.

Length validation uses `utf8.RuneCountInString`, so the limits are based on Unicode code points rather than UTF-8 byte length.

The constructor returns an error if validation fails. Otherwise, it returns a product initialized with the active status.

The description is trimmed but does not currently have a separate validation rule.

## 5. Domain Errors

The product package defines sentinel errors for validation failures:

- `ErrInvalidSKU`
- `ErrInvalidName`
- `ErrInvalidPrice`

These errors allow callers and tests to identify specific validation failures using `errors.Is()`.

## 6. Repository

The `Repository` interface defines the persistence operation currently required by the product module:

`Create(ctx context.Context, product *Product) error`

The interface is intentionally narrow and represents the product creation operation.

### PostgreSQL Repository

`PostgresRepository` implements the repository contract and uses the shared `database.DBTX` interface.

This allows the repository to execute queries through either a database connection pool or a transaction.

A compile-time interface assertion verifies that `PostgresRepository` implements `Repository`.

The `Create()` method:

1. Rejects a nil product.
2. Executes an INSERT statement against the `products` table.
3. Uses `RETURNING` to retrieve the generated ID and database timestamps.
4. Converts the product price from paise into `pgtype.Numeric`.
5. Populates the product's ID, creation timestamp, and update timestamp after a successful database operation.

Database errors are wrapped with product context while preserving the original error through `%w`.

## 7. Service and Transaction Coordination

The `Service` coordinates the product creation workflow.

It receives a `*database.TransactionManager` through `NewService()` and exposes `CreateProduct()` as the application operation for creating a product.

### Creation workflow

The operation follows this sequence:

1. Calls `NewProduct()` to construct and validate the product.
2. Returns immediately if domain validation fails.
3. Starts a database transaction using `WithinTransaction()`.
4. Constructs the PostgreSQL product repository using the transaction.
5. Constructs the PostgreSQL inventory repository using the same transaction.
6. Inserts the product and retrieves its generated ID.
7. Creates the initial inventory record using that product ID.
8. Returns from the transaction callback successfully so the transaction manager can commit.
9. Returns the product after the transaction completes successfully.

Both repository operations use the same `*sql.Tx`. Consequently, product creation and initial inventory creation belong to the same atomic database operation.

### Error handling and rollback

If product persistence fails, the service wraps and returns the error from the transaction callback.

If initial inventory creation fails, the service also returns the wrapped error from the callback.

In either case, the transaction manager receives an error and rolls back the transaction. This prevents a product record from remaining in the database when its initial inventory record cannot be created.

The service returns `nil` and the transaction error when the transaction does not complete successfully.

### Responsibility boundaries

The implementation keeps responsibilities separated:

- **Domain model and constructor:** Represent product data and enforce product validation.
- **Repository:** Execute product persistence operations.
- **Service:** Coordinate the product creation workflow and its business transaction.
- **Transaction manager:** Manage transaction lifecycle, commit, and rollback.

The service directly constructs the concrete PostgreSQL repositories using the transaction. No additional repository factory or abstraction is introduced for this workflow.

## 8. Service Tests

The product service has PostgreSQL integration tests covering successful creation, transaction rollback, and invalid input.

These tests use `TEST_DATABASE_URL` to connect to the test database. If the environment variable is not configured, the integration tests are skipped.

Database operations use contexts with a five-second timeout. Test records and database objects are cleaned up after execution.

### Successful Product Creation

`TestService_CreateProduct_Success` verifies the complete product creation workflow.

It checks that:

- The service successfully creates a product.
- The returned product is non-nil.
- The database-generated product ID is positive.
- `CreatedAt` and `UpdatedAt` are populated.
- The product record exists in the database after the operation.
- The corresponding inventory record exists.
- Both physical and reserved inventory quantities are initially zero.

This test verifies that the transaction commits the product and its initial inventory record.

### Rollback When Inventory Creation Fails

`TestService_CreateProduct_RollsBackWhenInventoryCreationFails` verifies transaction atomicity.

The test installs a PostgreSQL trigger that deliberately raises an exception when the initial inventory record is inserted for the test product.

The service is then called to create the product.

The test verifies that:

- The service returns an error.
- The error retains the underlying PostgreSQL exception.
- The PostgreSQL error has SQLSTATE `P0001` and the expected exception message.
- The returned product is nil.
- No product record remains in the database.

This confirms that a successful product insertion is rolled back when the subsequent inventory insertion fails.

The test removes the trigger and its function during cleanup.

Because the test uses fixed trigger and function names, concurrent executions against the same test database should be avoided.

### Invalid Input

`TestService_CreateProduct_InvalidInput` verifies that domain validation happens before database transaction handling.

The test constructs the service with a nil transaction manager and submits an empty SKU.

It verifies that:

- The returned error matches `ErrInvalidSKU` using `errors.Is()`.
- The returned product is nil.

Since the transaction manager is nil, successful completion of this test also demonstrates that invalid input is rejected before the service attempts to start a transaction.

### Test Coverage Summary

| Scenario                     | Verification                                 |
| ---------------------------- | -------------------------------------------- |
| Successful creation          | Product and initial inventory are committed  |
| Generated database fields    | ID and timestamps are populated              |
| Initial inventory            | Physical and reserved quantities are zero    |
| Inventory insertion failure  | Transaction rolls back                       |
| PostgreSQL error propagation | Original database error remains discoverable |
| Invalid SKU                  | Validation fails before transaction handling |

## 9. Current Scope

The documented implementation covers product construction, validation, PostgreSQL persistence, atomic creation of the initial inventory record, and service-level integration testing.

Other product operations, such as retrieval, update, deletion, and status changes, are not part of the repository contract shown here.
