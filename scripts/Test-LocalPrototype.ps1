cls
# Local-only acceptance check. It never uploads reports, reads personal source
# folders, enumerates hardware, prints secrets, or changes Windows settings.
# Run after installing the tested prototype with a NEW EMPTY application library.
$ErrorActionPreference = 'Stop'
$env:PSModulePath = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\Modules'
Import-Module Microsoft.PowerShell.Management -Force
Import-Module Microsoft.PowerShell.Security -Force
$principal = [Security.Principal.WindowsPrincipal]::new([Security.Principal.WindowsIdentity]::GetCurrent())
if (!$principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Run this service-restart test in an administrator PowerShell window.' }
function Assert([bool]$condition,[string]$message) { if (!$condition) { throw $message }; Write-Host "PASS: $message" }
$service = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
if (!$service -or $service.PathName -notmatch '^"([^"]+)"') { throw 'Install the prototype before running this check.' }
$binary = $Matches[1]
$state = (Get-ItemProperty 'HKLM:\Software\ImmichClone').State
$root = Join-Path ([IO.Path]::GetTempPath()) ('immich-local-test-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory $root | Out-Null
function Login {
    $script:session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $code = (Get-Content "$state/access-code.txt" -Raw).Trim()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $script:session
}
try {
    Assert ($service.StartName -eq 'NT AUTHORITY\LocalService') 'restricted service account'
    Assert ((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') 'native application responds'
    Login
    $existing = @(Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session)
    if ($existing.Count -ne 0) { throw 'This check requires a new empty test library. Nothing was uploaded or deleted.' }
    $fixture = Join-Path $root 'synthetic-test.txt'
    [IO.File]::WriteAllText($fixture,'Synthetic local acceptance fixture. No personal media.')
    $result = Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=synthetic-test.txt' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $fixture -WebSession $session
    $id = $result.asset.id
    Assert ($id -eq (Get-FileHash $fixture).Hash.ToLowerInvariant()) 'upload SHA256 matches source'
    $image = Join-Path $root 'synthetic-test.png'
    [IO.File]::WriteAllBytes($image,[Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAEAAAAAwCAIAAAAuKetIAAAAU0lEQVR4nO3PQQ3AIADAQEAbjhCLnYngcVnSU9DOfe74s6UDXjWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaB9L9IB7tRAThgAAAAASUVORK5CYII='))
    $photo = (Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=synthetic-test.png' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $image -WebSession $session).asset
    $ready = $false
    for ($attempt=0;$attempt -lt 20;$attempt++) {
        $items = @(Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session)
        $stored = $items | Where-Object { $_.id -eq $photo.id }
        if ($stored.preview -eq 'ready') { $ready=$true; break }
        Start-Sleep -Seconds 1
    }
    Assert $ready 'native image decoding and persistent preview job'
    Restart-Service ImmichClone
    Login
    $download = Join-Path $root 'download.txt'
    Invoke-WebRequest "http://127.0.0.1:2283/api/assets/$id/original" -WebSession $session -OutFile $download -UseBasicParsing
    Assert ((Get-FileHash $download).Hash -eq (Get-FileHash $fixture).Hash) 'original preserved across service restart'
    $backup = Join-Path $root 'backup.zip'
    Invoke-WebRequest http://127.0.0.1:2283/api/backup -WebSession $session -OutFile $backup -UseBasicParsing
    $restored = Join-Path $root 'restored'
    & $binary -mode restore -data $restored -archive $backup
    if ($LASTEXITCODE -ne 0) { throw 'Local restore failed' }
    & $binary -mode verify -data $restored
    if ($LASTEXITCODE -ne 0) { throw 'Local original verification failed' }
    Assert ((Get-FileHash "$restored/originals/$id").Hash -eq (Get-FileHash $fixture).Hash) 'independent restore preserves original bytes'
    Write-Host 'Local acceptance check passed. Two synthetic originals remain in the test library.'
    Write-Host 'No machine specifications, paths, access code or media were transmitted outside this PC.'
} finally {
    Remove-Item $root -Recurse -Force -ErrorAction Continue
}
