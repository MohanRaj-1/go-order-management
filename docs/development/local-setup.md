# Local Development Setup

This document describes how to configure the local development environment, provision the PostgreSQL database, run tests, and start the Order Management API.

## Prerequisites

The following tools are required:

| Dependency                       | Version                                          | Purpose                                               |
| -------------------------------- | ------------------------------------------------ | ----------------------------------------------------- |
| Go                               | 1.27.3                                           | Application development and execution                 |
| PostgreSQL                       | 18.6 (tested)                                    | Application database                                  |
| PostgreSQL client tools (`psql`) | Compatible with the installed PostgreSQL version | Database administration and connectivity verification |
| Git                              | —                                                | Source control                                        |

The project uses [pgx](https://github.com/jackc/pgx), through Go's `database/sql` interface, as its PostgreSQL driver.

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/MohanRaj-1/go-order-management.git
cd go-order-management
```

### 2. Verify the Go installation

```bash
go version
```

The project currently targets Go 1.27.3.

### 3. Install project dependencies

```bash
go mod download
```

### 4. Provision PostgreSQL

Ensure that the PostgreSQL service is running.

Connect to PostgreSQL using an administrative account:

```bash
psql -U postgres -h localhost -p 5432
```

Create an application role:

```sql
CREATE ROLE order_app WITH LOGIN PASSWORD '<your-password>';
```

Create the application database and assign ownership to the application role:

```sql
CREATE DATABASE order_management OWNER order_app;
```

Exit the PostgreSQL shell:

```sql
\q
```

### 5. Verify database connectivity

Connect using the application role:

```bash
psql -U order_app -h localhost -p 5432 -d order_management
```

When password authentication is required, allow `psql` to prompt for the password interactively. The password characters are not displayed, and you do not need to set `PGPASSWORD` for this check.

Verify the connected role and database:

```sql
SELECT current_user, current_database();
```

The expected values are:

- `current_user`: `order_app`
- `current_database`: `order_management`

Exit the PostgreSQL shell:

```sql
\q
```

## Configuration

The application reads its configuration from environment variables.

### Environment variables

| Variable                   | Required | Default | Description                                     |
| -------------------------- | -------- | ------- | ----------------------------------------------- |
| `DATABASE_URL`             | Yes      | —       | PostgreSQL connection string                    |
| `HTTP_ADDR`                | No       | `:8080` | HTTP server listen address                      |
| `HTTP_READ_HEADER_TIMEOUT` | No       | `5s`    | Maximum time to read request headers            |
| `HTTP_READ_TIMEOUT`        | No       | `15s`   | Maximum time to read the complete request       |
| `HTTP_WRITE_TIMEOUT`       | No       | `30s`   | Maximum time before a response write times out  |
| `HTTP_IDLE_TIMEOUT`        | No       | `60s`   | Maximum idle keep-alive connection time         |
| `HTTP_SHUTDOWN_TIMEOUT`    | No       | `10s`   | Maximum time allowed for graceful HTTP shutdown |
| `LOG_LEVEL`                | No       | `INFO`  | Minimum logging severity                        |

Duration values use Go duration syntax, such as `5s`, `1m`, or `500ms`.

Supported logging levels are `DEBUG`, `INFO`, `WARN`, and `ERROR`. Values are case-insensitive and surrounding whitespace is ignored.

The application rejects an empty database URL, an empty HTTP address, non-positive timeout values, and unsupported logging levels.

### Configure the environment

The following examples use Windows Command Prompt.

Set the database connection string:

```cmd
set DATABASE_URL=postgres://order_app@localhost:5432/order_management?sslmode=disable
```

The `sslmode=disable` setting is intended for local development only. Production database connections should use an appropriate TLS configuration.

For application startup, the Go process needs access to the database
password. The PostgreSQL driver can obtain the database password from the
`PGPASSWORD` environment variable.

Windows Command Prompt does not provide a built-in hidden password prompt
for setting environment variables. Avoid entering actual passwords directly
in commands or committed scripts, as they may be exposed through terminal
history or other local process information.

For local development, provide the password through an appropriate
local secrets-management mechanism or a securely configured environment.

Optionally configure the HTTP address and logging level:

```cmd
set HTTP_ADDR=:8080
set LOG_LEVEL=INFO
```

These environment variables apply only to the current Command Prompt session. Set them again when opening a new session.

### Security considerations

- Never commit passwords, database credentials, API keys, tokens, or other sensitive information to source control.
- Do not store actual credentials in configuration examples or documentation.
- Use environment variables or an appropriate secrets-management solution to provide sensitive configuration.
- Ensure local environment files containing secrets are excluded from version control.

## Running the Application

Start the API server from the repository root:

```cmd
go run ./cmd/api
```

During startup, the application loads its configuration, initializes the PostgreSQL connection pool, verifies database connectivity, and starts the HTTP server.

The default HTTP listen address is `:8080`.

If database initialization fails, the application exits without starting the HTTP server.

### Verify the health endpoint

With the application running, open another terminal and execute:

```cmd
curl http://localhost:8080/health
```

Expected response:

```text
OK
```

### Stop the application

Press `Ctrl+C` in the terminal running the application.

The application handles the shutdown signal, attempts graceful HTTP server shutdown, and closes the database connection pool.

## Running Tests

Run all project tests:

```cmd
go test ./... -count=1
```

Run tests for a specific package:

```cmd
go test ./internal/config -count=1 -v
```

The database connectivity test uses `DATABASE_URL` and is skipped when the variable is not set. When configured, it attempts to connect to the specified PostgreSQL database.

Ensure the specified PostgreSQL instance is running and accessible before running the database connectivity test.

## Troubleshooting

### DATABASE_URL must not be empty

Ensure that `DATABASE_URL` is set in the current terminal session.

### Password authentication failed

Verify the application role's password and ensure that the database credentials are available to the application process.

### Connection refused

Verify that the PostgreSQL service is running and that the host and port in the connection string are correct.

### Address already in use

Another process may already be listening on the configured HTTP address. Stop that process or configure a different `HTTP_ADDR`.

## Related Documentation

- [Project Requirements](../requirements.md)
- [Architecture](../architecture.md)
- [Roadmap](../roadmap.md)
