# Agent Handoff

Project: `13 - mini-aws-emulator`

## Current State

- Implementation: complete
- Baseline: `100%`, p95 `2.121 ms`, `754.371 ops/s`
- Confirmation: `100%`, p95 `1.688 ms`, `758.296 ops/s`
- Kit dependency: `cea7f9f`
- Default provider: pinned Kumo 0.25.3
- Real AWS: guarded and intentionally unverified in CI

## Ownership Boundaries

| Agent concern | Inputs | Outputs |
|---|---|---|
| architecture | problem forces, cloud matrix | ports and dependency rule |
| cloud local-first | Kumo release, AWS SDK | pinned runtime and provider switch |
| Go implementation | ports and spec | adapter, suites, tests |
| benchmark | behavior contract | two JSON evidence files |
| reuse review | project discoveries | kit commit `cea7f9f` |
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
