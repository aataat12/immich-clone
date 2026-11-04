$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('installed-import-endurance-'+[guid]::NewGuid().ToString('N'))
$app=Join-Path $root 'application';$state=Join-Path $root 'state'
function Checked([string]$exe,[string[]]$argv){& $exe @argv;if($LASTEXITCODE -ne 0){throw 'Installed import endurance command failed'}}
try{
 if((Get-Service ImmichClone -ErrorAction SilentlyContinue) -or (Test-Path 'HKLM:\Software\ImmichClone')){throw 'Clean disposable runner required'}
 if([IO.DriveInfo]::new([IO.Path]::GetPathRoot($root)).AvailableFreeSpace -lt 4GB){throw 'At least 4 GiB disposable free space required'}
 New-Item -ItemType Directory $root|Out-Null
 $setup=(Resolve-Path 'accepted-installer/ImmichClone-0.1.38-prototype-x64-setup.exe').Path
 if((Get-FileHash $setup).Hash.ToLowerInvariant() -ne '14e1f0dd6b5bc709d45235963e137e47ee5740fee13ec3c8c03d4756ddd24038'){throw 'Accepted installer hash mismatch'}
 $p=Start-Process $setup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$app+'"'),('/STATE="'+$state+'"')) -Wait -PassThru
 if($p.ExitCode -ne 0){throw 'Accepted installation failed'}
 $service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
 if($service.StartName -ne 'NT AUTHORITY\LocalService' -or (& "$app/immichclone.exe" -mode version) -ne '0.1.38-prototype'){throw 'Installed account/version mismatch'}
 $env:IMMICH_IMPORT_ENDURANCE_ROOT=$root
 Checked node @('tests/import-endurance.cjs','first')
 Restart-Service ImmichClone
 Checked node @('tests/import-endurance.cjs','second')
 Restart-Service ImmichClone
 Checked node @('tests/import-endurance.cjs','verify')
 Stop-Service ImmichClone
 Checked "$app/immichclone.exe" @('-mode','verify','-data',$state)
 Write-Output 'PASS: installed bounded sustained mixed import and replay; not 50k ingestion, broad natural-media corpus, long-duration endurance or Windows10 proof'
}finally{
 if(Test-Path "$app/unins000.exe"){$null=Start-Process "$app/unins000.exe" -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru}
 $service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'" -ErrorAction SilentlyContinue
 if($service -and $service.PathName.StartsWith(('"'+$app+'\immichclone.exe"'),[StringComparison]::OrdinalIgnoreCase)){Stop-Service ImmichClone -ErrorAction SilentlyContinue;sc.exe delete ImmichClone}
 Remove-Item Env:IMMICH_IMPORT_ENDURANCE_ROOT -ErrorAction SilentlyContinue
 Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
