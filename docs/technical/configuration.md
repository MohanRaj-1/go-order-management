# Configuration

## Overview

The `internal/config` package centralizes environment-variable loading, defaults, parsing, and validation so application components receive consistent, typed settings. During startup, `run()` in `cmd/api/main.go` calls `config.Load()` before initializing the logger, database connection pool, and HTTP server.

The returned `Config` supplies the database connection string, logging level, HTTP listen address, and request, connection, and shutdown timeouts. If configuration loading fails, the application returns an error before initializing these resources. Instructions for setting environment variables locally are provided in the [local development setup guide](../development/local-setup.md#environment-variables).

## Configuration Loading

`config.Load()` reads the process environment, using `os.Getenv()` for `DATABASE_URL` and `os.LookupEnv()` for the HTTP address, timeouts, and logging level. When optional variables are unset, it applies the default HTTP address, timeout durations, and logging level; `DATABASE_URL` has no default.

Timeout values are parsed with `time.ParseDuration()`, and the logging level is converted to a `slog.Level` after trimming whitespace and normalizing case. Explicitly set empty values are treated as provided values rather than triggering defaults. They are rejected during parsing or validation, depending on the configuration field. After parsing and validation succeed, `Load()` returns the populated `Config` value and a `nil` error.

## Environment Variables and Defaults

Defaults are applied individually in `Load()` and its helper functions; the package does not declare a separate defaults structure or a shared set of default constants. The initial `Config` literal sets `HTTPAddr` to `:8080` and reads `DatabaseURL` directly from the environment without a fallback. If `HTTP_ADDR` is present, its value replaces the initial address.

Each timeout field receives the result of a `durationFromEnv()` call, with its default duration supplied explicitly at the call site in `Load()`. The `LogLevel` field receives the result of `logLevelFromEnv()`, which returns `slog.LevelInfo` when `LOG_LEVEL` is unset. These assignments collect strings, parsed `time.Duration` values, and a `slog.Level` in a single `Config` value before validation.

## Validation

`Config.Validate()` requires non-empty `HTTPAddr` and `DatabaseURL` values, rejecting empty or whitespace-only strings. `HTTPAddr` has a default, while `DATABASE_URL` must be supplied through the environment. The whitespace check does not modify the stored values or verify address and connection-string syntax.

All five HTTP timeout fields must be greater than zero. A zero or negative duration returns an error identifying the corresponding environment variable.

Log levels are checked separately by `logLevelFromEnv()` during loading, before `Config.Validate()` runs. After trimming whitespace and normalizing case, only `DEBUG`, `INFO`, `WARN`, and `ERROR` are accepted; an unsupported value causes `Load()` to return an error. `Config.Validate()` does not independently check the `LogLevel` field.

Configuration loading returns on the first encountered error rather than collecting multiple failures. Within `Config.Validate()`, the HTTP address is checked first, followed by the database URL, then the read-header, read, write, idle, and shutdown timeouts in that order.

## Error Handling

`durationFromEnv()` wraps duration-parsing errors with the environment-variable name and supplied value, while `logLevelFromEnv()` creates an error identifying the unsupported value and accepted levels. `Load()` returns these helper errors directly with a zero-value `Config`; errors from `Config.Validate()` are instead wrapped with `validate configuration: %w`.

In `cmd/api/main.go`, `run()` wraps a configuration-loading error with `load configuration: %w` and returns before initializing application resources. The `%w` wrappers preserve the underlying errors for inspection through Go's error-unwrapping mechanisms. `main()` logs the returned error using `slog.Error()` and exits with status `1`.

## Testing

Tests in [`internal/config/config_test.go`](../../internal/config/config_test.go) exercise configuration loading and validation, grouped by the following behaviors:

- Default configuration: with optional variables unset and a database URL supplied, the tests check the default HTTP address, all five timeout durations, and the `INFO` logging level.
- Explicit values: the tests check that a custom read timeout is parsed and that a supplied database URL is preserved.
- Invalid or empty values: the tests expect errors for malformed read-timeout text, an explicitly empty database URL, and a whitespace-only HTTP address supplied directly to `Validate()`.
- Timeout validation: cases with a zero read timeout expect an error from both `Load()` and `Validate()`. These cases only assert that an error is returned; they do not verify that the timeout-specific validation error is reached. The direct validation case omits `DatabaseURL`, so it fails the database URL check before reaching timeout validation. Negative durations and non-positive values for the other timeout fields are not explicitly tested.
- Log-level parsing: table-driven subtests check all four supported levels, lowercase input, and surrounding whitespace with mixed case. A separate case expects an error for `TRACE`.

The tests use `t.Setenv()` to restore changed environment variables automatically after each test or subtest. The default-value test additionally saves and unsets optional variables, then restores their original presence and values through `t.Cleanup()`. Other loading tests override selected variables rather than clearing the entire configuration environment, so unrelated inherited values can still affect their results.

These are configuration-package tests: they do not call the application's `run()` function, open a database connection, or start the HTTP server, and they do not verify application startup behavior.

Run them from the repository root:

```cmd
go test ./internal/config -count=1 -v
```

## Current Limitations

The configuration package loads settings from the process environment with built-in defaults for optional values. It does not read configuration files, including `.env` files, query remote configuration sources, or retrieve secrets from a secrets-management service. Any external tooling used to supply configuration or secrets must make the required values available to the process before `config.Load()` runs. Configuration is loaded once during application startup; changes are not automatically reloaded while the application is running.
