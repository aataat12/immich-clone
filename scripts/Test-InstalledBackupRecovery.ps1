$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('installed-backup-recovery-'+[guid]::NewGuid().ToString('N'))
$app=Join-Path $root 'application';$disk=Join-Path $root 'disposable.vhd'
$letter=@('R','S','T','U','V')|Where-Object {!(Test-Path ($_+':\'))}|Select-Object -First 1
$probe=$null;$attached=$false
function Checked([string]$exe,[string[]]$argv){& $exe @argv;if($LASTEXITCODE -ne 0){throw 'Installed backup recovery command failed'}}
function DiskCommands([string[]]$lines){$file=Join-Path $root 'diskpart.txt';$lines|Set-Content $file;Checked diskpart.exe @('/s',$file)}
function Stop-OwnedService {
 $s=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
 if(!$s -or !$s.PathName.StartsWith(('"'+$app+'\immichclone.exe"'),[StringComparison]::OrdinalIgnoreCase)){throw 'Refusing an unowned service'}
 if($s.ProcessId){Stop-Process -Id $s.ProcessId -Force -ErrorAction SilentlyContinue}
 $end=[DateTime]::UtcNow.AddSeconds(20)
 while((Get-Service ImmichClone).Status -ne 'Stopped' -and [DateTime]::UtcNow -lt $end){Start-Sleep -Milliseconds 100}
 if((Get-Service ImmichClone).Status -ne 'Stopped'){throw 'Owned service did not stop'}
}
function Browser-Barrier([string]$phase,[string]$marker,[int]$seconds){
 $script:probe=Start-Process (Get-Command node).Source -ArgumentList @('tests/installed-backup-recovery.cjs',$phase) -PassThru -RedirectStandardOutput "$root/$phase.out" -RedirectStandardError "$root/$phase.err"
 $end=[DateTime]::UtcNow.AddSeconds($seconds)
 while(!(Test-Path "$root/$marker") -and !$probe.HasExited -and [DateTime]::UtcNow -lt $end){Start-Sleep -Milliseconds 50}
 if(!(Test-Path "$root/$marker")){Get-Content "$root/$phase.out";Get-Content "$root/$phase.err";throw 'Actual backup-copy barrier not reached'}
}
function Finish-Browser([string]$phase){
 if(!$probe.WaitForExit(60000)){throw 'Interrupted browser phase did not terminate'}
 Get-Content "$root/$phase.out"
 if($probe.ExitCode -ne 0){Get-Content "$root/$phase.err";throw 'Interrupted browser phase failed'}
}
try{
 if(!$letter -or (Get-Service ImmichClone -ErrorAction SilentlyContinue) -or (Test-Path 'HKLM:\Software\ImmichClone')){throw 'Clean disposable runner and unused drive letter required'}
 if([IO.DriveInfo]::new([IO.Path]::GetPathRoot($root)).AvailableFreeSpace -lt 24GB){throw 'At least24GiB disposable free space required before large I/O'}
 New-Item -ItemType Directory $root|Out-Null
 DiskCommands @("create vdisk file=`"$disk`" maximum=24576 type=expandable","select vdisk file=`"$disk`"",'attach vdisk','create partition primary','format fs=ntfs quick label=ImmichBackupProbe',"assign letter=$letter")
 $attached=$true;$volume=Get-Volume -DriveLetter $letter
 if($volume.FileSystem -ne 'NTFS' -or $volume.FileSystemLabel -ne 'ImmichBackupProbe' -or $volume.Size -gt 24577MB){throw 'Unexpected disposable backup volume'}
 $state=$letter+':\state'
 $setup=(Resolve-Path 'accepted-installer/ImmichClone-0.1.38-prototype-x64-setup.exe').Path
 if((Get-FileHash $setup).Hash.ToLowerInvariant() -ne '14e1f0dd6b5bc709d45235963e137e47ee5740fee13ec3c8c03d4756ddd24038'){throw 'Accepted installer hash mismatch'}
 $p=Start-Process $setup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$app+'"'),('/STATE="'+$state+'"')) -Wait -PassThru
 if($p.ExitCode -ne 0){throw 'Accepted installation failed'}
 $s=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
 if($s.StartName -ne 'NT AUTHORITY\LocalService' -or (& "$app/immichclone.exe" -mode version) -ne '0.1.38-prototype'){throw 'Installed account/version mismatch'}
 $env:IMMICH_METADATA_FIXTURE="$root/fixture.jpg"
 Checked go @('test','-mod=readonly','-count=1','./internal/library','-run','^TestMetadataFixtureEmitter$')
 Remove-Item Env:IMMICH_METADATA_FIXTURE
 $env:IMMICH_BACKUP_RECOVERY_ROOT=$root;$env:IMMICH_BACKUP_RECOVERY_STATE=$state
 Browser-Barrier 'seed-crash' 'crash-ready' 300
 Stop-OwnedService
 Finish-Browser 'seed-crash'
 Start-Service ImmichClone
 Checked node @('tests/installed-backup-recovery.cjs','recover-first')
 Restart-Service ImmichClone
 Browser-Barrier 'start-outage' 'outage-ready' 120
 DiskCommands @("select vdisk file=`"$disk`"",'detach vdisk');$attached=$false
 if(Test-Path ($letter+':\')){throw 'State volume did not actually detach'}
 Finish-Browser 'start-outage'
 Stop-OwnedService
 try{Start-Service ImmichClone -ErrorAction Stop}catch{}
 Start-Sleep -Seconds 2
 if((Get-Service ImmichClone).Status -eq 'Running' -or (Get-NetTCPConnection -LocalPort 2283 -State Listen -ErrorAction SilentlyContinue)){throw 'Absent-state startup unexpectedly exposed a library'}
 if(Test-Path ($letter+':\')){throw 'Absent state volume unexpectedly reappeared'}
 DiskCommands @("select vdisk file=`"$disk`"",'attach vdisk');$attached=$true
 if(!(Test-Path "$state/catalog.db")){throw 'Original catalog volume did not reconnect'}
 Checked "$app/immichclone.exe" @('-mode','verify','-data',$state)
 Start-Service ImmichClone
 Checked node @('tests/installed-backup-recovery.cjs','recover-second')
 Restart-Service ImmichClone
 Checked node @('tests/installed-backup-recovery.cjs','finish')
 Stop-Service ImmichClone
 Checked "$app/immichclone.exe" @('-mode','verify','-data',$state)
 DiskCommands @("select vdisk file=`"$disk`"",'detach vdisk');$attached=$false
 if(Test-Path ($letter+':\')){throw 'Original state must be unavailable for independent restoration'}
 Checked "$app/immichclone.exe" @('-mode','restore','-archive',"$root/download-second.zip",'-data',"$root/restored")
 Checked "$app/immichclone.exe" @('-mode','verify','-data',"$root/restored")
 # The inspector reads restored originals and metadata, never the original VHD.
 $env:IMMICH_BACKUP_RECOVERY_RESTORED="$root/restored"
 Checked go @('test','-mod=readonly','-v','-count=1','-timeout=5m','./internal/library','-run','^TestInstalledBackupRecoveryMetadata$')
 Write-Output 'PASS: installed >4GiB backup cycles, active-copy service crash, real mounted-volume outage, cold recovery, independent default-budget restore and hashes/metadata; not hardware-removal or Windows10 proof'
}finally{
 if($probe -and !$probe.HasExited){Stop-Process -Id $probe.Id -Force;$null=$probe.WaitForExit(10000)}
 if((Test-Path $disk) -and !$attached){try{DiskCommands @("select vdisk file=`"$disk`"",'attach vdisk');$attached=$true}catch{}}
 if(Test-Path "$app/unins000.exe"){$null=Start-Process "$app/unins000.exe" -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru}
 $s=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'" -ErrorAction SilentlyContinue
 if($s -and $s.PathName.StartsWith(('"'+$app+'\immichclone.exe"'),[StringComparison]::OrdinalIgnoreCase)){Stop-Service ImmichClone -ErrorAction SilentlyContinue;sc.exe delete ImmichClone}
 if((Test-Path $disk) -and $attached){try{DiskCommands @("select vdisk file=`"$disk`"",'detach vdisk')}catch{}}
 foreach($key in @('IMMICH_METADATA_FIXTURE','IMMICH_BACKUP_RECOVERY_ROOT','IMMICH_BACKUP_RECOVERY_STATE','IMMICH_BACKUP_RECOVERY_RESTORED')){Remove-Item "Env:$key" -ErrorAction SilentlyContinue}
 Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
