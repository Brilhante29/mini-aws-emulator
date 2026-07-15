# Intent: mini-aws-emulator

## Measurable Claim

One AWS SDK v2 adapter preserves 18 scoped S3, SQS, and DynamoDB behaviors when switched between pinned Kumo and real AWS configuration.

## Problem

Adds a local-cloud compatibility boundary that later gateway, outbox, event, and infrastructure projects can reuse without requiring an AWS account.

## In Scope

- Use the selected component pack: `backend-reliability-platform`.
- Keep the project under the Backend Reliability and Architecture Platform program.
- Preserve the benchmark contract: `conformance_rate_percent` in `benchmarks/results/kumo-baseline.json`.
- Keep the default path local-first and reproducible.

## Out Of Scope

- Paid credentials for the default demo.
- External infrastructure that is not required by the benchmark.
- Replacing local portfolio skills with external components silently.

## Default Demo Path

- Status: benchmarked
- Runtime: Static Go 1.25.10 binary inside Kumo 0.25.3 Alpine image
- Benchmark command: `powershell -NoProfile -ExecutionPolicy Bypass -File tools/benchmark.ps1`

## Public Proof

- Benchmark: conformance_rate_percent = 100 percent
- Result path: `benchmarks/results/kumo-baseline.json`
