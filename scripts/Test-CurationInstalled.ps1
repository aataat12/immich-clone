$ErrorActionPreference='Stop'
$PSNativeCommandUseErrorActionPreference=$false
if($env:GITHUB_ACTIONS -ne 'true' -or (Get-Service ImmichClone -ErrorAction SilentlyContinue) -or (Test-Path 'HKLM:\SOFTWARE\ImmichClone')){throw 'Installed metadata acceptance requires a clean disposable CI host'}
$fixture='ImmichClone-curation-'+[guid]::NewGuid().ToString('N')
$root=Join-Path $env:RUNNER_TEMP $fixture
$cleanup=@(@{Path=$root;Parent=$env:RUNNER_TEMP;Leaf=$fixture})
$app=$null;$attempted=$false
function Checked([string]$Exe,[string[]]$Arguments){& $Exe @Arguments;if($LASTEXITCODE -ne 0){throw 'Metadata fixture operation failed'}}
function Assert([bool]$Ok,[string]$Message){if(!$Ok){throw $Message};Write-Host ('PASS: '+$Message)}
function Login([string]$State){$s=[Microsoft.PowerShell.Commands.WebRequestSession]::new();$code=(Get-Content -LiteralPath (Join-Path $State 'access-code.txt') -Raw).Trim();$null=Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -ContentType 'application/json' -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $s;return $s}
function Edit($Session,$InputObject){return Invoke-RestMethod http://127.0.0.1:2283/api/metadata/edit -Method Post -ContentType 'application/json; charset=utf-8' -Headers @{'X-Immich-Request'='1'} -Body ($InputObject|ConvertTo-Json -Depth 12) -WebSession $Session}
function Export($Session){return Invoke-RestMethod http://127.0.0.1:2283/api/export -WebSession $Session}
function WaitVersion([string]$Version){for($i=0;$i -lt 120;$i++){try{if((Invoke-RestMethod http://127.0.0.1:2283/health -TimeoutSec 1).build -eq $Version){return $true}}catch{};Start-Sleep -Milliseconds 250};return $false}
function Fixture([string]$State,[string]$Stage){$env:IMMICH_CURATION_ROOT=$State;$env:IMMICH_CURATION_STAGE=$Stage;Checked go @('test','-mod=readonly','-count=1','-timeout=120s','./internal/library','-run','^TestCurationPublishedFixture$');Remove-Item Env:IMMICH_CURATION_ROOT,Env:IMMICH_CURATION_STAGE -ErrorAction SilentlyContinue}
function Snapshot([string]$State){return ((Get-ChildItem -LiteralPath $State -Recurse -Force|ForEach-Object{if($_.PSIsContainer){'D|'+[IO.Path]::GetRelativePath($State,$_.FullName)}else{'F|'+[IO.Path]::GetRelativePath($State,$_.FullName)+'|'+(Get-FileHash -LiteralPath $_.FullName).Hash}}|Sort-Object)-join "`n")}
function StartSetup([string]$Installer,[string]$State){return Start-Process -FilePath $Installer -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$app+'"'),('/STATE="'+$State+'"'),('/LOG="'+(Join-Path $root 'setup.log')+'"')) -PassThru -WindowStyle Hidden}
function Install([string]$Installer,[string]$State){$p=StartSetup $Installer $State;if(!$p.WaitForExit(180000)){throw 'Setup fixture timed out'};$p.Refresh();return $p.ExitCode}
function Uninstall {if($app -and (Test-Path -LiteralPath (Join-Path $app 'unins000.exe'))){$p=Start-Process -FilePath (Join-Path $app 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -PassThru -Wait -WindowStyle Hidden;Assert ($p.ExitCode -eq 0) 'owned fixture uninstall succeeds'};Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue)) 'owned fixture service removed'}
function SeedMetadata($Session){
 $assets=@((Export $Session).assets);$asset=$assets[0];$m=Invoke-RestMethod ('http://127.0.0.1:2283/api/metadata/asset/'+$asset.id) -WebSession $Session
 $null=Edit $Session @{action='set_asset';asset_id=$asset.id;added=$asset.added;expected=$m.token;favorite=$true;title='Family <trip>';caption="Reunion`nRésumé";notes='Synthetic recovery note'}
 $albumID=('a'*64);$emptyID=('b'*64)
 $a=Edit $Session @{action='create_album';album_id=$albumID;title='Family album';description='Keep ordering and cover'}
 $null=Edit $Session @{action='create_album';album_id=$emptyID;title='Empty album'}
 foreach($item in $assets){$a=Edit $Session @{action='add_member';album_id=$albumID;asset_id=$item.id;added=$item.added;expected=$a.token}}
 $a=Edit $Session @{action='move_member';album_id=$albumID;asset_id=$assets[-1].id;other_id=$assets[0].id;expected=$a.token}
 $null=Edit $Session @{action='set_cover';album_id=$albumID;asset_id=$assets[0].id;expected=$a.token}
 $out=Export $Session;Assert ($out.catalog_schema -eq '3' -and @($out.curation.albums_v3.PSObject.Properties).Count -eq 2) 'ordinary authenticated writes commit complete catalog-three organization'
 return $out
}
try{
 New-Item -ItemType Directory -Path $root|Out-Null
 $sid=[Security.Principal.WindowsIdentity]::GetCurrent().User.Value
 Checked icacls.exe @($root,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F',('*'+$sid+':(OI)(CI)F'),'*S-1-5-19:(OI)(CI)RX')
 $oldInstaller=Join-Path $root 'published-0.1.43.exe'
 Invoke-WebRequest https://github.com/aataat12/immich-clone/releases/download/v0.1.43-prototype/ImmichClone-0.1.43-prototype-x64-setup.exe -OutFile $oldInstaller
 Assert ((Get-FileHash -LiteralPath $oldInstaller).Hash.ToLowerInvariant() -eq '96670e90c4abc55cdc07f480267c6f896b784a2a6c11de9f9f5333edb4158415') 'genuine published 0.1.43 installer checksum matches'
 $candidate=(Resolve-Path dist/ImmichClone-0.1.44-prototype-x64-setup.exe).Path
 foreach($schema in @('1','2')){
  $leaf=$fixture+'-s'+$schema;$app=Join-Path $env:ProgramFiles $leaf;$state=Join-Path $env:ProgramData $leaf;$offline=$state+'.offline';$restored=$state+'.restored'
  foreach($entry in @(@{Path=$app;Parent=$env:ProgramFiles;Leaf=$leaf},@{Path=$state;Parent=$env:ProgramData;Leaf=$leaf},@{Path=$offline;Parent=$env:ProgramData;Leaf=($leaf+'.offline')},@{Path=$restored;Parent=$env:ProgramData;Leaf=($leaf+'.restored')})){if(Test-Path -LiteralPath $entry.Path){throw 'Fixture paths must be new'};$cleanup+=$entry}
  $attempted=$true;Assert ((Install $oldInstaller $state) -eq 0) 'genuine old release installs normally'
  Stop-Service ImmichClone;Fixture $state ('create'+$schema);Start-Service ImmichClone
  Assert (WaitVersion '0.1.43-prototype') ('genuine 0.1.43 writer positively opens starting catalog '+$schema)
  $session=Login $state;$before=Export $session;$beforeOrganization=$before.organization|ConvertTo-Json -Depth 40 -Compress
  Assert ($before.catalog_schema -eq $schema) 'starting catalog format is exact'
  $old=Join-Path $root ('genuine-old-'+$schema+'.exe');Copy-Item -LiteralPath (Join-Path $app 'immichclone.exe') -Destination $old
  $oldHash=(Get-FileHash -LiteralPath $old).Hash;$codeHash=(Get-FileHash -LiteralPath (Join-Path $state 'access-code.txt')).Hash
  $originals=@($before.assets|ForEach-Object{$_.id})
  foreach($commit in @($false,$true)){
   Stop-Service ImmichClone;$catalogHash=(Get-FileHash -LiteralPath (Join-Path $state 'catalog.db')).Hash
   $env:IMMICH_INSTALL_TEST_FAILURE='1';$env:IMMICH_CURATION_INSTALL_TEST='1'
   try{$p=StartSetup $candidate $state}finally{Remove-Item Env:IMMICH_INSTALL_TEST_FAILURE,Env:IMMICH_CURATION_INSTALL_TEST -ErrorAction SilentlyContinue}
   Assert (WaitVersion '0.1.44-prototype') 'actual candidate Setup reaches the candidate service before injected failure'
   $newOriginal=$null
   if($commit){$session=Login $state;$null=SeedMetadata $session;$new=Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=after-snapshot.txt' -Method Post -Headers @{'X-Immich-Request'='1'} -ContentType application/octet-stream -Body ([Text.Encoding]::UTF8.GetBytes('original retained after metadata snapshot '+$schema)) -WebSession $session;$newOriginal=$new.asset.id;Assert ((Export $session).catalog_schema -eq '3') 'version-three metadata commit is observed before failure'}
   $exact=$false
   for($i=0;$i -lt 1800 -and !$p.HasExited;$i++){
    try{$service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'";if($service.State -eq 'Stopped' -and (Get-FileHash -LiteralPath (Join-Path $app 'immichclone.exe')).Hash -eq $oldHash -and (Get-FileHash -LiteralPath (Join-Path $state 'catalog.db')).Hash -eq $catalogHash){$exact=$true}}catch{}
    Start-Sleep -Milliseconds 100;$p.Refresh()
   }
   if(!$p.HasExited){throw 'Injected Setup recovery timed out'};$p.Refresh()
   Assert ($p.ExitCode -ne 0 -and $exact) 'real failed Setup restores exact saved catalog and genuine prior executable before service restart'
   Assert (WaitVersion '0.1.43-prototype') 'genuine prior service is healthy after rollback'
   $session=Login $state;$rolled=Export $session
   Assert ($rolled.catalog_schema -eq $schema -and ($rolled.organization|ConvertTo-Json -Depth 40 -Compress) -ceq $beforeOrganization) 'rollback preserves exact starting schema and face assignment/correction provenance'
   Assert ((Get-FileHash -LiteralPath (Join-Path $state 'access-code.txt')).Hash -eq $codeHash) 'rollback preserves the saved access code'
   foreach($id in $originals){Assert ((Get-FileHash -LiteralPath (Join-Path $state ('originals/'+$id))).Hash.ToLowerInvariant() -eq $id) 'existing original hash remains unchanged'}
   if($newOriginal){Assert ((Get-FileHash -LiteralPath (Join-Path $state ('originals/'+$newOriginal))).Hash.ToLowerInvariant() -eq $newOriginal) 'new original imported after snapshot survives catalog rollback'}
  }
  Assert ((Install $candidate $state) -eq 0) 'actual candidate Setup successfully upgrades the genuine release'
  Assert (WaitVersion '0.1.44-prototype') 'candidate service is healthy after successful upgrade'
  $candidatePath=(Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").PathName
  $session=Login $state;$out=SeedMetadata $session;$expected=Join-Path $root ('expected-'+$schema+'.json');$out.curation|ConvertTo-Json -Depth 40|Set-Content -Encoding utf8NoBOM -LiteralPath $expected
  $archive=Join-Path $root ('backup-'+$schema+'.zip');Invoke-WebRequest http://127.0.0.1:2283/api/backup -WebSession $session -OutFile $archive
  $recovery=Join-Path $root ('recovery-'+$schema+'.exe');Copy-Item -LiteralPath (Join-Path $app 'immichclone.exe') -Destination $recovery
  Stop-Service ImmichClone;$snapshot=Snapshot $state
  foreach($mode in @('verify','backup')){
   $rejected=Join-Path $root ('rejected-'+$schema+'.zip');$oldArgs=@('-mode',$mode,'-data',$state);if($mode -eq 'backup'){$oldArgs+=@('-archive',$rejected)}
   & $old @oldArgs 2>&1|Out-Null
   Assert ($LASTEXITCODE -ne 0 -and (Snapshot $state) -ceq $snapshot -and !(Test-Path -LiteralPath $rejected)) ('genuine old read-only '+$mode+' refuses version three without source or output mutation')
  }
  $oldLogs=Join-Path $root ('old-refusal-logs-'+$schema);New-Item -ItemType Directory -Path $oldLogs|Out-Null;Checked icacls.exe @($oldLogs,'/grant','*S-1-5-19:(OI)(CI)M')
  Checked sc.exe @('config','ImmichClone','binPath=',('"'+$old+'" -mode service -data "'+$state+'" -logs "'+$oldLogs+'"'))
  try{Start-Service ImmichClone -ErrorAction Stop}catch{}
  for($i=0;$i -lt 80;$i++){if((Get-Service ImmichClone).Status -eq 'Stopped'){break};Start-Sleep -Milliseconds 100}
  Assert ((Get-Service ImmichClone).Status -eq 'Stopped' -and (Snapshot $state) -ceq $snapshot) 'genuine old LocalService writer refuses version three and preserves every source byte and name'
  Checked sc.exe @('config','ImmichClone','binPath=',$candidatePath);Start-Service ImmichClone;Assert (WaitVersion '0.1.44-prototype') 'candidate reopens metadata after genuine old-writer refusal'
  Uninstall
  if([IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($state)) -cne [IO.Path]::GetFullPath($env:ProgramData) -or [IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($offline)) -cne [IO.Path]::GetFullPath($env:ProgramData)){throw 'Unavailable-source move target refused'}
  Move-Item -LiteralPath $state -Destination $offline
  Checked $recovery @('-mode','restore','-data',$restored,'-archive',$archive)
  Checked $recovery @('-mode','verify','-data',$restored)
  $env:IMMICH_CURATION_EXPECTED=$expected;Fixture $restored 'inspect';Remove-Item Env:IMMICH_CURATION_EXPECTED -ErrorAction SilentlyContinue
  Assert ((Install $candidate $restored) -eq 0) 'actual Setup enrolls independently restored catalog-three state with original state unavailable'
  $session=Login $restored;$after=Export $session
  Assert (($after.curation|ConvertTo-Json -Depth 40 -Compress) -ceq ($out.curation|ConvertTo-Json -Depth 40 -Compress)) 'installed independent restore preserves complete favorite/text/albums/cover/order/empty album graph'
  Assert (($after.organization|ConvertTo-Json -Depth 40 -Compress) -ceq $beforeOrganization) 'independent restore preserves automatic/manual face provenance'
  Uninstall;$attempted=$false
 }
}finally{
 Remove-Item Env:IMMICH_INSTALL_TEST_FAILURE,Env:IMMICH_CURATION_INSTALL_TEST,Env:IMMICH_CURATION_ROOT,Env:IMMICH_CURATION_STAGE,Env:IMMICH_CURATION_EXPECTED -ErrorAction SilentlyContinue
 if($attempted -and $app -and (Test-Path -LiteralPath (Join-Path $app 'unins000.exe'))){Stop-Service ImmichClone -Force -ErrorAction SilentlyContinue;Uninstall}
 foreach($entry in $cleanup){if(Test-Path -LiteralPath $entry.Path){$resolved=[IO.Path]::GetFullPath($entry.Path);if([IO.Path]::GetDirectoryName($resolved) -cne [IO.Path]::GetFullPath($entry.Parent).TrimEnd('\') -or [IO.Path]::GetFileName($resolved) -cne $entry.Leaf){throw 'Fixture cleanup target refused'};Remove-Item -LiteralPath $resolved -Recurse -Force}}
}
