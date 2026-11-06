$ErrorActionPreference='Stop'
$evaluationBase=[IO.Path]::GetFullPath($env:RUNNER_TEMP).TrimEnd([IO.Path]::DirectorySeparatorChar)+[IO.Path]::DirectorySeparatorChar
$evaluationRoot=[IO.Path]::GetFullPath((Join-Path $evaluationBase ('nottingham automatic '+[guid]::NewGuid().ToString('N'))))
if (!$evaluationRoot.StartsWith($evaluationBase,[StringComparison]::OrdinalIgnoreCase) -or [IO.Path]::GetDirectoryName($evaluationRoot) -ne $evaluationBase.TrimEnd([IO.Path]::DirectorySeparatorChar)) { throw 'Research temporary directory boundary invalid' }
$priorRoot=$env:IMMICH_NOTTINGHAM_QUALITY_ROOT
try {
    New-Item -ItemType Directory -Path $evaluationRoot | Out-Null
    python -B experiments/automatic-names-nottingham/prepare-nottingham.py $evaluationRoot
    if ($LASTEXITCODE -ne 0) { throw 'Independent Nottingham research inputs unavailable or invalid' }
    ./scripts/Prepare-AIProvider.ps1 -Destination (Join-Path $evaluationRoot 'ai')
    $env:IMMICH_NOTTINGHAM_QUALITY_ROOT=$evaluationRoot
    go test -mod=readonly -v -count=1 -timeout=20m ./internal/library -run '^TestActualNativeIndependentNottinghamAutomaticQuality$'
    if ($LASTEXITCODE -ne 0) { throw 'Independent frozen Nottingham automatic naming gate failed' }
} finally {
    $env:IMMICH_NOTTINGHAM_QUALITY_ROOT=$priorRoot
    # Both absolute paths were verified before creation; no unrelated tree is removed.
    if (Test-Path -LiteralPath $evaluationRoot) { Remove-Item -LiteralPath $evaluationRoot -Recurse -Force }
}
