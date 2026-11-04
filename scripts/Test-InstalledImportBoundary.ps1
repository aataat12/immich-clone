$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('installed-boundary-'+[guid]::NewGuid().ToString('N'))
$app=Join-Path $root 'application'
$state=Join-Path $root 'state'
$probe=$null
try {
    if((Get-Service ImmichClone -ErrorAction SilentlyContinue) -or (Test-Path 'HKLM:\Software\ImmichClone')){throw 'Disposable clean runner required; refusing an existing installation'}
    $drive=[IO.DriveInfo]::new([IO.Path]::GetPathRoot($root))
    if($drive.AvailableFreeSpace -lt 12GB){throw 'Actual installed boundary needs at least 12 GiB disposable free space'}
    New-Item -ItemType Directory $root | Out-Null
    $setup=(Resolve-Path 'accepted-installer/ImmichClone-0.1.36-prototype-x64-setup.exe').Path
    if((Get-FileHash $setup).Hash.ToLowerInvariant() -ne '33d044091b0c03861a4357ccc7ccbf75ae42340d87be72659455255383a5c79a'){throw 'Accepted installer checksum mismatch'}
    $installed=Start-Process $setup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$app+'"'),('/STATE="'+$state+'"')) -Wait -PassThru
    if($installed.ExitCode -ne 0){throw 'Accepted installer failed'}
    $service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
    if($service.StartName -ne 'NT AUTHORITY\LocalService' -or (& "$app/immichclone.exe" -mode version) -ne '0.1.36-prototype'){throw 'Installed executable/account baseline mismatch'}
    $env:IMMICH_INSTALLED_BOUNDARY_ROOT=$root
    $node=(Get-Command node).Source
    $probe=Start-Process $node -ArgumentList @('tests/installed-import-boundary.cjs','start') -PassThru -RedirectStandardOutput "$root/probe.out" -RedirectStandardError "$root/probe.err"
    $deadline=[DateTime]::UtcNow.AddSeconds(240)
    while(!(Test-Path "$root/crash-ready") -and !$probe.HasExited -and [DateTime]::UtcNow -lt $deadline){Start-Sleep -Milliseconds 100}
    if(!(Test-Path "$root/crash-ready")){Get-Content "$root/probe.err";throw 'Actual 4 GiB transfer barrier not reached'}
    $servicePID=(Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").ProcessId
    Stop-Process -Id $servicePID -Force
    if(!$probe.WaitForExit(30000)){throw 'Interrupted browser did not finish recovery'}
    Get-Content "$root/probe.out"
    if($probe.ExitCode -ne 0){Get-Content "$root/probe.err";throw 'Installed interruption phase failed'}
    $deadline=[DateTime]::UtcNow.AddSeconds(15)
    while((Get-Service ImmichClone).Status -ne 'Stopped' -and [DateTime]::UtcNow -lt $deadline){Start-Sleep -Milliseconds 100}
    Start-Service ImmichClone
    & node tests/installed-import-boundary.cjs finish
    if($LASTEXITCODE -ne 0){throw 'Installed 4 GiB resume/readback failed'}
    Stop-Service ImmichClone
    & "$app/immichclone.exe" -mode verify -data $state
    if($LASTEXITCODE -ne 0){throw 'Installed offline original verification failed'}
} finally {
    if($probe -and !$probe.HasExited){Stop-Process -Id $probe.Id -Force;$null=$probe.WaitForExit(10000)}
    if(Test-Path "$app/unins000.exe"){$removed=Start-Process "$app/unins000.exe" -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru;if($removed.ExitCode -ne 0){Write-Warning 'Disposable uninstall did not succeed'}}
    $remaining=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'" -ErrorAction SilentlyContinue
    if($remaining -and $remaining.PathName.StartsWith(('"'+$app+'\immichclone.exe"'),[StringComparison]::OrdinalIgnoreCase)){Stop-Service ImmichClone -ErrorAction SilentlyContinue;sc.exe delete ImmichClone}
    Remove-Item Env:IMMICH_INSTALLED_BOUNDARY_ROOT -ErrorAction SilentlyContinue
    Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
