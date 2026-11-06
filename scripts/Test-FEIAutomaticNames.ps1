$ErrorActionPreference='Stop'
$evaluationBase=[IO.Path]::GetFullPath($env:RUNNER_TEMP).TrimEnd([IO.Path]::DirectorySeparatorChar)+[IO.Path]::DirectorySeparatorChar
$evaluationRoot=[IO.Path]::GetFullPath((Join-Path $evaluationBase ('fei automatic '+[guid]::NewGuid().ToString('N'))))
if (!$evaluationRoot.StartsWith($evaluationBase,[StringComparison]::OrdinalIgnoreCase) -or [IO.Path]::GetDirectoryName($evaluationRoot) -ne $evaluationBase.TrimEnd([IO.Path]::DirectorySeparatorChar)) { throw 'Research temporary directory boundary invalid' }
$priorRoot=$env:IMMICH_FEI_QUALITY_ROOT
try {
    New-Item -ItemType Directory -Path $evaluationRoot | Out-Null
    python -B experiments/automatic-names-fei/prepare-fei.py $evaluationRoot
    if ($LASTEXITCODE -ne 0) { throw 'Independent FEI research inputs unavailable or invalid' }
    ./scripts/Prepare-AIProvider.ps1 -Destination (Join-Path $evaluationRoot 'ai')
    $env:IMMICH_FEI_QUALITY_ROOT=$evaluationRoot
    go test -mod=readonly -v -count=1 -timeout=26m ./internal/library -run '^TestActualNativeIndependentFEIAutomaticQuality$'
    if ($LASTEXITCODE -ne 0) { throw 'Independent frozen FEI automatic naming gate failed' }
} finally {
    $env:IMMICH_FEI_QUALITY_ROOT=$priorRoot
    # Both absolute paths were verified before creation; no unrelated tree is removed.
    if (Test-Path -LiteralPath $evaluationRoot) { Remove-Item -LiteralPath $evaluationRoot -Recurse -Force }
}
