# Architecture Overview

## Service Boundaries

| Process | Owns | Does not own |
| --- | --- | --- |
| API Gateway | REST/JSON, JWT verification, request IDs, rate limiting, HTTP-to-gRPC mapping | Domain data or booking SQL |
| Identity Service | Users, password hashes, access/refresh tokens, refresh rotation | Reservations or seats |
| Booking Service | Catalog, seats, reservation state machine, expiry worker, transactional outbox | Identity and notification data |
| Notification Service | Kafka consumption, event deduplication, stored notifications | Reservation state |

Four processes preserve explicit ownership and allow each workload to fail, scale, and restart independently. They remain small enough to run together during local development.

## Communication Model

Use synchronous calls only when the client needs an immediate answer: authentication, catalog reads, reservation commands, and notification reads. The API Gateway exposes REST and calls services through gRPC with deadlines.

Use Kafka after a committed booking state change. The Booking Service writes the reservation and its outbox event in one PostgreSQL transaction. A publisher later sends that event to Kafka. A crash can cause redelivery, so the Notification Service records processed event IDs and applies each event idempotently before committing its consumer offset.

## Data and Consistency Rules

- PostgreSQL is the source of truth for seat availability.
- Each service connects only to its own logical database; there are no cross-database foreign keys.
- Booking uses `READ COMMITTED` transactions and explicit `SELECT ... FOR UPDATE` locks in deterministic seat order.
- Redis stores refresh sessions and rate-limit state only. It cannot guarantee seat uniqueness because it is outside the booking transaction.
- Kafka delivery is at least once. Exactly-once delivery is not assumed or advertised.
- No Kafka, Redis, or other network call may occur while a booking database transaction holds row locks.

## Decision Records

Significant architectural choices are documented in [`docs/adr/`](adr/README.md). Create an ADR before changing a service boundary, consistency guarantee, persistence owner, or delivery semantic.
