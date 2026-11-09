$ErrorActionPreference='Stop'
function Checked([string]$Exe,[string[]]$Arguments){ & $Exe @Arguments; if($LASTEXITCODE -ne 0){throw 'Feature fixture operation failed'} }
function WaitExperienceService {
 for($i=0;$i -lt 80;$i++){
  try{if((Invoke-RestMethod http://127.0.0.1:2283/health -TimeoutSec 1).status -eq 'ok'){return}}catch{}
  Start-Sleep -Milliseconds 250
 }
 throw 'Integrated browser fixture service health timed out'
}
$root=Join-Path $env:RUNNER_TEMP ('gallery44-foundation-'+[guid]::NewGuid().ToString('N'))
$owned=$false
$experience=$env:IMMICH_TEST_EXPERIENCE -eq '1'
try {
 if($experience -and $env:IMMICH_TEST_CURATION -ne '1'){throw 'Integrated experience requires the independent restore gate'}
 if(Get-Service ImmichClone -ErrorAction SilentlyContinue){throw 'Disposable feature gate refuses an existing service'}
 New-Item -ItemType Directory -Path $root|Out-Null
 $binary=Join-Path $root 'immichclone.exe';$state=Join-Path $root 'state';$logs=Join-Path $root 'logs'
 New-Item -ItemType Directory -Path $state,$logs|Out-Null
 Checked go @('build','-mod=readonly','-trimpath','-ldflags','-X main.version=0.1.44-foundation','-o',$binary,'./cmd/immichclone')
 Checked icacls.exe @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
 foreach($folder in @($state,$logs)){Checked icacls.exe @($folder,'/grant','*S-1-5-19:(OI)(CI)M')}
 $credential=[PSCredential]::new('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
 New-Service -Name ImmichClone -BinaryPathName ('"'+$binary+'" -mode service -data "'+$state+'" -logs "'+$logs+'"') -Credential $credential|Out-Null
 $owned=$true;Start-Service ImmichClone
 $ready=$false
 for($i=0;$i -lt 80;$i++){try{if((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok'){$ready=$true;break}}catch{};Start-Sleep -Milliseconds 250}
 if(!$ready){throw 'Feature fixture service health timed out'}
 $env:IMMICH_TEST_STATE=$state
 Checked node @('tests/gallery-search.cjs')
 Checked node @('tests/gallery-foundation.cjs')
 Checked node @('tests/gallery-viewer.cjs')
 Checked node @('tests/photo-comparison.cjs')
 Checked node @('tests/suggested-albums.cjs')
 if($env:IMMICH_TEST_ORGANIZATION -eq '1'){
  $env:IMMICH_ORG_STAGE='prepare'
  Checked node @('tests/organization56.cjs')
  Stop-Service ImmichClone
  Start-Service ImmichClone
  $ready=$false
  for($i=0;$i -lt 80;$i++){try{if((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok'){$ready=$true;break}}catch{};Start-Sleep -Milliseconds 250}
  if(!$ready){throw 'Organization restart fixture service health timed out'}
  $env:IMMICH_ORG_STAGE='recover'
  Checked node @('tests/organization56.cjs')
 }
 if($env:IMMICH_TEST_BROWSE78 -eq '1'){
  $env:IMMICH_BROWSE78_STAGE='prepare'
  Checked node @('tests/browse78.cjs')
  Stop-Service ImmichClone
  Start-Service ImmichClone
  $ready=$false
  for($i=0;$i -lt 80;$i++){try{if((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok'){$ready=$true;break}}catch{};Start-Sleep -Milliseconds 250}
  if(!$ready){throw 'Source browsing restart fixture service health timed out'}
  $env:IMMICH_BROWSE78_STAGE='verify'
  Checked node @('tests/browse78.cjs')
 }
 if($experience){
  $env:IMMICH_EXPERIENCE_FIXTURE=Join-Path $root 'expected-experience44.json'
  $env:IMMICH_EXPERIENCE_STAGE='prepare'
  Checked node @('tests/release44-experience.cjs')
  Restart-Service ImmichClone
  WaitExperienceService
  $env:IMMICH_EXPERIENCE_STAGE='recover'
  Checked node @('tests/release44-experience.cjs')
 }
 if($env:IMMICH_TEST_CURATION -eq '1'){
  Checked node @('tests/metadata-recovery.cjs')
  Stop-Service ImmichClone
  Checked sc.exe @('delete','ImmichClone')
  $owned=$false
  $offline=$state+'.offline';$restore=Join-Path $root 'restored'
  if([IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($offline)) -cne [IO.Path]::GetFullPath($root)){throw 'Fixture move target refused'}
  Move-Item -LiteralPath $state -Destination $offline
  Checked $binary @('-mode','restore','-data',$restore,'-archive',(Join-Path $root 'metadata-backup.zip'))
  Checked $binary @('-mode','verify','-data',$restore)
  $env:IMMICH_CURATION_ROOT=$restore;$env:IMMICH_CURATION_STAGE='inspect';$env:IMMICH_CURATION_EXPECTED=Join-Path $root 'expected-curation.json'
  if($env:IMMICH_TEST_ORGANIZATION -eq '1'){$env:IMMICH_EXPECT_PAUSED_BATCH='1'}
  Checked go @('test','-mod=readonly','-count=1','-timeout=120s','./internal/library','-run','^TestCurationPublishedFixture$')
  if($env:IMMICH_TEST_BROWSE78 -eq '1'){
   $env:IMMICH_BROWSE78_ROOT=$restore;$env:IMMICH_BROWSE78_EXPECTED=Join-Path $root 'expected-browse78.json'
   Checked go @('test','-mod=readonly','-count=1','-timeout=120s','./internal/library','-run','^TestGalleryBrowse78RestoredFixture$')
   Remove-Item Env:IMMICH_BROWSE78_ROOT,Env:IMMICH_BROWSE78_EXPECTED -ErrorAction SilentlyContinue
  }
  if($experience){
   for($i=0;$i -lt 80;$i++){
    if(!(Get-CimInstance Win32_Service -Filter "Name='ImmichClone'")){break}
    Start-Sleep -Milliseconds 100
   }
   Checked icacls.exe @($restore,'/grant','*S-1-5-19:(OI)(CI)M')
   New-Service -Name ImmichClone -BinaryPathName ('"'+$binary+'" -mode service -data "'+$restore+'" -logs "'+$logs+'"') -Credential $credential|Out-Null
   $owned=$true
   Start-Service ImmichClone
   WaitExperienceService
   $env:IMMICH_TEST_STATE=$restore
   $env:IMMICH_EXPERIENCE_STAGE='verify'
   Checked node @('tests/release44-experience.cjs')
  }
  Remove-Item Env:IMMICH_EXPECT_PAUSED_BATCH -ErrorAction SilentlyContinue
  Remove-Item Env:IMMICH_CURATION_ROOT,Env:IMMICH_CURATION_STAGE,Env:IMMICH_CURATION_EXPECTED -ErrorAction SilentlyContinue
 }
} finally {
 if($experience){Remove-Item Env:IMMICH_EXPERIENCE_FIXTURE,Env:IMMICH_EXPERIENCE_STAGE -ErrorAction SilentlyContinue}
 if($owned){
  Stop-Service ImmichClone -Force -ErrorAction Stop
  Checked sc.exe @('delete','ImmichClone')
  for($i=0;$i -lt 80;$i++){if(!(Get-CimInstance Win32_Service -Filter "Name='ImmichClone'")){break};Start-Sleep -Milliseconds 100}
 }
 if(Test-Path -LiteralPath $root){
  $resolved=[IO.Path]::GetFullPath($root);$parent=[IO.Path]::GetFullPath($env:RUNNER_TEMP).TrimEnd('\')+'\'
  if(!$resolved.StartsWith($parent,[StringComparison]::OrdinalIgnoreCase)){throw 'Feature fixture cleanup target refused'}
  Remove-Item -LiteralPath $resolved -Recurse -Force
 }
}
