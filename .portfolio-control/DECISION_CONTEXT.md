# Decision Context: #13 mini-aws-emulator

## Problem

Prove a narrow AWS protocol compatibility contract locally, using Kumo as the
standard local-first runtime and allowing the same AWS SDK v2 adapter to target
real AWS only behind explicit safety flags.

## Architecture Decision

Use Go hexagonal architecture: cloud capability ports define behavior,
conformance and benchmark use cases depend on those ports, and one AWS SDK v2
adapter is configured for Kumo or AWS. The CLI is sufficient; there is no
control API, GraphQL layer, broker, or custom emulator to maintain.

## Reuse Decision

Consume manifest V2, benchmark V2, exact-head publication evidence, and the
local-first cloud matrix. Add a Go/Kumo producer that locks go.sum, records the
Kumo OCI digest and app image digest, and keeps conformance separate from
latency/throughput samples.

## Evidence Boundary

The V1 JSON is the raw conformance/benchmark output. The V2 JSON contains the
scoped fixture/config digests, three independent runs, provider/image/lock
digests, and explicit unsupported AWS behavior. Local Kumo numbers are not AWS
production performance claims.

## Principles

- SRP: ports, adapter, conformance, benchmark, reporting, and runtime config have separate reasons to change.
- OCP/DIP: provider selection is configuration around the same capability ports.
- LSP/ISP: the fake and AWS adapter satisfy narrow cloud contracts.
- KISS/YAGNI: no custom emulator, broker, microservices, or control API.
- LIsP: provider substitutions preserve the cloud capability abstractions.