# Architecture Decision

Status: accepted

## Context

The project must verify three external cloud capabilities locally, keep real AWS pluggable, and avoid coupling the behavioral suite to one SDK implementation or emulator.

## Decision

Use hexagonal architecture with three narrow output ports:

- `ObjectStorage`
- `Queue`
- `KeyValueStore`

Conformance and benchmark packages depend on these ports. The AWS SDK v2 adapter implements all three. Local mode configures static credentials, path-style S3, required-only checksums, and a Kumo endpoint. Real mode uses the same clients without endpoint override.

The CLI package is the composition root. It owns process lifecycle, configuration, safety guards, health wait, result assembly, and exit status.

## Dependency Rule

```text
cmd + AWS adapter -> conformance/benchmark -> cloud ports
```

Core packages must not import AWS SDK or Kumo packages. Kumo remains an executable dependency pinned in Docker.

## Alternatives Rejected

- Direct SDK script: no substitutable fake and mixed concerns.
- Layered controller/service/repository: wrong vocabulary for three external capabilities.
- Custom emulator: duplicates Kumo and weakens the measurable compatibility claim.
- Separate Kumo and AWS adapters: duplicates identical SDK calls and creates drift.
- Microservices: no independent deployment or ownership boundary.

## Consequences

The project has explicit substitution and fast unit tests, but ports add a small amount of interface code. Full protocol correctness still depends on integration tests against Kumo and, when explicitly selected, AWS.
