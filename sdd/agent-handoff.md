# Agent Handoff

Project: `13 - mini-aws-emulator`

## Current State

- Implementation: complete
- V1 baseline: `100%`, p95 `1.715 ms`, `730.145 ops/s`
- V1 confirmation: `100%`, p95 `1.605 ms`, `850.726 ops/s`
- V2 local publication evidence: `100%`, median p95 `4.764 ms`, mean `478.878 ops/s`, zero failures, three runs
- Benchmark source commit: `4cadbea634226180214c65ac282bc58886486076`
- Local gates: V2 schema and provenance validator passed; Docker build passed tests, vet, and 81.2% core coverage
- Exact-head GitHub Actions: pending because this worktree is intentionally not pushed in the current scope
- Kit dependency: `1ddbda4`
- Default provider: pinned Kumo 0.25.3
- Real AWS: guarded and intentionally unverified in CI

## Ownership Boundaries

| Agent concern | Inputs | Outputs |
|---|---|---|
| architecture | problem forces, cloud matrix | ports and dependency rule |
| cloud local-first | Kumo release, AWS SDK | pinned runtime and provider switch |
| Go implementation | ports and spec | adapter, suites, tests |
| benchmark | behavior contract | three per-run samples and summaries plus V2 publication artifact |
| reuse review | project discoveries | kit commit `1ddbda4`, with provider-provenance follow-up |
| publication | README, CI, benchmark | public evidence |

## Invariants

- Do not import AWS SDK outside `internal/adapters/awssdk` and composition needs.
- Do not use an unpinned Kumo image.
- Do not weaken the AWS real-mode guard.
- Do not increase conformance scope without adding named assertions and updating the claim.
- Do not hide new SDK warnings behind the known-warning counter.
- Keep the default path secret-free.

## Next Safe Extension

Add another Kumo-supported AWS capability only when a later portfolio project needs it. Add a new port, scoped conformance checks, measured operations, unsupported-behavior notes, and a fresh benchmark; do not turn this repository into a generic emulator.
