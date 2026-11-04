param([Parameter(Mandatory=$true)][string]$Binary)
$ErrorActionPreference = 'Stop'
$Binary = (Resolve-Path $Binary).Path
$root = Join-Path ([IO.Path]::GetTempPath()) ('restore limits synthetic ' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $root | Out-Null
$oldFixture=$env:IMMICH_RESTORE_LIMIT_FIXTURE
$oldTarget=$env:IMMICH_RESTORE_LIMIT_TARGET
function Invoke-SyntheticApp([string[]]$Arguments) {
    $stdout=Join-Path $root 'stdout.txt'; $stderr=Join-Path $root 'stderr.txt'
    $quoted = $Arguments | ForEach-Object { '"' + $_.Replace('"','\"') + '"' }
    $p=Start-Process -FilePath $Binary -ArgumentList $quoted -Wait -PassThru -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    return $p.ExitCode
}
try {
    $env:IMMICH_RESTORE_LIMIT_FIXTURE=$root
    & go test ./internal/library -run '^TestRestoreLimitsFixtureEmitter$' -count=1
    if ($LASTEXITCODE -ne 0) { throw 'Synthetic old-format fixture generation failed' }
    $env:IMMICH_RESTORE_LIMIT_FIXTURE=$null
    $archive=Join-Path $root 'fixture.zip'
    $before=(Get-FileHash $archive -Algorithm SHA256).Hash
    $logs=Join-Path $root 'logs'
    $cases=@(
      @('-restore-max-bytes','1'), @('-restore-max-entries','1'),
      @('-restore-max-manifest-bytes','1'), @('-restore-max-catalog-bytes','1024'),
      @('-restore-max-metadata-bytes','32'), @('-restore-memory-bytes','1'),
      @('-restore-timeout','1ns')
    )
    $i=0
    foreach ($case in $cases) {
      $parent=Join-Path $root ('refusal'+$i); New-Item -ItemType Directory $parent | Out-Null
      $target=Join-Path $parent 'target'
      $exit=Invoke-SyntheticApp (@('-mode','restore','-archive',$archive,'-data',$target,'-logs',$logs)+$case)
      if ($exit -eq 0 -or @(Get-ChildItem -Force $parent).Count -ne 0) { throw 'CLI budget refusal or staging cleanup failed' }
      $i++
    }
    foreach ($explicit in @($false,$true)) {
      $target=Join-Path $root ('valid'+$explicit)
      $restoreArgs=@('-mode','restore','-archive',$archive,'-data',$target,'-logs',$logs)
      if ($explicit) { $restoreArgs+=@('-restore-max-bytes','16777216','-restore-max-entries','100','-restore-max-manifest-bytes','1048576','-restore-max-catalog-bytes','8388608','-restore-max-metadata-bytes','1048576','-restore-memory-bytes','536870912','-restore-timeout','30s') }
      if ((Invoke-SyntheticApp $restoreArgs) -ne 0) { throw 'Valid CLI restore refused' }
      if ((Invoke-SyntheticApp @('-mode','verify','-data',$target,'-logs',$logs)) -ne 0) { throw 'CLI restored originals failed verification' }
      $env:IMMICH_RESTORE_LIMIT_TARGET=$target
      & go test ./internal/library -run '^TestRestoreLimitsFixtureInspector$' -count=1
      if ($LASTEXITCODE -ne 0) { throw 'CLI restored metadata inspection failed' }
      $env:IMMICH_RESTORE_LIMIT_TARGET=$null
    }
    if ((Get-FileHash $archive -Algorithm SHA256).Hash -ne $before) { throw 'CLI restore changed input archive' }
    Write-Output 'PASS: actual CLI default and explicit old-format restore, seven budget refusals, cleanup, originals and metadata preservation'
} finally {
    $env:IMMICH_RESTORE_LIMIT_FIXTURE=$oldFixture
    $env:IMMICH_RESTORE_LIMIT_TARGET=$oldTarget
    Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue
}
