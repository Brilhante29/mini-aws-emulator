#!/bin/sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
IMAGE="${IMAGE:-mini-aws-emulator}"
ITERATIONS="${BENCHMARK_ITERATIONS:-25}"
WARMUP_ITERATIONS="${BENCHMARK_WARMUP_ITERATIONS:-5}"
REPEAT="${REPEAT:-1}"
OUTPUT_FILE="${OUTPUT_FILE:-kumo-baseline.json}"
RESULT="$ROOT/benchmarks/results/$OUTPUT_FILE"
TEMP_FILE="$(mktemp)"
trap 'rm -f "$TEMP_FILE"' EXIT INT TERM

mkdir -p "$ROOT/benchmarks/results"
docker build -t "$IMAGE" "$ROOT"
if ! docker run --rm \
  -e "BENCHMARK_ITERATIONS=$ITERATIONS" \
  -e "BENCHMARK_WARMUP_ITERATIONS=$WARMUP_ITERATIONS" \
  -e "REPEAT=$REPEAT" \
  "$IMAGE" >"$TEMP_FILE"; then
  cat "$TEMP_FILE" >&2
  exit 1
fi
mv "$TEMP_FILE" "$RESULT"
cat "$RESULT"
