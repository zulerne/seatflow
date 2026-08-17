# SeatFlow

SeatFlow is a small event-seat reservation system built to demonstrate reliable concurrent booking in Go. Its central problem is simple to describe and difficult to implement correctly: when many users try to reserve the same seat, exactly one request may succeed.

> **Status:** repository bootstrap and local infrastructure are complete; protobuf contracts are the next milestone.

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

Requires Go 1.26, Task 3, Docker 20.10.4 or newer, and Docker Compose with `up --wait` support. The bootstrap has no third-party Go dependencies.

```bash
task build      # build all four binaries into bin/
task test       # run unit and process-level integration tests
task test-race  # run the suite with Go's race detector
task run        # start all four process skeletons; Ctrl-C stops them
```

Run `task help` to see which planned tasks become available in later issues. Each process validates configuration before starting. The Gateway requires `SEATFLOW_ENV` and `HTTP_ADDR`; the three internal services require `SEATFLOW_ENV` and `GRPC_ADDR`. `task run` supplies local defaults.

All processes emit JSON logs with `service` and `environment` fields. Their `main` functions own the root signal context, and the shared lifecycle waits for SIGINT or SIGTERM before exiting cleanly. Resource-specific shutdown is intentionally deferred until those resources exist.

## Local Infrastructure

Start PostgreSQL, Redis, and the single-node Kafka KRaft broker and wait for all health checks:

```bash
task infra-up
```

The default local endpoints are:

| Dependency | Host connection | Container-network connection |
| --- | --- | --- |
| PostgreSQL | `postgres://seatflow:seatflow@localhost:5432/seatflow?sslmode=disable` | `postgres://seatflow:seatflow@postgres:5432/seatflow?sslmode=disable` |
| Redis | `localhost:6379` | `redis:6379` |
| Kafka | `localhost:9092` | `kafka:19092` |

The defaults live in `deploy/.env.example` and are local-development credentials, not production secrets. Override a value in the shell or point Task at another env file, for example `task infra-up COMPOSE_ENV_FILE=deploy/local.env`.

All published ports bind to `127.0.0.1`. The Compose stack uses pinned `postgres:18.4-trixie`, `redis:8.2.8-bookworm`, and `apache/kafka:4.3.1` images. PostgreSQL's health check verifies that the configured database accepts connections, Redis must answer `PONG`, and Kafka must answer a metadata request. These checks detect an unavailable process or an unready endpoint; they do not prove that every query, command, or message will succeed.

Named volumes survive `task infra-down`: `postgres-data` contains the PostgreSQL 18 cluster, `redis-data` contains append-only persistence, and `kafka-data` contains the KRaft metadata log, topic partitions, and consumer offsets. A plain container restart therefore preserves PostgreSQL data. Removing the volumes is an explicit destructive action and is not part of `task infra-down`.

### Smoke checks

Verify Redis:

```bash
docker compose --file deploy/compose.yaml exec -T redis redis-cli ping
```

Verify PostgreSQL persistence with a disposable marker:

```bash
docker compose --file deploy/compose.yaml exec -T postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "CREATE TABLE infra_persistence_check (value text NOT NULL); INSERT INTO infra_persistence_check VALUES ('\''survived'\'');"'

docker compose --file deploy/compose.yaml restart postgres
docker compose --file deploy/compose.yaml up --detach --wait postgres

docker compose --file deploy/compose.yaml exec -T postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "TABLE infra_persistence_check; DROP TABLE infra_persistence_check;"'
```

Create a Kafka topic, produce one record, and consume it:

```bash
docker compose --file deploy/compose.yaml exec -T kafka \
  /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:19092 \
  --create --if-not-exists --topic seatflow-smoke --partitions 1 --replication-factor 1

printf 'seatflow-smoke\n' | docker compose --file deploy/compose.yaml exec -T kafka \
  /opt/kafka/bin/kafka-console-producer.sh --bootstrap-server kafka:19092 \
  --topic seatflow-smoke

docker compose --file deploy/compose.yaml exec -T kafka \
  /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server kafka:19092 \
  --topic seatflow-smoke --from-beginning --max-messages 1 --timeout-ms 10000
```

A broker accepts records. A topic is a named stream split into ordered partitions. A consumer group divides those partitions among its members and tracks offsets. A record key makes records with the same key choose the same partition, preserving their relative order within that partition.

Stop the containers without deleting data:

```bash
task infra-down
```

This issue has no application transaction boundary, goroutines, retry loop, or idempotency behavior: only Docker-managed processes and health probes were added. A realistic failure is a port collision or unavailable image; `task infra-up` then fails instead of reporting a healthy stack. Docker owns each container's process lifecycle, Compose waits for health on startup, and `task infra-down` requests and waits for container shutdown.
