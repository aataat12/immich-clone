# Synthetic filesystem feasibility probe; not a production storage implementation.
$ErrorActionPreference = 'Stop'
$root = Join-Path ([IO.Path]::GetTempPath()) ('immich-storage-' + [guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($root) | Out-Null
$checks = [ordered]@{}
function Assert([bool]$Condition, [string]$Name) {
    if (!$Condition) { throw "Failed: $Name" }
    $checks[$Name] = 'passed'
}
try {
    $bytes = New-Object byte[] 1048576
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
    $staging = Join-Path $root 'upload.part'
    $original = Join-Path $root 'original.bin'
    $stream = [IO.FileStream]::new($staging, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    try { $stream.Write($bytes, 0, $bytes.Length); $stream.Flush($true) } finally { $stream.Dispose() }
    $expected = (Get-FileHash $staging -Algorithm SHA256).Hash
    [IO.File]::Move($staging, $original)
    Assert (!(Test-Path $staging) -and (Test-Path $original)) 'same-volume-publish'
    Assert ((Get-FileHash $original).Hash -eq $expected) 'original-byte-preservation'

    # An interrupted upload is not visible as an original.
    [IO.File]::WriteAllBytes($staging, $bytes[0..1023])
    Assert ((Get-Item $staging).Length -lt $bytes.Length -and (Get-FileHash $original).Hash -eq $expected) 'partial-upload-isolation'
    $rejected = $false
    try { [IO.File]::Move($staging, $original) } catch [IO.IOException] { $rejected = $true }
    Assert ($rejected -and (Get-FileHash $original).Hash -eq $expected) 'existing-original-not-overwritten'

    # Independent process observes the same committed hash.
    $env:IMMICH_PROBE_PATH = $original
    $env:IMMICH_PROBE_HASH = $expected
    & powershell.exe -NoProfile -NonInteractive -Command 'if ((Get-FileHash $env:IMMICH_PROBE_PATH).Hash -ne $env:IMMICH_PROBE_HASH) { exit 1 }; exit 0'
    Assert ($LASTEXITCODE -eq 0) 'independent-process-read'

    $backup = Join-Path $root 'backup'
    $restore = Join-Path $root 'restore'
    [IO.Directory]::CreateDirectory($backup) | Out-Null
    [IO.Directory]::CreateDirectory($restore) | Out-Null
    Copy-Item $original (Join-Path $backup 'original.bin')
    Remove-Item $original
    Copy-Item (Join-Path $backup 'original.bin') (Join-Path $restore 'original.bin')
    Assert ((Get-FileHash (Join-Path $restore 'original.bin')).Hash -eq $expected) 'synthetic-original-backup-restore'
    [ordered]@{
        scope = 'Synthetic file operations only; not application-consistent backup, crash/power-loss durability, media processing, service identity, or Windows 10 certification'
        os = (Get-CimInstance Win32_OperatingSystem).Caption
        version = [Environment]::OSVersion.Version.ToString()
        powershell = $PSVersionTable.PSVersion.ToString()
        fileSystem = (Get-Volume -DriveLetter ([IO.Path]::GetPathRoot($root).Substring(0,1))).FileSystem
        checks = $checks
    } | ConvertTo-Json -Depth 5
} finally {
    Remove-Item Env:IMMICH_PROBE_PATH, Env:IMMICH_PROBE_HASH -ErrorAction SilentlyContinue
    Remove-Item $root -Recurse -Force -ErrorAction Continue
}
