# Architecture Record: mini-aws-emulator

## Decision

- Architecture: `hexagonal`
- Stack profile: `go-backend`
- API style: `cli`
- Messaging: `none`
- Database/runtime: `dynamodb-compatible-key-value` / `Static Go 1.25.10 binary inside Kumo 0.25.3 Alpine image`

## Reason

Cloud services are material external boundaries, so ports isolate the behavioral suite from the AWS SDK while the composition root selects local or real configuration.

## Dependency Direction

CLI composition and AWS adapters depend inward on benchmark, conformance, and cloud ports; core packages import no AWS SDK or Kumo implementation.

## Boundaries

- cloud capability ports for objects, queues, and key-value data
- scoped conformance use cases
- mixed-operation benchmark harness
- AWS SDK v2 output adapter
- runtime configuration and real-cloud safety guard
- JSON evidence and Docker composition

## Library Policy

Use official AWS SDK v2 clients and Smithy logging, pin Kumo by release plus digest, keep ports narrow, and report intentionally handled SDK warnings numerically.

## Principle Check

- SRP: keep benchmark, API, use cases, and adapters separate.
- OCP: new providers must be adapters, not domain rewrites.
- LSP: replacement providers must preserve observable behavior.
- ISP: ports stay narrow.
- DIP: application depends on behavior, not infrastructure.
- KISS/YAGNI: leave out anything that does not improve the benchmark.
