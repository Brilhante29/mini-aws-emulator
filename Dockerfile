# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.25.10
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod go test ./...
RUN --mount=type=cache,target=/go/pkg/mod go vet ./...
RUN --mount=type=cache,target=/go/pkg/mod \
    go test -covermode=atomic \
      -coverpkg=./internal/benchmark,./internal/conformance,./internal/report,./internal/runtimeconfig \
      -coverprofile=/tmp/coverage.out \
      ./internal/benchmark ./internal/conformance ./internal/report ./internal/runtimeconfig && \
    mkdir -p /out && \
    go tool cover -func=/tmp/coverage.out | awk '/^total:/ { gsub("%", "", $3); print $3 }' > /out/coverage-percent.txt && \
    awk '{ if ($1 + 0 < 75) exit 1 }' /out/coverage-percent.txt
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.kumoVersion=0.25.3 -X main.kumoDigest=sha256:7ea090ae0b6d1d34615e8b7bd04a2f1cd864ec640a6826a91e90f40e975e196b -X main.awsSDKVersion=1.41.9" \
      -o /out/conformance ./cmd/conformance

FROM alpine:3.23 AS runner
RUN addgroup -g 10001 -S runner && adduser -u 10001 -S runner -G runner
COPY --from=build /out/conformance /app/conformance
COPY --from=build /out/coverage-percent.txt /app/coverage-percent.txt
USER runner
ENTRYPOINT ["/app/conformance"]

FROM ghcr.io/sivchari/kumo:0.25.3@sha256:7ea090ae0b6d1d34615e8b7bd04a2f1cd864ec640a6826a91e90f40e975e196b AS demo
COPY --from=build /out/conformance /app/conformance
COPY --from=build /out/coverage-percent.txt /app/coverage-percent.txt
COPY --chmod=755 tools/demo.sh /app/demo.sh
USER kumo
ENTRYPOINT ["/app/demo.sh"]
