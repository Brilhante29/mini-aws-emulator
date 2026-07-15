# Technical Decision Record

## Runtime

- Go 1.25.10
- Static Linux binary
- Kumo 0.25.3 as final Docker base
- Kumo image pinned by tag and OCI manifest digest

## AWS Client

Use AWS SDK for Go v2 with one configured adapter.

Local mode:

- static `test/test` credentials
- `BaseEndpoint` set to Kumo
- S3 path-style addressing
- S3 request and response checksums set to `when_required`

Real mode:

- default endpoint resolution
- default credential chain
- DynamoDB existence waiter
- explicit `ALLOW_REAL_AWS=true`
- explicit globally unique `RUN_ID`

## Messaging

SQS is a tested cloud capability, not an application messaging architecture. Kafka, RabbitMQ, NATS, and an outbox are rejected because there is no delivery workflow, consumer topology, or broker benchmark in this repository.

## API Style

Use a CLI because execution is finite and produces one result document. REST, GraphQL, gRPC, WebSocket, and SSE would add a server lifecycle without improving conformance.

## Diagnostics

Smithy emits one known close warning per Kumo S3 response. A custom logger counts only the exact warning as `sdk_response_close_warnings`; all other SDK messages are delegated to the standard logger. This avoids noisy CI while preserving evidence.

## Supply Chain

The Kumo release uses tag `v0.25.3`, while its GoReleaser configuration publishes image tag `0.25.3`. The committed image reference includes both `0.25.3` and the reviewed manifest digest. Project validation rejects the mutable Kumo image reference.

## Rejected Libraries And Services

- LocalStack: not required because Kumo is the portfolio standard and supports the selected services.
- An HTTP framework: no server is needed.
- A benchmark framework: direct operation timing keeps the workload and metric formula visible.
- A database: Kumo owns local state; the runner persists no application data.
