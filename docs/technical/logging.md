# Logging

## Overview

The application uses structured logging to record lifecycle events and failures with contextual fields that can be processed without extracting values from message text. Logging is provided by Go's standard `log/slog` package, with logger construction implemented in [`internal/logger/logger.go`](../../internal/logger/logger.go).

`logger.New()` accepts an output destination and minimum logging level and returns a `*slog.Logger` configured to write JSON records. During startup, `run()` in `cmd/api/main.go` calls this constructor with `os.Stdout` and `cfg.LogLevel`, then uses the returned logger for application lifecycle messages.

## Logger Initialization

`logger.New()` creates a handler with `slog.NewJSONHandler()`, passing the supplied output writer and a `slog.HandlerOptions` value. The options set `Level` to the supplied minimum level, so the handler filters out records below that threshold.

Accepting an `io.Writer` keeps the constructor independent of a specific output destination: the application can use standard output, while tests can capture records in a `bytes.Buffer`. Other destinations can be supplied through the same interface without changing the constructor. Finally, `slog.New(handler)` connects the handler to the returned logger, which uses it to process and write log records.

## Log Format and Structured Fields

The JSON handler writes each record as a single JSON object on its own line, with standard fields for `time`, `level`, and `msg`. The timestamp is generated when the logging call creates the record, and the level is represented as text such as `INFO` or `ERROR`.

Additional attributes are passed as key-value pairs after the message, as in `appLogger.Info("Starting HTTP server", "address", server.Addr)`. Strings, numbers, and booleans retain their corresponding JSON representations, while error attributes used by the application are serialized as their error-message strings rather than stack traces. The constructor configures only the minimum level; it does not customize timestamp, level, or attribute formatting, and it does not enable source-location fields.

For example, the server startup call produces a record like the following, shown with an illustrative timestamp:

```json
{"time":"2026-11-03T08:00:00Z","level":"INFO","msg":"Starting HTTP server","address":":8080"}
```

## Log Levels

The application accepts `DEBUG`, `INFO`, `WARN`, and `ERROR` through `LOG_LEVEL`, with `INFO` as the default when the variable is unset. Values are case-insensitive and may include surrounding whitespace.

The configured level sets the JSON handler's minimum threshold: records at that level or higher are written, while lower-level records are filtered out. For example, `INFO` includes informational messages, warnings, and errors but excludes debug messages. An unsupported value causes configuration loading to return an error and prevents application initialization; details are documented in [Configuration](configuration.md#validation).

## Application Logging

`run()` in [`cmd/api/main.go`](../../cmd/api/main.go) uses the configured application logger for the following lifecycle events. Informational events use `Info()`, and failures use `Error()`, subject to the configured minimum level.

- Startup: after database initialization succeeds, the application logs `Database connection established`. Local environment-file loading, configuration-loading, and database-initialization failures are returned to `main()` rather than logged at those call sites.
- Runtime: immediately before calling `ListenAndServe()`, the server goroutine logs `Starting HTTP server` with an `address` attribute. This records the startup attempt, not confirmation that the server is listening. A returned server error other than `http.ErrServerClosed` is logged as `HTTP server failed` with an `error` attribute.
- Shutdown: a termination signal produces `Shutdown signal received`. Errors from graceful shutdown and forced closure produce `Graceful shutdown failed` and `Force closing server failed`, respectively, each with an `error` attribute. After shutdown processing and receipt of the server result, the application logs `HTTP server stopped`. Deferred database cleanup logs `Failed to close database connection` with an `error` attribute if closing the pool fails.
- Application failure: when `run()` returns an error, `main()` logs `Application failed` with an `error` attribute and exits with status `1`. This call uses the default `slog` logger; the application does not replace it with `appLogger`, so it does not inherit the application's JSON handler or configured minimum level.

## Testing

Tests in [`internal/logger/logger_test.go`](../../internal/logger/logger_test.go) pass a `bytes.Buffer` to `New()` to capture output in memory without writing to standard output or a file.

The JSON-output test writes an informational record and decodes it with `json.Unmarshal()`, verifying that the output is valid JSON. It checks the exact `msg` and `level` values and confirms that the `order_id` and `operation` attributes retain their supplied string values. The order-creation message is a test example, not an implemented application event.

The filtering test configures an `INFO` minimum level, then emits debug, informational, warning, and error messages. It checks the captured text to confirm that the debug message is absent and the other three messages are present; it does not decode each record or check their severity fields individually.

These tests cover logger construction and output behavior in isolation. They do not verify timestamps, error-attribute serialization, thresholds other than `INFO`, writer failures, or application lifecycle logging through `run()` and `main()`.

Run the logger tests from the repository root:

```cmd
go test ./internal/logger -count=1 -v
```

## Current Limitations

The application writes its configured JSON logs to standard output. Although the logger accepts any `io.Writer`, the application does not configure file output, rotation, retention, or log shipping; collecting and retaining standard-output logs requires external tooling.

Logging currently covers application lifecycle events rather than individual HTTP requests. There is no request-logging middleware or automatic request correlation ID, and the logger does not add source locations or stack traces. The final application-failure message uses the default `slog` logger separately, so the application does not produce a uniform JSON format for every log message.
