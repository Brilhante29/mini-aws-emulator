#!/usr/bin/env python3
"""Validate the committed V2 benchmark contract and its source provenance."""

from __future__ import annotations

import hashlib
import json
import re
import subprocess
import sys
import uuid
from datetime import datetime
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
RESULT = ROOT / "benchmarks" / "publication" / "kumo-baseline-v2.json"
SCHEMA = ROOT / ".portfolio" / "contracts" / "benchmark-result-v2.schema.json"
FIXTURE_PATHS = (
    "internal/adapters/awssdk/adapter.go",
    "internal/benchmark/benchmark.go",
    "internal/cloud/ports.go",
    "internal/conformance/suite.go",
)
CONFIG_PATHS = (
    "Dockerfile",
    "compose.yaml",
    "go.mod",
    "go.sum",
    "internal/runtimeconfig/config.go",
    "tools/benchmark.ps1",
    "tools/benchmark-v2.ps1",
)


def fail(message: str) -> None:
    raise ValueError(message)


def resolve_ref(schema: dict[str, Any], reference: str) -> dict[str, Any]:
    if not reference.startswith("#/"):
        fail(f"unsupported schema reference: {reference}")
    value: Any = schema
    for part in reference[2:].split("/"):
        value = value[part.replace("~1", "/").replace("~0", "~")]
    return value


def matches_type(value: Any, expected: str) -> bool:
    if expected == "object":
        return isinstance(value, dict)
    if expected == "array":
        return isinstance(value, list)
    if expected == "string":
        return isinstance(value, str)
    if expected == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if expected == "number":
        return isinstance(value, (int, float)) and not isinstance(value, bool)
    if expected == "boolean":
        return isinstance(value, bool)
    return True


def validate_schema(value: Any, rule: dict[str, Any], schema: dict[str, Any], path: str = "$") -> None:
    if "$ref" in rule:
        validate_schema(value, resolve_ref(schema, rule["$ref"]), schema, path)
        return
    if "const" in rule and value != rule["const"]:
        fail(f"{path} must equal {rule['const']!r}")
    if "enum" in rule and value not in rule["enum"]:
        fail(f"{path} must be one of {rule['enum']!r}")
    expected = rule.get("type")
    if expected and not matches_type(value, expected):
        fail(f"{path} must be {expected}")
    if isinstance(value, dict):
        properties = rule.get("properties", {})
        for required in rule.get("required", []):
            if required not in value:
                fail(f"{path}.{required} is required")
        if rule.get("additionalProperties") is False:
            unexpected = sorted(set(value) - set(properties))
            if unexpected:
                fail(f"{path} has unexpected properties: {unexpected}")
        for name, child in value.items():
            if name in properties:
                validate_schema(child, properties[name], schema, f"{path}.{name}")
    elif isinstance(value, list):
        if len(value) < rule.get("minItems", 0):
            fail(f"{path} has too few items")
        if "items" in rule:
            for index, child in enumerate(value):
                validate_schema(child, rule["items"], schema, f"{path}[{index}]")
    elif isinstance(value, str):
        if len(value) < rule.get("minLength", 0):
            fail(f"{path} is too short")
        if "pattern" in rule and re.search(rule["pattern"], value) is None:
            fail(f"{path} does not match {rule['pattern']}")
        if rule.get("format") == "uuid":
            uuid.UUID(value)
        if rule.get("format") == "date-time":
            normalized = value.replace("Z", "+00:00")
            normalized = re.sub(r"(\.\d{6})\d+([+-]\d{2}:\d{2})$", r"\1\2", normalized)
            datetime.fromisoformat(normalized)
        if rule.get("format") == "uri" and "://" not in value:
            fail(f"{path} is not an absolute URI")
    elif isinstance(value, (int, float)) and not isinstance(value, bool):
        if "minimum" in rule and value < rule["minimum"]:
            fail(f"{path} is below its minimum")


def git(*args: str, text: bool = True) -> str | bytes:
    output = subprocess.check_output(["git", *args], cwd=ROOT, text=text)
    return output.strip() if text else output


def source_bytes(commit: str, relative_path: str) -> bytes:
    return git("show", f"{commit}:{relative_path}", text=False)


def combined_digest(commit: str, paths: tuple[str, ...]) -> str:
    lines = []
    for relative_path in sorted(paths):
        digest = hashlib.sha256(source_bytes(commit, relative_path)).hexdigest()
        lines.append(f"{relative_path}|{digest}")
    payload = ("\n".join(lines) + "\n").encode()
    return "sha256:" + hashlib.sha256(payload).hexdigest()


