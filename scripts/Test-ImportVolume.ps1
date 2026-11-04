$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('installed-import-volume-'+[guid]::NewGuid().ToString('N'))
$app=Join-Path $root 'application'
$disk=Join-Path $root 'disposable.vhd'
$letter=@('R','S','T','U','V') | Where-Object { !(Test-Path ($_+':\')) } | Select-Object -First 1
$probe=$null
function Checked([string]$exe,[string[]]$argv){& $exe @argv;if($LASTEXITCODE -ne 0){throw 'Mounted-volume acceptance command failed'}}
function DiskCommands([string[]]$lines){$file=Join-Path $root 'diskpart.txt';$lines|Set-Content $file;Checked diskpart.exe @('/s',$file)}
try {
 if(!$letter -or (Get-Service ImmichClone -ErrorAction SilentlyContinue) -or (Test-Path 'HKLM:\Software\ImmichClone')){throw 'Unused volume letter and clean disposable runner required'}
 New-Item -ItemType Directory $root|Out-Null
 DiskCommands @("create vdisk file=`"$disk`" maximum=8192 type=expandable","select vdisk file=`"$disk`"",'attach vdisk','create partition primary','format fs=ntfs quick label=ImmichImportProbe',"assign letter=$letter")
 $volume=Get-Volume -DriveLetter $letter
 if($volume.FileSystem -ne 'NTFS' -or $volume.FileSystemLabel -ne 'ImmichImportProbe' -or $volume.Size -gt 8193MB){throw 'Unexpected disposable volume'}
 $state=$letter+':\state'
 $setup=(Resolve-Path 'accepted-installer/ImmichClone-0.1.38-prototype-x64-setup.exe').Path
 if((Get-FileHash $setup).Hash.ToLowerInvariant() -ne '14e1f0dd6b5bc709d45235963e137e47ee5740fee13ec3c8c03d4756ddd24038'){throw 'Accepted installer hash mismatch'}
 $p=Start-Process $setup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$app+'"'),('/STATE="'+$state+'"')) -Wait -PassThru
 if($p.ExitCode -ne 0){throw 'Accepted installation failed'}
 $service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
 if($service.StartName -ne 'NT AUTHORITY\LocalService' -or (& "$app/immichclone.exe" -mode version) -ne '0.1.38-prototype'){throw 'Installed account/version mismatch'}
 $env:IMMICH_IMPORT_VOLUME_ROOT=$root;$env:IMMICH_IMPORT_VOLUME_STATE=$state
 $probe=Start-Process (Get-Command node).Source -ArgumentList @('tests/import-volume.cjs','start') -PassThru -RedirectStandardOutput "$root/probe.out" -RedirectStandardError "$root/probe.err"
 $until=[DateTime]::UtcNow.AddSeconds(90)
 while(!(Test-Path "$root/detach-ready") -and !$probe.HasExited -and [DateTime]::UtcNow -lt $until){Start-Sleep -Milliseconds 100}
 if(!(Test-Path "$root/detach-ready")){Get-Content "$root/probe.err";throw 'Partial-upload detach barrier not reached'}
 DiskCommands @("select vdisk file=`"$disk`"",'detach vdisk')
 if(Test-Path ($letter+':\')){throw 'Volume did not actually detach'}
 if(!$probe.WaitForExit(45000)){throw 'Browser interruption recovery timed out'}
 Get-Content "$root/probe.out"
 if($probe.ExitCode -ne 0){Get-Content "$root/probe.err";throw 'Browser failed volume interruption phase'}
 $service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
 if($service.ProcessId){Stop-Process -Id $service.ProcessId -Force -ErrorAction SilentlyContinue}
 $until=[DateTime]::UtcNow.AddSeconds(15)
 while((Get-Service ImmichClone).Status -ne 'Stopped' -and [DateTime]::UtcNow -lt $until){Start-Sleep -Milliseconds 100}
 # Startup while the entire catalog/state volume is absent must fail, not create a replacement library.
 try{Start-Service ImmichClone -ErrorAction Stop}catch{}
 Start-Sleep -Seconds 2
 if((Get-Service ImmichClone).Status -eq 'Running' -or (Get-NetTCPConnection -LocalPort 2283 -State Listen -ErrorAction SilentlyContinue)){throw 'Absent-volume startup unexpectedly exposed a library'}
 if(Test-Path ($letter+':\')){throw 'Absent volume unexpectedly reappeared'}
 DiskCommands @("select vdisk file=`"$disk`"",'attach vdisk')
 if(!(Test-Path "$state/catalog.db")){throw 'Original state volume did not reconnect'}
 Checked "$app/immichclone.exe" @('-mode','verify','-data',$state)
 Start-Service ImmichClone
 Checked node @('tests/import-volume.cjs','resume')
 Stop-Service ImmichClone
 Checked "$app/immichclone.exe" @('-mode','verify','-data',$state)
 Write-Output 'PASS: actual installed LocalService mounted NTFS state-volume detach, absent-volume startup refusal, reconnect and browser resume; not physical hardware or Windows10 proof'
}finally{
 if($probe -and !$probe.HasExited){Stop-Process -Id $probe.Id -Force;$null=$probe.WaitForExit(10000)}
 if(Test-Path $disk){try{DiskCommands @("select vdisk file=`"$disk`"",'attach vdisk')}catch{}}
 if(Test-Path "$app/unins000.exe"){$null=Start-Process "$app/unins000.exe" -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru}
 $service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'" -ErrorAction SilentlyContinue
 if($service -and $service.PathName.StartsWith(('"'+$app+'\immichclone.exe"'),[StringComparison]::OrdinalIgnoreCase)){Stop-Service ImmichClone -ErrorAction SilentlyContinue;sc.exe delete ImmichClone}
 if(Test-Path $disk){try{DiskCommands @("select vdisk file=`"$disk`"",'detach vdisk')}catch{}}
 foreach($key in @('IMMICH_IMPORT_VOLUME_ROOT','IMMICH_IMPORT_VOLUME_STATE')){Remove-Item "Env:$key" -ErrorAction SilentlyContinue}
 Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
