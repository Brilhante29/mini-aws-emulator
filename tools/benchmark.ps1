param(
    [string]$Image = "mini-aws-emulator",
    [int]$Iterations = 25,
    [int]$Repeat = 1,
    [string]$OutputFile = "kumo-baseline.json"
)
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$results = Join-Path $root "benchmarks/results"
New-Item -ItemType Directory -Force -Path $results | Out-Null

& docker build -t $Image $root
if ($LASTEXITCODE -ne 0) { throw "Docker build failed" }

$output = & docker run --rm `
    -e "BENCHMARK_ITERATIONS=$Iterations" `
    -e "REPEAT=$Repeat" `
    $Image
if ($LASTEXITCODE -ne 0) { throw "Docker benchmark failed" }
$json = ($output -join "`n") + "`n"
$parsed = $json | ConvertFrom-Json
if ($parsed.metric -ne "conformance_rate_percent") { throw "Unexpected benchmark metric" }
$path = Join-Path $results $OutputFile
[IO.File]::WriteAllText($path, $json, [Text.UTF8Encoding]::new($false))
Write-Output $json
