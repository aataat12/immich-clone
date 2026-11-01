$ErrorActionPreference = 'Stop'
$root = Join-Path $env:RUNNER_TEMP ('immich-volume-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory $root | Out-Null
$disk = Join-Path $root 'bounded.vhd'
$letter = @('R','S','T','U','V') | Where-Object { !(Test-Path ($_+':\')) } | Select-Object -First 1
if (!$letter) { throw 'No unused test drive letter' }
$commands = Join-Path $root 'create.txt'
@("create vdisk file=`"$disk`" maximum=96 type=expandable", "select vdisk file=`"$disk`"", 'attach vdisk', 'create partition primary', 'format fs=ntfs quick label=ImmichProbe', "assign letter=$letter") | Set-Content $commands
try {
    & diskpart.exe /s $commands
    if ($LASTEXITCODE -ne 0 -or !(Test-Path ($letter+':\'))) { throw 'Bounded test volume could not be mounted' }
    $volume = Get-Volume -DriveLetter $letter
    if ($volume.Size -gt 100MB -or $volume.FileSystem -ne 'NTFS') { throw 'Unexpected test volume; refusing fill' }
    $env:IMMICH_SMALL_VOLUME = $letter+':\'
    go test -mod=readonly -v ./internal/library -run '^TestActualDiskFull$'
    if ($LASTEXITCODE -ne 0) { throw 'Actual full-disk recovery test failed' }
} finally {
    Remove-Item Env:IMMICH_SMALL_VOLUME -ErrorAction SilentlyContinue
    @("select vdisk file=`"$disk`"", 'detach vdisk') | Set-Content $commands
    & diskpart.exe /s $commands
    Remove-Item $root -Recurse -Force -ErrorAction Continue
}
