# Customer module

The `internal/customer` package supports registering customers and retrieving them by ID. The service depends on a repository interface so it can enforce input rules independently of PostgreSQL.

## Domain model and validation

A `Customer` contains an ID, name, email, and creation and update timestamps. `NewCustomer` trims surrounding whitespace from the name and email before validating them. Names must be non-empty and at most 150 Unicode code points. Emails must be non-empty, at most 254 bytes, and accepted by Go's mail parser as a standalone address. The constructor preserves the email's case. PostgreSQL assigns the ID and timestamps when the customer is stored.

## Service and repository

- `CreateCustomer` validates input before calling the repository and returns the persisted customer with its generated fields.
- `GetCustomerByID` rejects zero and negative IDs before calling the repository. For a positive ID, it returns the customer found by the repository.
- `Repository` defines the persistence contract. `Create` fills in the supplied customer with the stored values, and `GetByID` loads a customer by its primary key. Service tests use a fake implementation to verify these use cases without connecting to PostgreSQL.

## PostgreSQL implementation

`PostgresRepository` uses parameterized queries against the `customers` table. It reads back the stored row on insert and selects by primary key on lookup. It accepts a `database.DBTX`, so the caller can supply either a database pool (`*sql.DB`) or an existing transaction (`*sql.Tx`).

PostgreSQL enforces case-insensitive email uniqueness through a unique index on `LOWER(email)`. See the [database schema](database.md#database-schema) for the table constraints.

## Error handling

| Condition | Error |
| --- | --- |
| Invalid name | `ErrInvalidName` |
| Invalid email | `ErrInvalidEmail` |
| Zero or negative customer ID | `ErrInvalidCustomerID` |
| No customer with the requested ID | `ErrNotFound` |

The PostgreSQL repository translates `sql.ErrNoRows` into `ErrNotFound` and wraps other database errors, including email uniqueness violations. The service returns a nil customer on failure and adds operation context to repository errors using `%w`. Callers can use `errors.Is(err, customer.ErrNotFound)` to recognize a missing customer through that wrapper.
