$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('media compatibility '+[guid]::NewGuid().ToString('N'))
$priorFixtures=$env:IMMICH_COMPAT_FIXTURES
$priorProvider=$env:IMMICH_COMPAT_PROVIDER
try {
    New-Item -ItemType Directory "$root/fixtures" -Force | Out-Null
    ./scripts/Prepare-MediaProvider.ps1 -Destination "$root/media"
    python -B experiments/media-compatibility/prepare.py "$root/fixtures"
    if ($LASTEXITCODE -ne 0) { throw 'Independent compatibility image fixture generation failed' }
    ./experiments/media-compatibility/prepare-video.ps1 -Root "$root/fixtures"
    $env:IMMICH_COMPAT_FIXTURES="$root/fixtures"
    $env:IMMICH_COMPAT_PROVIDER="$root/media"
    go test -mod=readonly -v -count=1 -timeout=120s ./internal/library -run '^TestMediaCompatibilityImageMatrix$'
    if ($LASTEXITCODE -ne 0) { throw 'Production image matrix or independent recovery failed' }
    go test -mod=readonly -v -count=1 -timeout=120s ./internal/media -run 'TestActualVideoMetadataCompatibilityMatrix|TestMetadataRoutingAndBounds|TestAnalysisOutputAllowanceRemainsFinite'
    if ($LASTEXITCODE -ne 0) { throw 'Actual video metadata/resource compatibility failed' }
} finally {
    $env:IMMICH_COMPAT_FIXTURES=$priorFixtures
    $env:IMMICH_COMPAT_PROVIDER=$priorProvider
    Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
