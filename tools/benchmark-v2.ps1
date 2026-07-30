param(
  [string]$Image = "mini-aws-emulator:benchmark",
  [int]$Iterations = 25,
  [ValidateRange(1, 10)]
  [int]$Repeat = 3,
  [ValidateSet("local", "github-actions", "other-ci")]
  [string]$Producer = "local",
  [string]$CiRunUrl = "",
  [string]$OutputPath = "benchmarks/publication/kumo-baseline-v2.json"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
  $sourceCommit = (& git rev-parse HEAD).Trim()
  $dirty = @(& git status --porcelain)
  if ($dirty.Count -gt 0) { throw "V2 benchmark requires a clean tree before execution." }
  if ($sourceCommit -notmatch "^[0-9a-f]{40}$") { throw "Could not resolve a full source commit." }
  if ($Producer -ne "local" -and [string]::IsNullOrWhiteSpace($CiRunUrl)) { throw "-CiRunUrl is required for a non-local producer." }

  function Get-RequiredProperty {
    param([object]$Object, [string]$Name)
    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property -or $null -eq $property.Value) { throw "Missing required benchmark property: $Name" }
    return $property.Value
  }

  function Get-CombinedDigest {
    param([string[]]$RelativePaths)
    $lines = foreach ($relative in ($RelativePaths | Sort-Object)) {
      $file = Join-Path $root $relative
      if (-not (Test-Path -LiteralPath $file -PathType Leaf)) { throw "Digest input missing: $relative" }
      $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $file).Hash.ToLowerInvariant()
      "${relative}|${hash}"
    }
    $bytes = [Text.Encoding]::UTF8.GetBytes(($lines -join "`n") + "`n")
    $digest = [Security.Cryptography.SHA256]::Create().ComputeHash($bytes)
    return "sha256:" + (([BitConverter]::ToString($digest) -replace "-", "").ToLowerInvariant())
  }

  function Get-Median {
    param([double[]]$Values)
    $ordered = @($Values | Sort-Object)
    $middle = [int][Math]::Floor($ordered.Count / 2)
    if (($ordered.Count % 2) -eq 1) { return [double]$ordered[$middle] }
    return ([double]$ordered[$middle - 1] + [double]$ordered[$middle]) / 2
  }

  $dockerText = Get-Content -Raw -LiteralPath (Join-Path $root "Dockerfile")
  $kumoMatch = [regex]::Match($dockerText, "ghcr\.io/sivchari/kumo:(?<version>[^@\s]+)@(?<digest>sha256:[0-9a-f]{64})")
  if (-not $kumoMatch.Success) { throw "Could not extract pinned Kumo version and digest." }
  $kumoVersion = $kumoMatch.Groups["version"].Value
  $kumoDigest = $kumoMatch.Groups["digest"].Value
  $fixtureDigest = Get-CombinedDigest @("internal/conformance/suite.go", "internal/benchmark/benchmark.go", "internal/cloud/ports.go")
  $configDigest = Get-CombinedDigest @("Dockerfile", "go.mod", "go.sum", "tools/benchmark.ps1")
  $lockHash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $root "go.sum")).Hash.ToLowerInvariant()
  $startedAt = [DateTime]::UtcNow
  $timer = [Diagnostics.Stopwatch]::StartNew()

  & docker build -t $Image $root
  if ($LASTEXITCODE -ne 0) { throw "Docker image build failed." }
  $resultNames = New-Object System.Collections.Generic.List[string]
  $rawResults = New-Object System.Collections.Generic.List[object]
  for ($run = 1; $run -le $Repeat; $run++) {
    $resultName = if ($run -eq 1) { "kumo-baseline.json" } elseif ($run -eq 2) { "kumo-confirmation.json" } else { "kumo-publication-run-$run.json" }
    $resultNames.Add($resultName)
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $root "tools/benchmark.ps1") -Image $Image -Iterations $Iterations -Repeat $run -OutputFile $resultName -SkipBuild
    if ($LASTEXITCODE -ne 0) { throw "Docker benchmark failed on repetition $run." }
    $rawResults.Add((Get-Content -Raw -LiteralPath (Join-Path $root "benchmarks/results/$resultName") | ConvertFrom-Json))
  }
  $timer.Stop()

  $conformanceSamples = @($rawResults | ForEach-Object { [double](Get-RequiredProperty $_ "value") })
  $p95Samples = @($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "p95_operation_latency_ms") })
  $throughputSamples = @($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "operations_per_second") })
  $failedSamples = @($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "failed_operations") })
  $coverageSamples = @($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "coverage_percent") })
  $warningSamples = @($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "sdk_response_close_warnings") })
  $p95 = [Math]::Round((Get-Median $p95Samples), 3)
  $conformance = [Math]::Round((($conformanceSamples | Measure-Object -Minimum).Minimum), 2)
  $throughput = [Math]::Round((($throughputSamples | Measure-Object -Average).Average), 3)
  $failed = [Math]::Round((($failedSamples | Measure-Object -Maximum).Maximum), 0)
  $coverage = [Math]::Round((($coverageSamples | Measure-Object -Minimum).Minimum), 2)
  $warnings = [Math]::Round((($warningSamples | Measure-Object -Maximum).Maximum), 0)
  $measured = [int](Get-RequiredProperty $rawResults[0].summary "measured_operations")
  $imageDigest = (& docker image inspect --format "{{.Id}}" $Image).Trim()
  if ($imageDigest -notmatch "^sha256:[0-9a-f]{64}$") { throw "Docker did not return a content digest." }

  $runSummaries = @($rawResults | ForEach-Object {
    [ordered]@{
      conformance_rate_percent = [double]$_.value
      p95_operation_latency_ms = [double]$_.summary.p95_operation_latency_ms
      operations_per_second = [double]$_.summary.operations_per_second
      measured_operations = [int]$_.summary.measured_operations
      failed_operations = [int]$_.summary.failed_operations
      coverage_percent = [double]$_.summary.coverage_percent
      sdk_response_close_warnings = [int]$_.summary.sdk_response_close_warnings
    }
  })
  $aggregateSummary = [ordered]@{
    aggregation = "minimum_conformance_mean_throughput_median_p95_max_failures"
    repetitions = $Repeat
    conformance_rate_percent = $conformance
    p95_operation_latency_ms = $p95
    operations_per_second = $throughput
    measured_operations_per_run = $measured
    failed_operations = $failed
    coverage_percent = $coverage
    sdk_response_close_warnings = $warnings
    runs = $runSummaries
  }
  $metrics = @(
    [ordered]@{ name = "conformance_rate_percent"; value = $conformance; unit = "percent"; direction = "target"; samples = @($conformanceSamples); failures = [int]($conformance -lt 100); summary = $aggregateSummary },
    [ordered]@{ name = "p95_operation_latency_ms"; value = $p95; unit = "milliseconds"; direction = "lower_is_better"; samples = @($p95Samples); failures = 0; summary = $aggregateSummary },
    [ordered]@{ name = "operations_per_second"; value = $throughput; unit = "operations_per_second"; direction = "higher_is_better"; samples = @($throughputSamples); failures = [int]($failed -gt 0); summary = $aggregateSummary },
    [ordered]@{ name = "failed_operations"; value = $failed; unit = "operations"; direction = "target"; samples = @($failedSamples); failures = [int]($failed -gt 0); summary = $aggregateSummary },
    [ordered]@{ name = "core_coverage_percent"; value = $coverage; unit = "percent"; direction = "target"; samples = @($coverageSamples); failures = [int]($coverage -lt 75); summary = $aggregateSummary },
    [ordered]@{ name = "sdk_response_close_warnings"; value = $warnings; unit = "diagnostics"; direction = "target"; samples = @($warningSamples); failures = 0; summary = $aggregateSummary }
  )
  $artifactDigest = Get-CombinedDigest @($resultNames | ForEach-Object { "benchmarks/results/$_" })
  $provenance = [ordered]@{
    source_commit = $sourceCommit
    clean_tree = $true
    image_ref = $Image
    image_digest = $imageDigest
    dependency_lock_digest = "sha256:$lockHash"
    producer = $Producer
    artifact_digest = $artifactDigest
  }
  if ($CiRunUrl) { $provenance.ci_run_url = $CiRunUrl }
  $output = Join-Path $root $OutputPath
  New-Item -ItemType Directory -Force -Path (Split-Path -Parent $output) | Out-Null
  $v2 = [ordered]@{
    schema_version = 2
    run_id = [guid]::NewGuid().ToString()
    project = "mini-aws-emulator"
    benchmark_id = "aws.compatibility.v1"
    workload = [ordered]@{
      version = "1.0.0"
      fixture_digest = $fixtureDigest
      config_digest = $configDigest
      warmup_iterations = 0
      measured_iterations = $measured
      concurrency = 1
    }
    metrics = $metrics
    execution = [ordered]@{
      command = "powershell -NoProfile -ExecutionPolicy Bypass -File tools/benchmark-v2.ps1 -Image $Image -Iterations $Iterations -Repeat $Repeat"
      started_at = $startedAt.ToString("o")
      duration_seconds = [Math]::Round($timer.Elapsed.TotalSeconds, 3)
      exit_code = 0
      repeat = $Repeat
    }
    environment = [ordered]@{
      runtime = "Go 1.25.10, AWS SDK Go v2 1.41.9, Smithy Go 1.26.0, Kumo $kumoVersion"
      architecture = "Linux amd64 Docker container"
      hardware_class = "local-docker"
      kumo_digest = $kumoDigest
      cloud_provider_mode = "kumo-local-first"
    }
    provenance = $provenance
    comparability_key = "aws-compatibility:1.0.0:kumo-$kumoVersion:aws-sdk-go-v2-1.41.9:go1.25.10:amd64"
  }
  [IO.File]::WriteAllText((Join-Path $root $OutputPath),(($v2 | ConvertTo-Json -Depth 12) + [Environment]::NewLine),(New-Object Text.UTF8Encoding($false)))
  $v2 | ConvertTo-Json -Depth 12
  Write-Host "v2_result=$(Join-Path $root $OutputPath)"
  Write-Host "source_commit=$sourceCommit"
  Write-Host "image_digest=$imageDigest"
  Write-Host "artifact_digest=$artifactDigest"
} finally {
  Pop-Location
}