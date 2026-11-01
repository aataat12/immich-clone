$ErrorActionPreference = 'Stop'
$root = Join-Path $env:RUNNER_TEMP ('immich-app-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory $root | Out-Null
$binary = Join-Path $root 'immichclone.exe'
$state = Join-Path $root 'state'
$name = 'ImmichClone'
function Checked([string]$exe,[string[]]$arguments) { & $exe @arguments; if ($LASTEXITCODE -ne 0) { throw "$exe failed: $LASTEXITCODE" } }
function Assert([bool]$condition,[string]$message) { if (!$condition) { throw $message }; Write-Output "PASS: $message" }
function Login {
    $code = (Get-Content "$state/access-code.txt" -Raw).Trim()
    $script:session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $script:session
}
try {
    Checked go @('build','-trimpath','-ldflags','-X main.version=0.1.0-prototype','-o',$binary,'./cmd/immichclone')
    Checked $binary @('-mode','init','-data',$state)
    Checked icacls @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
    Checked icacls @($state,'/grant','*S-1-5-19:(OI)(CI)M')
    $credential = [PSCredential]::new('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
    New-Service -Name $name -BinaryPathName ('"'+$binary+'" -mode service -data "'+$state+'"') -Credential $credential | Out-Null
    Start-Service $name
    Assert ((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') 'actual application starts as LocalService'
    Login
    # Generate a fixture using the already-executed standard image encoders in tests.
    # This tiny known PNG fixture exercises actual upload/preview retrieval in service.
    $fixture = Join-Path $root 'synthetic.png'
    [IO.File]::WriteAllBytes($fixture,[Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAEAAAAAwCAIAAAAuKetIAAAAU0lEQVR4nO3PQQ3AIADAQEAbjhCLnYngcVnSU9DOfe74s6UDXjWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaA1oDWgNaB9L9IB7tRAThgAAAAASUVORK5CYII='))
    $headers = @{'X-Immich-Request'='1'}
    $result = Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=synthetic.png' -Method Post -Headers $headers -InFile $fixture -WebSession $session
    Assert (!$result.duplicate -and $result.asset.id -eq (Get-FileHash $fixture).Hash.ToLowerInvariant()) 'upload returns original SHA256'
    $id = $result.asset.id
    $duplicate = Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=duplicate.png' -Method Post -Headers $headers -InFile $fixture -WebSession $session
    Assert $duplicate.duplicate 'duplicate upload resolves without overwriting original'
    $items = @()
    for ($attempt=0;$attempt -lt 15;$attempt++) {
        $items = @(Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session)
        if ($items.Count -eq 1 -and $items[0].preview -ne 'queued') { break }
        Start-Sleep -Seconds 1
    }
    Assert ($items.Count -eq 1 -and $items[0].preview -eq 'ready') 'persistent preview job produces PNG under LocalService'
    $preview = Join-Path $root 'preview.png'
    Invoke-WebRequest "http://127.0.0.1:2283/api/assets/$id/preview" -WebSession $session -OutFile $preview
    Assert ((Get-Item $preview).Length -gt 0) 'generated preview can be retrieved'
    if ($env:IMMICH_BROWSER_MODULE) {
        $env:IMMICH_TEST_STATE = $state
        Checked node @('tests/browser.cjs')
        Remove-Item Env:IMMICH_TEST_STATE
    }
    Restart-Service $name
    Login
    $download = Join-Path $root 'download.png'
    Invoke-WebRequest "http://127.0.0.1:2283/api/assets/$id/original" -WebSession $session -OutFile $download
    Assert ((Get-FileHash $download).Hash -eq (Get-FileHash $fixture).Hash) 'original download preserved across service restart'
    $backup = Join-Path $root 'backup.zip'
    Invoke-WebRequest http://127.0.0.1:2283/api/backup -WebSession $session -OutFile $backup
    Stop-Service $name
    $restored = Join-Path $root 'restored'
    Checked $binary @('-mode','restore','-data',$restored,'-archive',$backup)
    Checked $binary @('-mode','verify','-data',$restored)
    Assert ((Get-FileHash "$restored/originals/$id").Hash -eq (Get-FileHash $fixture).Hash) 'independent backup restoration verifies original bytes'
    # Prove inability to write rather than relying on administrator test execution.
    Checked icacls @($state,'/deny','*S-1-5-19:(OI)(CI)W')
    $failed = $false
    try { Start-Service $name -ErrorAction Stop } catch { $failed = $true }
    Assert $failed 'service startup surfaces denied storage permissions'
    Checked icacls @($state,'/remove:d','*S-1-5-19')
    Write-Output 'Application slice complete on Server 2022; Windows 10 remains untested.'
} finally {
    if (Get-Service $name -ErrorAction SilentlyContinue) { Stop-Service $name -Force -ErrorAction Continue; Checked sc.exe @('delete',$name) }
    Remove-Item $root -Recurse -Force -ErrorAction Continue
}
