$ErrorActionPreference = 'Stop'
$root = Join-Path $env:RUNNER_TEMP ('installer test ' + [guid]::NewGuid().ToString('N'))
$app = Join-Path $root 'application'
$state = Join-Path $root 'state'
New-Item -ItemType Directory $root | Out-Null
function Assert([bool]$condition,[string]$message) { if (!$condition) { throw $message }; Write-Output "PASS: $message" }
function Install([string]$version) {
    $setup = (Resolve-Path "dist/ImmichClone-$version-prototype-x64-setup.exe").Path
    $process = Start-Process $setup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$app+'"'),('/STATE="'+$state+'"'),('/LOG="'+$root+'/setup-'+$version+'.log"')) -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        Get-Content "$root/setup-$version.log" -Tail 25 | ForEach-Object { Write-Host $_ }
        if (Test-Path "$app/install-diagnostic.txt") { Get-Content "$app/install-diagnostic.txt" | ForEach-Object { Write-Host $_ } }
    }
    return $process.ExitCode
}
try {
    Assert ((Install '0.1.0') -eq 0) 'installer fresh install succeeds in path containing spaces'
    $service = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
    Write-Output ('Disposable CI service evidence: '+($service | Select-Object Name,StartName,StartMode,State | ConvertTo-Json -Compress))
    if (!$service -or $service.StartName -ne 'NT AUTHORITY\LocalService' -or $service.StartMode -ne 'Auto') { Get-Content "$root/setup-0.1.0.log" -Tail 35 }
    Assert ($service.StartName -eq 'NT AUTHORITY\LocalService' -and $service.StartMode -eq 'Auto') 'installer configures restricted automatic service'
    & "$env:SystemRoot/System32/WindowsPowerShell/v1.0/powershell.exe" -NoProfile -NonInteractive -ExecutionPolicy Bypass -File scripts/Test-LocalPrototype.ps1
    Assert ($LASTEXITCODE -eq 0) 'local acceptance helper executes under built-in Windows PowerShell'
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $code = (Get-Content "$state/access-code.txt" -Raw).Trim()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $fixture = Join-Path $root 'synthetic.txt'
    [IO.File]::WriteAllText($fixture,'synthetic installer preservation fixture')
    $asset = (Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=synthetic.txt' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $fixture -WebSession $session).asset
    $hash = (Get-FileHash $fixture).Hash
    Assert ((Install '0.1.1') -eq 0) 'installer upgrade succeeds'
    Assert ((& "$app/immichclone.exe" -mode version) -eq '0.1.1-prototype') 'upgrade deploys new executable version'
    Assert ((Get-FileHash "$state/originals/$($asset.id)").Hash -eq $hash) 'upgrade preserves original bytes'
    $env:IMMICH_INSTALL_TEST_FAILURE = '1'
    $failedCode = Install '0.1.2'
    Remove-Item Env:IMMICH_INSTALL_TEST_FAILURE
    Assert ($failedCode -ne 0) 'injected failed upgrade returns failure'
    Assert ((& "$app/immichclone.exe" -mode version) -eq '0.1.1-prototype') 'failed upgrade restores previous executable'
    Assert ((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') 'rollback restores healthy service'
    Assert ((Get-FileHash "$state/originals/$($asset.id)").Hash -eq $hash) 'failed upgrade preserves original bytes'
    $recoveryTool = Join-Path $root 'recovery-tool.exe'
    Copy-Item "$app/immichclone.exe" $recoveryTool
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $archive = Join-Path $root 'recovery.zip'
    Invoke-WebRequest http://127.0.0.1:2283/api/backup -WebSession $session -OutFile $archive
    $uninstall = Start-Process "$app/unins000.exe" -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru
    Assert ($uninstall.ExitCode -eq 0) 'silent uninstall succeeds'
    Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue)) 'uninstall removes service'
    Assert ((Get-FileHash "$state/originals/$($asset.id)").Hash -eq $hash) 'uninstall retains original bytes'
    Assert ((Test-Path "$state/catalog.db") -and (Test-Path "$state/access-code.txt")) 'uninstall retains catalog and access code'
    # Make the old installation unavailable without destroying its files.
    Move-Item -LiteralPath $state -Destination ($state+'.offline')
    $state = Join-Path $root 'restored state'
    & $recoveryTool -mode restore -data $state -archive $archive
    Assert ($LASTEXITCODE -eq 0) 'offline restoration succeeds at a different location'
    Assert ((Install '0.1.1') -eq 0) 'installer starts a restored library without the original data folder'
    $session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
    $restoredCode = (Get-Content "$state/access-code.txt" -Raw).Trim()
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$restoredCode}|ConvertTo-Json) -WebSession $session
    $retrieved = Join-Path $root 'restored-download.txt'
    Invoke-WebRequest "http://127.0.0.1:2283/api/assets/$($asset.id)/original" -WebSession $session -OutFile $retrieved
    Assert ((Get-FileHash $retrieved).Hash -eq $hash) 'reinstalled service authenticates with restored secret and returns original bytes'
    $uninstall = Start-Process "$app/unins000.exe" -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru
    Assert ($uninstall.ExitCode -eq 0) 'restored installation uninstalls cleanly'
} finally {
    Remove-Item Env:IMMICH_INSTALL_TEST_FAILURE -ErrorAction SilentlyContinue
    if (Get-Service ImmichClone -ErrorAction SilentlyContinue) { Stop-Service ImmichClone -Force -ErrorAction Continue; & sc.exe delete ImmichClone }
    # Only this disposable, randomly generated CI directory is removed.
    Remove-Item $root -Recurse -Force -ErrorAction Continue
}
