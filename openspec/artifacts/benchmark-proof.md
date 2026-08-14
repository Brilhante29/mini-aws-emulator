# Benchmark Proof: mini-aws-emulator

## Primary Metric

- Metric: `conformance_rate_percent`
- Unit: `percent`
- Result: V2 aggregate conformance_rate_percent = 100 percent across three Kumo runs
- Secondary result: median p95 = 4.764 ms; mean throughput = 478.878 ops/s; failed operations = 0
- V1 result path: `benchmarks/results/kumo-baseline.json`
- V2 publication path: `benchmarks/publication/kumo-baseline-v2.json`

## Command

    powershell -NoProfile -ExecutionPolicy Bypass -File tools/benchmark-v2.ps1 -Repeat 3

## Evidence

| Artifact | Purpose |
|---|---|
| `benchmarks/results/kumo-baseline.json` | V1 baseline: 18/18, 1.715 ms p95, 730.145 ops/s |
| `benchmarks/results/kumo-confirmation.json` | V1 confirmation: 18/18, 1.605 ms p95, 850.726 ops/s |
| `benchmarks/publication/kumo-baseline-v2.json` | Three-run publication aggregate, samples, structured run summaries, and provenance |

V2 policy: minimum conformance, median p95, mean throughput, summed failures, and minimum coverage across three independent Docker runs. The artifact passed `benchmark-result-v2.schema.json` and project-specific provenance validation with zero errors.

The README/post number must come from the committed benchmark JSON, not from manual text.
