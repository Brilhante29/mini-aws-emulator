#!/bin/sh
set -eu

provider="${CLOUD_PROVIDER:-kumo}"
if [ "$provider" = "aws" ]; then
  exec /app/conformance
fi
if [ "$provider" != "kumo" ]; then
  echo "CLOUD_PROVIDER must be kumo or aws" >&2
  exit 2
fi

export CLOUD_ENDPOINT="${CLOUD_ENDPOINT:-http://127.0.0.1:4566}"
kumo --host 127.0.0.1 --port 4566 >/tmp/kumo.log 2>&1 &
kumo_pid=$!
cleanup() {
  kill "$kumo_pid" 2>/dev/null || true
  wait "$kumo_pid" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

if /app/conformance; then
  exit 0
fi
status=$?
echo "Kumo log after conformance failure:" >&2
cat /tmp/kumo.log >&2
exit "$status"
