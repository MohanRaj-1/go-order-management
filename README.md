# Go Order Management

A Go backend engineering project focused on building a reliable order management system while exploring database consistency, concurrency, resilience, and distributed-systems concepts.

## Project Goals

- Design a maintainable backend architecture in Go.
- Build REST APIs for products, inventory, and orders.
- Use PostgreSQL for persistence and transactional integrity.
- Handle concurrent requests without overselling inventory.
- Explore timeouts, cancellation, retries, and idempotency.
- Practice automated testing, observability, and performance analysis.
- Document engineering decisions and trade-offs.

## Project Scope

The system will evolve incrementally, beginning with a modular monolith. Architecture and technology choices will be driven by concrete requirements rather than unnecessary complexity.

## Technology Stack

- **Language:** Go
- **Database:** PostgreSQL
- **API:** HTTP/REST
- **Testing:** Go testing tools and appropriate integration testing

Additional technologies will be introduced only when justified by the project's requirements.

## Development Roadmap

See [docs/roadmap.md](docs/roadmap.md) for the planned milestones and completion criteria.

## Project Status

**Status:** Core application foundation and initial domain functionality implemented; development in progress.

The project currently includes:

- HTTP server with a health endpoint and graceful shutdown.
- Environment-based configuration and validation, with optional local `.env` loading.
- Structured JSON logging.
- PostgreSQL connection initialization, connection pool configuration, and startup connectivity verification.
- Database schema and versioned migrations for customers, products, inventory, orders, and order items.
- Customer domain and PostgreSQL persistence, with HTTP endpoints for customer creation and retrieval.
- Product domain, validation, and PostgreSQL persistence.
- Initial inventory record creation as part of product creation.
- Transaction management to ensure product and initial inventory creation succeed or roll back together.
- Unit tests and database integration tests for important application behavior.

The current implementation follows a modular monolith architecture. Product and inventory creation share a database transaction to preserve data consistency.

Further development will focus on order-management workflows, inventory operations, API expansion, concurrency control, and distributed-systems concepts.

For local configuration, copy `.env.example` to `.env` and follow the [local development setup guide](docs/development/local-setup.md#configure-the-environment). `.env` is ignored by Git; production and CI can supply environment variables directly.
The project is under active development and is not yet production-ready.

## Engineering Principles

- Correctness before optimization.
- Simplicity before unnecessary abstraction.
- Explicit error handling.
- Safe concurrency and data consistency.
- Automated testing for important behavior.
- Clear documentation and deliberate Git history.

## Learning Focus

This project is intended to develop practical backend engineering skills through implementation, testing, debugging, and architectural reasoning.
