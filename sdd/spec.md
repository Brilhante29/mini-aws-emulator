# Specification

Project: `13 - mini-aws-emulator`

## Intent

Provide a reproducible local-cloud compatibility harness that proves selected S3, SQS, and DynamoDB behavior through the same AWS SDK v2 adapter that can target real AWS.

## User-Visible Claim

A single Docker run produces numeric JSON showing scoped conformance, operation latency, throughput, startup time, coverage, failures, provider identity, and SDK diagnostics.

## In Scope

- Six S3 behavior checks.
- Six SQS behavior checks.
- Six DynamoDB behavior checks.
- Nine measured operations per iteration.
- Kumo local-first execution.
- Guarded real AWS configuration.
- Provider-independent ports and an in-memory fake.
- Two committed benchmark runs.

## Out Of Scope

- Implementing an AWS emulator.
- Full AWS conformance.
- IAM, quotas, multi-region, managed durability, and production retry semantics.
- A web API, UI, broker, Kubernetes deployment, or Terraform.
- Real AWS execution in CI.

## Acceptance Criteria

- [x] `docker run --rm mini-aws-emulator` exits zero on Kumo.
- [x] All 18 scoped checks pass.
- [x] All 225 measured operations succeed.
- [x] Primary metric is `conformance_rate_percent=100`.
- [x] Core coverage is at least 75%.
- [x] Kumo release and digest are present in result JSON.
- [x] Known SDK close warnings are numeric, while unrelated warnings remain visible.
- [x] Real AWS is refused without explicit opt-in and run ID.
- [x] README opens with the committed benchmark number.
