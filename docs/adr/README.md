# Architecture Decision Records

ADRs capture decisions that materially affect SeatFlow's boundaries, consistency, or operations. Copy [`000-template.md`](000-template.md), assign the next three-digit number, and use a short kebab-case filename such as `001-postgresql-seat-availability.md`.

Set the status to `Proposed`, `Accepted`, `Superseded`, or `Deprecated`. Do not rewrite an accepted decision to hide a change; add a new ADR and link the superseded record.

Initial decisions to record as implementation progresses include:

1. PostgreSQL as the source of truth for seat availability.
2. External REST and internal gRPC.
3. Transactional outbox instead of a database/Kafka dual write.
4. At-least-once delivery with idempotent consumers.
5. Redis limited to sessions and rate limiting.
6. One PostgreSQL instance with separate logical databases.
7. `READ COMMITTED` with explicit row locks.
8. Fail-open rate limiting and fail-closed refresh sessions.
