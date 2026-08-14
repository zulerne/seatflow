# SeatFlow

SeatFlow is a small event-seat reservation system built to demonstrate reliable concurrent booking in Go. Its central problem is simple to describe and difficult to implement correctly: when many users try to reserve the same seat, exactly one request may succeed.

> **Status:** project charter complete; application bootstrap has not started.

## Project Goals

SeatFlow is a learning-oriented microservice project focused on PostgreSQL transactions, explicit row locking, idempotency, asynchronous delivery, and graceful service operation. PostgreSQL is the source of truth for seat availability. Redis may support sessions and rate limiting, but it must never decide whether a seat is available.

The system uses synchronous REST/gRPC calls when the caller needs an immediate result. Booking events are delivered asynchronously through Kafka using a transactional outbox. Delivery is **at least once**, not exactly once; consumers must therefore be idempotent.

## MVP Scope

- Register, sign in, and rotate refresh tokens.
- Browse events, shows, and available seats.
- Hold one to four seats, then confirm or cancel the reservation.
- Expire unconfirmed reservations automatically.
- View notifications produced from booking events.
- Prevent duplicate reservations with idempotency keys.
- Run locally with Docker Compose and deploy through Helm.
- Exercise unit, integration, concurrency, race, and end-to-end tests.

## Non-goals

The first release excludes real payments, email/SMS delivery, social login, a full admin UI, event sourcing, distributed transactions, service mesh, multi-region deployment, database sharding, and claims of exactly-once delivery. A React frontend and payment service may be considered only after the core workflow is complete.

## Architecture

SeatFlow has four independently runnable services:

```mermaid
flowchart LR
    C[Client] -->|REST/JSON| G[API Gateway]
    G -->|gRPC| I[Identity Service]
    G -->|gRPC| B[Booking Service]
    G -->|gRPC| N[Notification Service]
    G --> R[(Redis)]
    I --> R
    I --> IDB[(Identity DB)]
    B --> BDB[(Booking DB)]
    BDB -->|transactional outbox| K[Kafka]
    K -->|at-least-once| N
    N --> NDB[(Notification DB)]
```

The Gateway owns the external HTTP boundary; Identity owns users and sessions; Booking owns seat state, reservations, expiry, and outbox publication; Notification consumes events and deduplicates them. The services share one PostgreSQL server during the MVP but use separate logical databases and credentials.

See [the architecture overview](docs/architecture.md) for boundaries and consistency rules, and [the ADR directory](docs/adr/README.md) for decision records.

## Development

Implementation begins with the repository bootstrap. The intended contributor interface will include `make fmt`, `make vet`, `make test`, `make test-race`, `make infra-up`, and `make run`; these commands do not exist yet.
