# #13 mini-aws-emulator

**Status:** scaffold

**Proves:** emulacao minima de APIs AWS.

**Benchmark target:** conformance_tests_passed.

**Stack:** go, sqlite, aws-sdk, docker.

## Next milestone

Implement the smallest Docker-runnable version and produce the first JSON benchmark under enchmarks/results/.

## Run

`ash
docker build -t mini-aws-emulator .
docker run --rm mini-aws-emulator
`

## Benchmark

`ash
docker run --rm mini-aws-emulator benchmark
`

| Metric | Value | Unit |
|---|---:|---|
| conformance_tests_passed | pending | pending |

## Architecture

Defined in sdd/spec.md before implementation.

## References

See REFERENCES.md.