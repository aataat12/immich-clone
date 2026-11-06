$ErrorActionPreference = 'Stop'
function Test-BoundedVolume([int]$sizeMiB,[string]$pattern) {
    $root=Join-Path $env:RUNNER_TEMP ('immich-volume-'+[guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory $root | Out-Null
    $disk=Join-Path $root 'bounded.vhd'
    $letter=@('R','S','T','U','V') | Where-Object { !(Test-Path ($_+':\')) } | Select-Object -First 1
    if (!$letter) { throw 'No unused test drive letter' }
    if ($sizeMiB -notin @(96,192)) { throw 'Unapproved disposable test volume size' }
    $commands=Join-Path $root 'create.txt'
    @("create vdisk file=`"$disk`" maximum=$sizeMiB type=expandable", "select vdisk file=`"$disk`"", 'attach vdisk', 'create partition primary', 'format fs=ntfs quick label=ImmichProbe', "assign letter=$letter") | Set-Content $commands
    try {
        & diskpart.exe /s $commands
        if ($LASTEXITCODE -ne 0 -or !(Test-Path ($letter+':\'))) { throw 'Bounded test volume could not be mounted' }
        $volume=Get-Volume -DriveLetter $letter
        if ($volume.Size -gt (($sizeMiB+1)*1MB) -or $volume.FileSystem -ne 'NTFS' -or $volume.FileSystemLabel -ne 'ImmichProbe') { throw 'Unexpected volume; refusing fill' }
        $env:IMMICH_SMALL_VOLUME=$letter+':\'
        go test -mod=readonly -v -timeout=120s ./internal/library -run $pattern -count=1
        if ($LASTEXITCODE -ne 0) { throw 'Actual full-disk recovery test failed' }
    } finally {
        Remove-Item Env:IMMICH_SMALL_VOLUME -ErrorAction SilentlyContinue
        @("select vdisk file=`"$disk`"", 'detach vdisk') | Set-Content $commands
        & diskpart.exe /s $commands
        $resolved=[IO.Path]::GetFullPath($root)
        $parent=[IO.Path]::GetFullPath($env:RUNNER_TEMP).TrimEnd('\')+'\'
        if(!$resolved.StartsWith($parent,[StringComparison]::OrdinalIgnoreCase)){throw 'Volume fixture cleanup target refused'}
        Remove-Item -LiteralPath $resolved -Recurse -Force -ErrorAction Continue
    }
}
Test-BoundedVolume 96 '^TestActualDiskFull$'
Test-BoundedVolume 96 '^TestActualCurationDiskFull$'
Test-BoundedVolume 96 '^TestActualDeletionDiskFull$'
# Backup preflight intentionally reserves space; this separate fixture must first fit.
Test-BoundedVolume 192 '^TestActualBackupDiskFull$'
