# HTTP Server

## Overview

The application uses Go's standard `net/http` package to accept HTTP requests and dispatch them to registered handlers. Server initialization, route registration, and lifecycle management are implemented in [`cmd/api/main.go`](../../cmd/api/main.go).

The current router uses `http.ServeMux` and exposes a health endpoint at `GET /health`. The application manages server startup, failure handling, graceful shutdown, and database resource cleanup.

## Server Configuration

The `run()` function creates an `http.Server` after loading and validating application configuration and establishing database connectivity. Its listen address comes from `HTTP_ADDR`, and its request handler is the router returned by `newRouter()`.

The server receives the following settings from the configuration package:

| Server field | Configuration field | Purpose |
| --- | --- | --- |
| `Addr` | `HTTPAddr` | Address on which the server listens |
| `Handler` | Router returned by `newRouter()` | Dispatches requests to registered handlers |
| `ReadHeaderTimeout` | `HTTPReadHeaderTimeout` | Limits the time allowed to read request headers |
| `ReadTimeout` | `HTTPReadTimeout` | Limits the time allowed to read the entire request, including its body |
| `WriteTimeout` | `HTTPWriteTimeout` | Limits the time allowed to write the response |
| `IdleTimeout` | `HTTPIdleTimeout` | Limits the wait for another request on a keep-alive connection |

`HTTP_SHUTDOWN_TIMEOUT` sets the deadline for graceful shutdown separately from the request and connection timeouts above. Environment variables, defaults, and local configuration instructions are documented in the [local development setup guide](../development/local-setup.md#environment-variables).

## Routing

`newRouter()` creates an `http.ServeMux` and registers the health handler using a method-qualified pattern:

```go
mux.HandleFunc("GET /health", healthHandler)
```

The router is assigned to `http.Server.Handler`. Requests for unregistered paths receive the standard `404 Not Found` response. Go's method-aware routing also matches `HEAD` requests to the `GET /health` pattern; other methods for that path receive `405 Method Not Allowed`.

## Server Lifecycle

### Startup

`main()` calls `run()`, which loads an optional local `.env` file before parsing configuration and initializing the application logger. Existing environment variables take precedence over file values; local configuration details are documented in the [setup guide](../development/local-setup.md#configure-the-environment). Database initialization uses a separate five-second context, and a failure returns before the HTTP server starts.

After creating the router and server, the application calls `ListenAndServe()` in a goroutine. The goroutine logs the configured address before attempting to listen and sends the returned error to the buffered `serverErrors` channel. The startup log records the attempt to start the server; it does not confirm that the listen operation succeeded.

The main execution path waits for either a server result or a termination signal.

### Runtime Error Handling

The application receives the result of `ListenAndServe()` through `serverErrors`. Errors other than `http.ErrServerClosed` are logged and retained for return from `run()`. This includes startup failures such as an address already being in use.

`http.ErrServerClosed` is treated as an expected result of closing the server. Both a server result and a termination signal lead to the shutdown sequence.

If `run()` returns an error, `main()` logs the application failure and exits with status `1`.

### Graceful Shutdown

`signal.NotifyContext()` creates `signalCtx`, which is canceled when the application receives `os.Interrupt` or `SIGTERM`. When that context is canceled, the application logs the shutdown signal and begins shutdown.

The shutdown context is created from `context.Background()` with the configured `HTTPShutdownTimeout`. Using a fresh context gives shutdown its own deadline even though the signal context has already been canceled.

The application calls `server.Shutdown(shutdownCtx)` to stop accepting connections, close idle connections, and allow active requests to finish within the deadline. If shutdown returns an error, including a deadline expiration, the application logs the error and calls `server.Close()` to force connections closed. Forced closure closes active connections, so clients may experience interrupted requests. Handlers that do not observe request cancellation may continue running.

Before returning, `run()` receives the `ListenAndServe()` result if it has not already done so and logs that the HTTP server has stopped. Server and shutdown errors are preserved together using `errors.Join()` when both occur.

The deferred database cleanup runs as `run()` returns. A database close failure is logged and included in the returned error.

## Health Endpoint

`healthHandler()` returns status `200 OK` and the response body `OK` followed by a newline.

```cmd
curl http://localhost:8080/health
```

Expected body:

```text
OK
```

The endpoint confirms that the HTTP handler can respond. It does not query the database or check other dependencies on each request. Database connectivity is checked during application startup.

## Testing

The HTTP tests are implemented in [`cmd/api/main_test.go`](../../cmd/api/main_test.go) using `net/http/httptest`.

| Test | Verified behavior |
| --- | --- |
| `TestHealthHandler` | A direct call to the health handler returns status `200` and the exact body `OK\n` |
| `TestRouterNotFound` | A request to `/missing` through the router returns status `404` |

Run the HTTP package tests from the repository root:

```cmd
go test ./cmd/api -count=1 -v
```

These tests exercise the handler and router in memory. They do not start a network listener or initialize the database.

## Current Limitations

- The health endpoint is the only registered application endpoint and does not provide a dependency readiness check.
- The server uses plain HTTP through `ListenAndServe()`; TLS is not configured in the application.
- Authentication, request validation, and application middleware are not implemented.
- Existing HTTP tests do not cover successful health routing through the mux, method handling, network startup failures, timeout enforcement, signal handling, or graceful and forced shutdown.
