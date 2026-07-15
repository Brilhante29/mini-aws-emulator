# #13 mini-aws-emulator: conformance_rate_percent = 100 percent

One AWS SDK v2 adapter preserves 18 scoped S3, SQS, and DynamoDB behaviors when switched between pinned Kumo and real AWS configuration.

This repository belongs to the Backend Reliability and Architecture Platform program. Its job is narrow: prove the measurable claim through the selected component pack before adding unrelated infrastructure or features.

The benchmark is the proof. conformance_rate_percent = 100 percent.  The result is stored in `benchmarks/results/kumo-baseline.json` and can be reproduced from the Docker/local path.

The important architecture decision is hexagonal. Cloud services are material external boundaries, so ports isolate the behavioral suite from the AWS SDK while the composition root selects local or real configuration.

The default path stays local-first. The project uses go-backend, exposes cli, uses messaging mode `none`, and stores data with `dynamodb-compatible-key-value`. The dependency rule is explicit: CLI composition and AWS adapters depend inward on benchmark, conformance, and cloud ports; core packages import no AWS SDK or Kumo implementation.

The rejected work matters as much as the implemented work. Anything that does not improve the benchmark stays out of the first version.

Post angle: start with the number, show the architecture boundary, then explain which future adapter can be added without changing the core use cases.
