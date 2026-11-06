param([switch]$AutomaticNames)
$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('native quality '+[guid]::NewGuid().ToString('N'))
$prior=$env:IMMICH_NATIVE_QUALITY_ROOT
$priorAutomatic=$env:IMMICH_AUTOMATIC_QUALITY
try {
    New-Item -ItemType Directory $root | Out-Null
    ./scripts/Prepare-AIProvider.ps1 -Destination "$root/ai"
    python -B experiments/face-quality/prepare-native.py $root
    if ($LASTEXITCODE -ne 0) { throw 'Native quality evaluation inputs unavailable' }
    $env:IMMICH_NATIVE_QUALITY_ROOT=$root
    if ($AutomaticNames) { $env:IMMICH_AUTOMATIC_QUALITY='1' } else { $env:IMMICH_AUTOMATIC_QUALITY=$null }
    go test -mod=readonly -v -count=1 -timeout=14m ./internal/library -run '^TestActualNativeCrossPhotoQuality$'
    if ($LASTEXITCODE -ne 0) { throw 'Frozen native Go/provider quality acceptance failed' }
} finally {
    $env:IMMICH_NATIVE_QUALITY_ROOT=$prior
    $env:IMMICH_AUTOMATIC_QUALITY=$priorAutomatic
    Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
