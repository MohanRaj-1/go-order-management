# Go Order Management — Project Roadmap

## Project Overview

**Repository:** `go-order-management`

**Language:** Go

**Database:** PostgreSQL

**Project type:** Backend engineering and distributed-systems learning project

**Target duration:** 8 weeks, adjusted according to learning progress

## Goal

Build a maintainable order management backend while developing practical skills in API design, database engineering, concurrency, resilience, testing, observability, and system design.

## Scope

- REST API design and request validation
- Clean, maintainable Go architecture
- PostgreSQL persistence, migrations, and transactions
- Inventory consistency under concurrent requests
- Contexts, deadlines, cancellation, and bounded retries
- Idempotency and failure handling
- Service-to-service communication where justified
- Automated testing, logging, metrics, and performance analysis
- Documentation and a versioned release

## Eight-Week Plan

### Week 1 — Requirements and Architecture

Define the domain, core entities, API contract, package boundaries, project structure, and development conventions.

### Week 2 — PostgreSQL and Persistence

Design the schema, introduce migrations, implement queries and persistence boundaries, and learn transaction fundamentals.

### Week 3 — Concurrency and Consistency

Implement safe inventory updates, investigate race conditions, and understand database locking and atomic operations.

### Week 4 — Resilience and Failure Handling

Apply contexts, deadlines, cancellation, bounded retries, idempotency, and appropriate error handling.

### Week 5 — Service Boundaries and Communication

Evaluate service boundaries and implement HTTP or gRPC communication when there is a meaningful architectural reason.

### Week 6 — Testing and Observability

Add unit and integration tests, exercise concurrent operations, use Go's race detector where applicable, and introduce structured logging and metrics.

### Week 7 — Performance and System Design

Benchmark critical operations, investigate bottlenecks, and document architectural trade-offs.

### Week 8 — Hardening and Release

Review the codebase, improve documentation, run validation, clean up the Git history, and prepare a versioned release.

## Non-Goals

- Starting with microservices.
- Adding infrastructure or dependencies without a clear requirement.
- Implementing real payment processing.
- Using real customer data.

Technology choices will be driven by the project's requirements and learning objectives. Any proposed addition must have a clear purpose and justify its complexity.

## Completion Criteria

- The core API works and is backed by PostgreSQL.
- Inventory and order operations preserve data consistency.
- Important failure paths are tested.
- Concurrency, transactions, and resilience decisions can be explained.
- The project includes useful tests, documentation, and setup instructions.
- The repository has a clean Git history and a documented release.

## Progress Tracking

This roadmap describes the intended plan. Actual progress, completed milestones, and deviations will be recorded as implementation proceeds.
