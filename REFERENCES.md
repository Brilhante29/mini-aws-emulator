# References

## Runtime And SDK Sources

| Source | Use | License / note |
|---|---|---|
| [sivchari/kumo](https://github.com/sivchari/kumo) | Local AWS-compatible runtime | MIT; consumed as an unmodified OCI image |
| [Kumo v0.25.3](https://github.com/sivchari/kumo/releases/tag/v0.25.3) | Reviewed release | Release tag includes `v` |
| [Kumo GoReleaser config at v0.25.3](https://github.com/sivchari/kumo/blob/v0.25.3/.goreleaser.yml) | Verified image-tag and manifest publication rules | Container version is published without the leading `v` |
| [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) | S3, SQS, DynamoDB, credentials, and configuration clients | Apache-2.0 |
| [AWS SDK checksum configuration](https://docs.aws.amazon.com/sdkref/latest/guide/feature-dataintegrity.html) | Local S3 checksum policy | Local mode uses `when_required` |
| [smithy-go](https://github.com/aws/smithy-go) | SDK logging and transport middleware | Apache-2.0 |
| [Go modules reference](https://go.dev/ref/mod) | Reproducible Go dependency graph | Official Go documentation |
| [Dockerfile reference](https://docs.docker.com/reference/dockerfile/) | Multi-stage build and immutable runtime composition | Official Docker documentation |

## Process Sources

| Source | Use |
|---|---|
| [OpenSpec](https://openspec.dev/) | Spec artifact graph and change reasoning |
| [portfolio-reuse-kit](https://github.com/Brilhante29/portfolio-reuse-kit) | Decision brain, skills, project contract, validation, and SDD |
| [programadorLhama](https://github.com/programadorLhama) | Reference for compact repository organization and explicit boundaries |
| [Rocketseat Nest Clean example](https://github.com/rocketseat-education/05-nest-clean) | Reference for inward dependency direction and testing organization |

No source code was copied from the organization references. Kumo is executed as a versioned dependency; its source is not vendored or modified in this repository.
