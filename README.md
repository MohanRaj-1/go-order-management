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

**Status:** Initial setup

Implementation has not started. Requirements, architecture, and project structure will be developed incrementally.

## Engineering Principles

- Correctness before optimization.
- Simplicity before unnecessary abstraction.
- Explicit error handling.
- Safe concurrency and data consistency.
- Automated testing for important behavior.
- Clear documentation and deliberate Git history.

## Learning Focus

This project is intended to develop practical backend engineering skills through implementation, testing, debugging, and architectural reasoning.

It is a learning project, not a claim of production readiness.
