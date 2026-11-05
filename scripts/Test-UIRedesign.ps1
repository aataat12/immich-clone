param([string]$Version='0.1.41-prototype')
$ErrorActionPreference='Stop'
function Checked([string]$Exe,[string[]]$Arguments){ & $Exe @Arguments; if($LASTEXITCODE -ne 0){throw 'UI fixture operation failed'} }
function Assert([bool]$Value,[string]$Label){if(!$Value){throw $Label};Write-Host ('PASS: '+$Label)}
function Wait-Health{
 for($i=0;$i -lt 80;$i++){try{if((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok'){return}}catch{};Start-Sleep -Milliseconds 250}
 throw 'UI fixture service health timed out'
}
function Remove-FixtureService{
 if(Get-Service ImmichClone -ErrorAction SilentlyContinue){
  Stop-Service ImmichClone -Force -ErrorAction Stop
  Checked sc.exe @('delete','ImmichClone')
  for($i=0;$i -lt 80;$i++){if(!(Get-CimInstance Win32_Service -Filter "Name='ImmichClone'")){return};Start-Sleep -Milliseconds 100}
  throw 'UI fixture service was not removed'
 }
}
$root=Join-Path $env:RUNNER_TEMP ('ui-redesign-'+[guid]::NewGuid().ToString('N'))
$owned=$false
$oldNavAttach=$env:IMMICH_NAV_ATTACH;$oldPeopleAttach=$env:IMMICH_PEOPLE_ATTACH;$oldState=$env:IMMICH_TEST_STATE
try{
 Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue)) 'UI gate refuses an existing service'
 Assert (![string]::IsNullOrWhiteSpace($env:IMMICH_BROWSER_MODULE)) 'pinned browser module supplied'
 New-Item -ItemType Directory -Path $root|Out-Null
 $binary=Join-Path $root 'immichclone.exe'
 Checked go @('build','-trimpath','-ldflags',("-X main.version=$Version"),'-o',$binary,'./cmd/immichclone')
 Checked icacls.exe @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
 # Each standalone browser regression gets its own real LocalService library.
 # It never launches the now-refused elevated direct serve mode.
 foreach($kind in @('hidden','people')){
  $state=Join-Path $root ($kind+'-state');$logs=Join-Path $root ($kind+'-logs')
  New-Item -ItemType Directory -Path $state,$logs|Out-Null
  foreach($path in @($state,$logs)){Checked icacls.exe @($path,'/grant','*S-1-5-19:(OI)(CI)M')}
  $credential=[PSCredential]::new('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
  New-Service -Name ImmichClone -BinaryPathName ('"'+$binary+'" -mode service -data "'+$state+'" -logs "'+$logs+'"') -Credential $credential|Out-Null
  $owned=$true;Start-Service ImmichClone;Wait-Health
  $env:IMMICH_TEST_STATE=$state
  if($kind -eq 'hidden'){
   $env:IMMICH_NAV_ATTACH='1'
   Checked node @('tests/hidden-navigation.cjs')
   $env:IMMICH_NAV_ATTACH=$oldNavAttach
  }else{
   $env:IMMICH_PEOPLE_ATTACH='1'
   Checked node @('tests/people-index.cjs')
   $env:IMMICH_PEOPLE_ATTACH=$oldPeopleAttach
  }
  Remove-FixtureService;$owned=$false
 }
 $env:IMMICH_TEST_STATE=$oldState
 # Uses real pinned providers and original service/browser/recovery assertions.
 ./scripts/Test-Application.ps1 -Version $Version -UIReview
 if($LASTEXITCODE -ne 0){throw 'UI full application fixture failed'}
 Write-Host 'PASS: existing-function redesign compiled and exercised against Windows LocalService and real browsers'
}
finally{
 $env:IMMICH_NAV_ATTACH=$oldNavAttach;$env:IMMICH_PEOPLE_ATTACH=$oldPeopleAttach;$env:IMMICH_TEST_STATE=$oldState
 if($owned){Remove-FixtureService}
 if(Test-Path -LiteralPath $root){
  $resolved=[IO.Path]::GetFullPath($root);$parent=[IO.Path]::GetFullPath($env:RUNNER_TEMP).TrimEnd('\')+'\'
  if(!$resolved.StartsWith($parent,[StringComparison]::OrdinalIgnoreCase)){throw 'UI fixture cleanup target refused'}
  Remove-Item -LiteralPath $resolved -Recurse -Force
 }
}