def metric_map(result: dict[str, Any]) -> dict[str, dict[str, Any]]:
    metrics = {metric["name"]: metric for metric in result["metrics"]}
    if len(metrics) != len(result["metrics"]):
        fail("metric names must be unique")
    return metrics


def validate_semantics(result: dict[str, Any]) -> None:
    if result["project"] != "mini-aws-emulator":
        fail("unexpected project in V2 evidence")
    if result["benchmark_id"] != "aws-sdk-kumo-conformance":
        fail("unexpected benchmark_id")

    repeat = result["execution"]["repeat"]
    if repeat < 3:
        fail("V2 publication requires at least three repetitions")
    metrics = metric_map(result)
    required_metrics = {
        "conformance_rate_percent",
        "p95_operation_latency_ms",
        "operations_per_second",
        "failed_operations",
        "core_coverage_percent",
        "sdk_response_close_warnings",
    }
    if not required_metrics.issubset(metrics):
        fail(f"missing metrics: {sorted(required_metrics - set(metrics))}")
    for name, metric in metrics.items():
        if len(metric["samples"]) != repeat:
            fail(f"metric {name} must retain one sample per repetition")
        runs = metric["summary"].get("runs")
        if not isinstance(runs, list) or len(runs) != repeat or not all(isinstance(run, dict) for run in runs):
            fail(f"metric {name} must retain {repeat} structured run summaries")
        if metric["summary"].get("provider") != "kumo":
            fail(f"metric {name} was not produced against Kumo")
        if metric["summary"].get("protocol_client") != "official AWS SDK for Go v2":
            fail(f"metric {name} was not produced through AWS SDK Go v2")
    if metrics["conformance_rate_percent"]["value"] != 100:
        fail("conformance must be exactly 100 percent")
    if metrics["failed_operations"]["value"] != 0:
        fail("failed_operations must be zero")
    if metrics["core_coverage_percent"]["value"] < 75:
        fail("core coverage must be at least 75 percent")

    source_commit = result["provenance"]["source_commit"]
    subprocess.check_call(["git", "merge-base", "--is-ancestor", source_commit, "HEAD"], cwd=ROOT)
    source_dockerfile = source_bytes(source_commit, "Dockerfile").decode()
    pinned = re.search(r"ghcr\.io/sivchari/kumo:([^@\s]+)@(sha256:[0-9a-f]{64})", source_dockerfile)
    if pinned is None:
        fail("source Dockerfile does not pin Kumo by version and digest")
    provider_image = f"ghcr.io/sivchari/kumo:{pinned.group(1)}@{pinned.group(2)}"
    if result["environment"].get("provider_image") != provider_image:
        fail("evidence provider image does not match the source Dockerfile")
    if result["environment"].get("cloud_provider_mode") != "kumo-local-first":
        fail("evidence is not marked kumo-local-first")

    source_go_mod = source_bytes(source_commit, "go.mod").decode()
    sdk = re.search(r"(?m)^\s*github\.com/aws/aws-sdk-go-v2 v([0-9.]+)$", source_go_mod)
    if sdk is None or f"AWS SDK Go v2 {sdk.group(1)}" not in result["environment"]["runtime"]:
        fail("evidence runtime does not match the pinned AWS SDK Go v2 dependency")
    if result["workload"]["fixture_digest"] != combined_digest(source_commit, FIXTURE_PATHS):
        fail("fixture digest does not match benchmark source")
    if result["workload"]["config_digest"] != combined_digest(source_commit, CONFIG_PATHS):
        fail("config digest does not match benchmark source")
    expected_lock = "sha256:" + hashlib.sha256(source_bytes(source_commit, "go.sum")).hexdigest()
    if result["provenance"]["dependency_lock_digest"] != expected_lock:
        fail("dependency lock digest does not match source go.sum")


def main() -> int:
    schema = json.loads(SCHEMA.read_text(encoding="utf-8"))
    result = json.loads(RESULT.read_text(encoding="utf-8"))
    validate_schema(result, schema, schema)
    validate_semantics(result)
    metrics = metric_map(result)
    print(
        "benchmark_v2_validation_passed "
        f"repeat={result['execution']['repeat']} "
        f"conformance={metrics['conformance_rate_percent']['value']} "
        f"p95_ms={metrics['p95_operation_latency_ms']['value']} "
        f"ops_s={metrics['operations_per_second']['value']}"
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (KeyError, OSError, subprocess.CalledProcessError, ValueError, json.JSONDecodeError) as error:
        print(f"benchmark_v2_validation_failed: {error}", file=sys.stderr)
        raise SystemExit(1)
