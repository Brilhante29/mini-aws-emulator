# Benchmark Plan

## Claim

The pinned Kumo runtime preserves all 18 selected S3, SQS, and DynamoDB behaviors and completes the mixed workload without functional failures.

## Primary Metric

`conformance_rate_percent = passed_checks / total_checks * 100`

Required threshold: exactly 100%.

## Secondary Metrics

- `p95_operation_latency_ms`
- `operations_per_second`
- `measured_operations`
- `failed_operations`
- `startup_ms`
- `coverage_percent`
- `sdk_response_close_warnings`

Required functional thresholds:

- `failed_operations=0`
- `measured_operations=225` for the default run
- `coverage_percent>=75`

## Method

1. Build the Docker image; build gates run tests, vet, and coverage.
2. Start pinned Kumo and wait for `/health`.
3. Run 18 unmeasured conformance checks.
4. Create one bucket, queue, and table for the benchmark.
5. Run 25 iterations of nine operations: three per service.
6. Measure each operation wall-clock duration and total successful throughput.
7. Clean resources, write JSON, and fail the process on any mismatch.
8. Repeat once and commit both JSON results.

The conformance suite and setup warm SDK clients and Kumo before the measured loop. Setup and cleanup durations are excluded from p95 and throughput.

## Environment

- Docker Desktop 27.4.0
- Linux/amd64
- 16 CPUs
- 16.45 GB Docker memory
- Go 1.25.10
- AWS SDK Go v2 1.41.9
- Smithy Go 1.26.0
- Kumo 0.25.3 pinned by digest

## Interpretation

Conformance is the primary proof and must remain stable. Sub-3 ms p95 values are sensitive to local scheduler noise, so latency is reported, not generalized to AWS or production. The deterministic 83 close-warning diagnostics document an emulator/SDK compatibility gap and do not replace behavioral assertions.
