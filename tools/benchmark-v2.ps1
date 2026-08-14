param(
  [string]$Image = "mini-aws-emulator:benchmark",
  [ValidateRange(1, 100)]
  [int]$WarmupIterations = 5,
  [ValidateRange(1, 1000)]
  [int]$Iterations = 25,
  [ValidateRange(3, 10)]
  [int]$Repeat = 3,
  [ValidateSet("local", "github-actions", "other-ci")]
  [string]$Producer = "local",
  [string]$CiRunUrl = "",
  # Must match publication_result_path in project.yaml: writing the V2 contract
  # under benchmarks/results left the declared publication path holding stale
  # evidence from an older commit.
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

  function Get-SHA256Text {
    param([string]$Value)
    $bytes = [Text.Encoding]::UTF8.GetBytes($Value)
    $digest = [Security.Cryptography.SHA256]::Create().ComputeHash($bytes)
    return "sha256:" + (([BitConverter]::ToString($digest) -replace "-", "").ToLowerInvariant())
  }

  function Get-CombinedDigest {
    param([string[]]$RelativePaths)
    $orderedPaths = [string[]]@($RelativePaths)
    [Array]::Sort($orderedPaths, [StringComparer]::Ordinal)
    $lines = foreach ($relative in $orderedPaths) {
      $hash = Get-GitBlobSHA256 $relative
      "${relative}|${hash}"
    }
    return Get-SHA256Text (($lines -join "`n") + "`n")
  }

  function Get-GitBlobSHA256 {
    param([string]$RelativePath)
    $startInfo = [Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = "git"
    $startInfo.Arguments = "cat-file blob `"${sourceCommit}:${RelativePath}`""
    $startInfo.WorkingDirectory = $root
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    if (-not $process.Start()) { throw "Could not read Git blob: $RelativePath" }
    $stream = [IO.MemoryStream]::new()
    $process.StandardOutput.BaseStream.CopyTo($stream)
    $stderr = $process.StandardError.ReadToEnd()
    $process.WaitForExit()
    if ($process.ExitCode -ne 0) { throw "Could not read Git blob ${RelativePath}: $stderr" }
    $digest = [Security.Cryptography.SHA256]::Create().ComputeHash($stream.ToArray())
    return (([BitConverter]::ToString($digest) -replace "-", "").ToLowerInvariant())
  }

  function Get-Median {
    param([double[]]$Values)
    $ordered = @($Values | Sort-Object)
    $middle = [int][Math]::Floor($ordered.Count / 2)
    if (($ordered.Count % 2) -eq 1) { return [double]$ordered[$middle] }
    return ([double]$ordered[$middle - 1] + [double]$ordered[$middle]) / 2
  }

  function Get-Sum {
    param([double[]]$Values)
    return [int](($Values | Measure-Object -Sum).Sum)
  }

  function New-Metric {
    param(
      [string]$Name,
      [double]$Value,
      [string]$Unit,
      [string]$Direction,
      [double[]]$Samples,
      [int]$Failures,
      [object]$Summary
    )
    return [ordered]@{
      name = $Name
      value = $Value
      unit = $Unit
      direction = $Direction
      samples = @($Samples)
      failures = $Failures
      summary = $Summary
    }
  }

  $dockerText = Get-Content -Raw -LiteralPath (Join-Path $root "Dockerfile")
  $kumoMatch = [regex]::Match($dockerText, "ghcr\.io/sivchari/kumo:(?<version>[^@\s]+)@(?<digest>sha256:[0-9a-f]{64})")
  if (-not $kumoMatch.Success) { throw "Could not extract pinned Kumo version and digest." }
  $kumoVersion = $kumoMatch.Groups["version"].Value
  $kumoDigest = $kumoMatch.Groups["digest"].Value

  $goModText = Get-Content -Raw -LiteralPath (Join-Path $root "go.mod")
  $goMatch = [regex]::Match($goModText, "(?m)^toolchain go(?<version>[0-9.]+)$")
  $sdkMatch = [regex]::Match($goModText, "(?m)^\s*github\.com/aws/aws-sdk-go-v2 v(?<version>[0-9.]+)$")
  $smithyMatch = [regex]::Match($goModText, "(?m)^\s*github\.com/aws/smithy-go v(?<version>[0-9.]+)")
  if (-not $goMatch.Success -or -not $sdkMatch.Success -or -not $smithyMatch.Success) { throw "Could not extract pinned Go/AWS SDK/Smithy versions." }
  $goVersion = $goMatch.Groups["version"].Value
  $sdkVersion = $sdkMatch.Groups["version"].Value
  $smithyVersion = $smithyMatch.Groups["version"].Value

  $fixtureDigest = Get-CombinedDigest @(
    "internal/adapters/awssdk/adapter.go",
    "internal/benchmark/benchmark.go",
    "internal/cloud/ports.go",
    "internal/conformance/suite.go"
  )
  $configDigest = Get-CombinedDigest @(
    "Dockerfile",
    "compose.yaml",
    "go.mod",
    "go.sum",
    "internal/runtimeconfig/config.go",
    "tools/benchmark.ps1",
    "tools/benchmark-v2.ps1"
  )
  $lockHash = Get-GitBlobSHA256 "go.sum"
  $startedAt = [DateTime]::UtcNow
  $timer = [Diagnostics.Stopwatch]::StartNew()

  & docker build --build-arg "VERSION=$sourceCommit" -t $Image $root
  if ($LASTEXITCODE -ne 0) { throw "Docker image build failed." }

  $rawResults = New-Object System.Collections.Generic.List[object]
  for ($run = 1; $run -le $Repeat; $run++) {
    $output = & docker run --rm `
      -e "BENCHMARK_WARMUP_ITERATIONS=$WarmupIterations" `
      -e "BENCHMARK_ITERATIONS=$Iterations" `
      -e "REPEAT=$run" `
      -e "RUN_ID=r$run" `
      $Image
    if ($LASTEXITCODE -ne 0) { throw "Docker benchmark failed on repetition $run." }
    try {
      $raw = (($output -join "`n") | ConvertFrom-Json)
    } catch {
      throw "Docker benchmark repetition $run returned invalid JSON: $($_.Exception.Message)"
    }
    if ($raw.environment.provider -ne "kumo") { throw "Repetition $run did not run against Kumo." }
    if ($raw.environment.provider_digest -ne $kumoDigest) { throw "Repetition $run used an unexpected Kumo digest." }
    if (@($raw.services) -join "," -ne "s3,sqs,dynamodb") { throw "Repetition $run did not exercise the scoped services." }
    $rawResults.Add($raw)
  }
  $timer.Stop()

  $conformanceSamples = [double[]]@($rawResults | ForEach-Object { [double](Get-RequiredProperty $_ "value") })
  $p95Samples = [double[]]@($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "p95_operation_latency_ms") })
  $throughputSamples = [double[]]@($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "operations_per_second") })
  $failedSamples = [double[]]@($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "failed_operations") })
  $coverageSamples = [double[]]@($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "coverage_percent") })
  $warningSamples = [double[]]@($rawResults | ForEach-Object { [double](Get-RequiredProperty $_.summary "sdk_response_close_warnings") })
  $conformanceFailures = [int](($rawResults | ForEach-Object { [int]$_.summary.total_checks - [int]$_.summary.passed_checks } | Measure-Object -Sum).Sum)
  $operationFailures = Get-Sum $failedSamples
  $conformance = [Math]::Round((($conformanceSamples | Measure-Object -Minimum).Minimum), 3)
  $p95 = [Math]::Round((Get-Median $p95Samples), 3)
  $throughput = [Math]::Round((($throughputSamples | Measure-Object -Average).Average), 3)
  $coverage = [Math]::Round((($coverageSamples | Measure-Object -Minimum).Minimum), 3)
  $warnings = [int](($warningSamples | Measure-Object -Maximum).Maximum)
  $measuredOperations = $Iterations * 9
  $warmupOperations = $WarmupIterations * 9

  $runSummaries = @($rawResults | ForEach-Object {
    [ordered]@{
      repeat = [int]$_.repeat
      conformance_rate_percent = [double]$_.value
      p95_operation_latency_ms = [double]$_.summary.p95_operation_latency_ms
      operations_per_second = [double]$_.summary.operations_per_second
      measured_operations = [int]$_.summary.measured_operations
      warmup_operations = [int]$_.summary.warmup_operations
      failed_operations = [int]$_.summary.failed_operations
      coverage_percent = [double]$_.summary.coverage_percent
      sdk_response_close_warnings = [int]$_.summary.sdk_response_close_warnings
    }
  })
  $aggregateSummary = [ordered]@{
    aggregation = "minimum_conformance_mean_throughput_median_latency_sum_failures"
    repetitions = $Repeat
    provider = "kumo"
    protocol_client = "official AWS SDK for Go v2"
    services = @("s3", "sqs", "dynamodb")
    scoped_checks_per_run = 18
    operations_per_iteration = 9
    warmup_iterations_per_run = $WarmupIterations
    measured_iterations_per_run = $Iterations
    runs = $runSummaries
  }
  $metrics = @(
    (New-Metric "conformance_rate_percent" $conformance "percent" "target" $conformanceSamples $conformanceFailures $aggregateSummary),
    (New-Metric "p95_operation_latency_ms" $p95 "milliseconds" "lower_is_better" $p95Samples $operationFailures $aggregateSummary),
    (New-Metric "operations_per_second" $throughput "operations_per_second" "higher_is_better" $throughputSamples $operationFailures $aggregateSummary),
    (New-Metric "failed_operations" ([double]$operationFailures) "operations" "target" $failedSamples $operationFailures $aggregateSummary),
    (New-Metric "core_coverage_percent" $coverage "percent" "target" $coverageSamples ([int]($coverage -lt 75)) $aggregateSummary),
    (New-Metric "sdk_response_close_warnings" ([double]$warnings) "diagnostics" "target" $warningSamples 0 $aggregateSummary)
  )

  $imageDigest = (& docker image inspect --format "{{.Id}}" $Image).Trim()
  $imageArchitecture = (& docker image inspect --format "{{.Architecture}}" $Image).Trim()
  if ($imageDigest -notmatch "^sha256:[0-9a-f]{64}$") { throw "Docker did not return a content digest." }
  if ([string]::IsNullOrWhiteSpace($imageArchitecture)) { throw "Docker did not return the image architecture." }
  $rawSummaryJSON = ($runSummaries | ConvertTo-Json -Depth 8 -Compress)

  $provenance = [ordered]@{
    source_commit = $sourceCommit
    clean_tree = $true
    image_ref = $Image
    image_digest = $imageDigest
    dependency_lock_digest = "sha256:$lockHash"
    producer = $Producer
    artifact_digest = Get-SHA256Text $rawSummaryJSON
  }
  if ($CiRunUrl) { $provenance.ci_run_url = $CiRunUrl }

  $hardwareClass = if ($Producer -eq "github-actions") { "github-hosted-runner" } else { "local-docker" }
  $v2 = [ordered]@{
    schema_version = 2
    run_id = [guid]::NewGuid().ToString()
    project = "mini-aws-emulator"
    benchmark_id = "aws-sdk-kumo-conformance"
    workload = [ordered]@{
      version = "2.0.0"
      fixture_digest = $fixtureDigest
      config_digest = $configDigest
      warmup_iterations = $warmupOperations
      measured_iterations = $measuredOperations
      concurrency = 1
    }
    metrics = $metrics
    execution = [ordered]@{
      command = "pwsh -NoProfile -File tools/benchmark-v2.ps1 -Image $Image -WarmupIterations $WarmupIterations -Iterations $Iterations -Repeat $Repeat"
      started_at = $startedAt.ToString("o")
      duration_seconds = [Math]::Round($timer.Elapsed.TotalSeconds, 3)
      exit_code = 0
      repeat = $Repeat
    }
    environment = [ordered]@{
      runtime = "Go $goVersion; AWS SDK Go v2 $sdkVersion; Smithy Go $smithyVersion; Kumo $kumoVersion"
      architecture = "linux/$imageArchitecture Docker image"
      hardware_class = $hardwareClass
      cloud_provider_mode = "kumo-local-first"
      provider_image = "ghcr.io/sivchari/kumo:$kumoVersion@$kumoDigest"
      implemented_services = "s3,sqs,dynamodb"
      aws_parity = "not_measured"
    }
    provenance = $provenance
    comparability_key = "aws-sdk-kumo:2.0.0:kumo-${kumoVersion}:sdk-${sdkVersion}:go-${goVersion}:s3-sqs-dynamodb:warmup-${WarmupIterations}:measure-${Iterations}:c1"
  }

  if ($v2.execution.repeat -lt 3) { throw "V2 publication requires at least three repetitions." }
  if ($v2.workload.warmup_iterations -lt 1) { throw "V2 publication requires real warmup operations." }
  foreach ($metric in $v2.metrics) {
    if (@($metric.samples).Count -ne $Repeat) { throw "Metric $($metric.name) does not retain one sample per repetition." }
    if ($metric.failures -lt 0) { throw "Metric $($metric.name) has an invalid failure count." }
  }
  if ($conformance -ne 100 -or $operationFailures -ne 0) { throw "Functional benchmark gate failed." }

  $output = Join-Path $root $OutputPath
  New-Item -ItemType Directory -Force -Path (Split-Path -Parent $output) | Out-Null
  [IO.File]::WriteAllText($output, (($v2 | ConvertTo-Json -Depth 14) + [Environment]::NewLine), [Text.UTF8Encoding]::new($false))
  $v2 | ConvertTo-Json -Depth 14
  Write-Host "v2_result=$output"
  Write-Host "source_commit=$sourceCommit"
  Write-Host "image_digest=$imageDigest"
  Write-Host "artifact_digest=$($provenance.artifact_digest)"
} finally {
  Pop-Location
}
