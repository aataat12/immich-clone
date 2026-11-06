$ErrorActionPreference='Stop'
$PSNativeCommandUseErrorActionPreference=$false
if($env:GITHUB_ACTIONS -ne 'true' -or (Get-Service ImmichClone -ErrorAction SilentlyContinue) -or (Test-Path 'HKLM:\SOFTWARE\ImmichClone')){throw 'Schema-two Setup acceptance requires a clean disposable CI host'}
$fixtureName='ImmichClone-schema-install-'+[guid]::NewGuid().ToString('N')
$root=Join-Path $env:RUNNER_TEMP $fixtureName
$app=Join-Path $env:ProgramFiles $fixtureName
$initialState=Join-Path $env:ProgramData $fixtureName
$restoredState=Join-Path $env:ProgramData ($fixtureName+'-restored')
$offlineState=$initialState+'.offline'
$state=$initialState
$setupAttempted=$false
$priorFixtureRoot=$env:IMMICH_R43_SCHEMA_FIXTURE_ROOT
$priorFixtureStage=$env:IMMICH_R43_SCHEMA_FIXTURE_STAGE
$priorFailure=$env:IMMICH_INSTALL_TEST_FAILURE
foreach($path in @($root,$app,$initialState,$restoredState,$offlineState)){if(Test-Path -LiteralPath $path){throw 'Schema-two Setup fixture paths must be new'}}
foreach($version in @('0.1.43','0.1.44')){if(!(Test-Path -LiteralPath "dist/ImmichClone-$version-prototype-x64-setup.exe")){throw 'Both exact candidate and failed-upgrade test installers are required'}}
function Checked([string]$Exe,[string[]]$Arguments){& $Exe @Arguments;if($LASTEXITCODE -ne 0){throw 'Schema-two Setup fixture operation failed'}}
function Assert([bool]$Condition,[string]$Message){if(!$Condition){throw $Message};Write-Host ('PASS: '+$Message)}
function Fixture([string]$Stage){
 $env:IMMICH_R43_SCHEMA_FIXTURE_ROOT=$state
 $env:IMMICH_R43_SCHEMA_FIXTURE_STAGE=$Stage
 Checked go @('test','-mod=readonly','-count=1','-timeout=90s','./internal/library','-run','^TestAutomaticSchemaPublishedWriterFixture$')
}
function Install([string]$Version){
 $installer=(Resolve-Path "dist/ImmichClone-$Version-prototype-x64-setup.exe").Path
 $process=Start-Process $installer -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$app+'"'),('/STATE="'+$state+'"'),('/LOG="'+(Join-Path $root ('setup-'+$Version+'.log'))+'"')) -Wait -PassThru -WindowStyle Hidden
 return $process.ExitCode
}
function Uninstall {
 $uninstaller=Join-Path $app 'unins000.exe'
 if(!(Test-Path -LiteralPath $uninstaller)){throw 'Owned schema-two uninstaller is missing'}
 $process=Start-Process $uninstaller -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru -WindowStyle Hidden
 Assert ($process.ExitCode -eq 0) 'schema-two installed candidate uninstalls successfully'
 Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue) -and !(Test-Path 'HKLM:\SOFTWARE\ImmichClone')) 'schema-two uninstall removes only the owned service and enrollment'
}
function OriginalHashes([string]$Data){
 $originals=@(Get-ChildItem -LiteralPath (Join-Path $Data 'originals') -File)
 Assert ($originals.Count -eq 3) 'schema-two fixture retains all three original files'
 return (($originals|ForEach-Object {
  $hash=(Get-FileHash -LiteralPath $_.FullName).Hash.ToLowerInvariant()
  if($_.Name -cne $hash){throw 'Schema-two fixture original content hash differs from its immutable identifier'}
  $_.Name+'|'+$hash
 }|Sort-Object)-join "`n")
}
function Login([string]$Data){
 $session=[Microsoft.PowerShell.Commands.WebRequestSession]::new()
 $code=(Get-Content -LiteralPath (Join-Path $Data 'access-code.txt') -Raw).Trim()
 $null=Invoke-RestMethod 'http://127.0.0.1:2283/api/login' -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
 return $session
}
function CaptureState([string]$Data){
 $ready=$false
 for($i=0;$i -lt 60;$i++){
  try{if((Invoke-RestMethod 'http://127.0.0.1:2283/health' -TimeoutSec 2).build -eq '0.1.43-prototype'){$ready=$true;break}}catch{}
  Start-Sleep -Milliseconds 200
 }
 Assert $ready 'actual schema-two installed service is healthy with the exact candidate version'
 $service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
 Assert ($service.StartName -eq 'NT AUTHORITY\LocalService' -and $service.State -eq 'Running' -and $service.StartMode -eq 'Auto') 'schema-two library runs under the normal automatic LocalService registration'
 $session=Login $Data
 $catalog=Invoke-RestMethod 'http://127.0.0.1:2283/api/export' -WebSession $session
 Assert ($catalog.catalog_schema -eq '2') 'actual installed service preserves catalog version two'
 $labels=@($catalog.organization.face_labels_v1.PSObject.Properties|ForEach-Object Value)
 $automatic=@($labels|Where-Object {$_.name_source -eq 'automatic' -and $_.name -eq 'Schema person'})
 Assert ($labels.Count -eq 3 -and $automatic.Count -eq 1 -and $automatic[0].automatic_policy -eq 'known-person-two-sources-065-015-v1' -and $automatic[0].automatic_score -ge .65) 'actual installed service preserves automatic assignment provenance and policy'
 Assert ((Get-Content -LiteralPath (Join-Path $Data 'state-format.txt') -Raw) -ceq "immichclone-schema-1`n") 'catalog-two installation retains the independent version-one storage layout'
 return @{
  Labels=($catalog.organization.face_labels_v1|ConvertTo-Json -Depth 20 -Compress)
  Originals=(OriginalHashes $Data)
  CodeHash=(Get-FileHash -LiteralPath (Join-Path $Data 'access-code.txt')).Hash
  BinaryHash=(Get-FileHash -LiteralPath (Join-Path $app 'immichclone.exe')).Hash
 }
}
function AssertPreserved($Before,$After,[string]$Message){
 Assert ($Before.Labels -ceq $After.Labels -and $Before.Originals -ceq $After.Originals -and $Before.CodeHash -ceq $After.CodeHash -and $Before.BinaryHash -ceq $After.BinaryHash) $Message
}
try {
 New-Item -ItemType Directory -Path $root,$initialState|Out-Null
 $sid=[Security.Principal.WindowsIdentity]::GetCurrent().User.Value
 foreach($folder in @($root,$initialState)){
  Checked icacls.exe @($folder,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F',('*'+$sid+':(OI)(CI)F'))
 }
 Fixture 'create'
 Fixture 'promote'
 $originalsBefore=OriginalHashes $initialState
 $codeBefore=(Get-FileHash -LiteralPath (Join-Path $initialState 'access-code.txt')).Hash
 $setupAttempted=$true
 Assert ((Install '0.1.43') -eq 0) 'actual candidate Setup enrolls an existing synthetic schema-two library'
 $first=CaptureState $state
 Assert ($first.Originals -ceq $originalsBefore -and $first.CodeHash -ceq $codeBefore) 'initial schema-two Setup preserves the existing originals and access code'

 # Retain nontrivial manual undo history beside the automatic assignment. No
 # analysis is rerun and no synthetic face is treated as quality evidence.
 $session=Login $state
 $organization=Invoke-RestMethod 'http://127.0.0.1:2283/api/organization' -WebSession $session
 $manual=@($organization.faces|Where-Object {$_.name_source -eq 'manual'})[0]
 Assert (![string]::IsNullOrWhiteSpace($manual.id)) 'manual reference is available for the catalog-history fixture'
 $null=Invoke-RestMethod 'http://127.0.0.1:2283/api/faces/label' -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{ids=@($manual.id);name='Schema reference corrected'}|ConvertTo-Json) -WebSession $session
 $baseline=CaptureState $state
 Assert ($baseline.Labels.Contains('previous_source') -and $baseline.Labels.Contains('Schema reference corrected')) 'schema-two fixture includes explicit correction and undo history'
 $recoveryTool=Join-Path $root 'recovery-tool.exe'
 Copy-Item -LiteralPath (Join-Path $app 'immichclone.exe') -Destination $recoveryTool
 Assert ((Get-FileHash -LiteralPath $recoveryTool).Hash -ceq $baseline.BinaryHash) 'independent recovery tool matches the actual installed candidate'

 $env:IMMICH_INSTALL_TEST_FAILURE='1'
 try{$failed=Install '0.1.44'}finally{$env:IMMICH_INSTALL_TEST_FAILURE=$priorFailure}
 Assert ($failed -ne 0) 'injected schema-two upgrade failure is refused'
 $rolledBack=CaptureState $state
 AssertPreserved $baseline $rolledBack 'actual Setup rollback preserves schema-two labels/history, exact previous binary, access code and all original hashes'
 Assert ((& (Join-Path $app 'immichclone.exe') -mode version) -eq '0.1.43-prototype') 'schema-two rollback restores the actual previous executable version'

 $archive=Join-Path $root 'schema-two-http-backup.zip'
 $session=Login $state
 Invoke-WebRequest 'http://127.0.0.1:2283/api/backup' -WebSession $session -OutFile $archive
 Assert ((Get-Item -LiteralPath $archive).Length -gt 0) 'actual installed service creates the schema-two recovery archive'
 Uninstall
 Assert ((OriginalHashes $state) -ceq $baseline.Originals -and (Get-FileHash -LiteralPath (Join-Path $state 'access-code.txt')).Hash -ceq $baseline.CodeHash -and (Test-Path -LiteralPath (Join-Path $state 'catalog.db'))) 'schema-two uninstall retains catalog, originals and access code'
 if([IO.Path]::GetFullPath($state) -cne [IO.Path]::GetFullPath($initialState) -or [IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($offlineState)) -cne [IO.Path]::GetFullPath($env:ProgramData).TrimEnd('\') -or [IO.Path]::GetFileName($offlineState) -cne ($fixtureName+'.offline')){throw 'Schema-two unavailable-source move target refused'}
 Move-Item -LiteralPath $initialState -Destination $offlineState
 Assert (!(Test-Path -LiteralPath $initialState)) 'original schema-two state is unavailable at its original location'
 $state=$restoredState
 Checked $recoveryTool @('-mode','restore','-data',$state,'-archive',$archive)
 Fixture 'inspect'
 Assert ((Install '0.1.43') -eq 0) 'actual Setup reinstalls the independently restored schema-two library'
 $restored=CaptureState $state
 AssertPreserved $baseline $restored 'independent installed restore retains schema two, automatic policy, correction/undo history, original hashes and access code'
 Uninstall
 Assert ((OriginalHashes $state) -ceq $baseline.Originals -and (Get-FileHash -LiteralPath (Join-Path $state 'access-code.txt')).Hash -ceq $baseline.CodeHash -and (Test-Path -LiteralPath (Join-Path $state 'catalog.db'))) 'independent schema-two uninstall again preserves the restored library'
 Fixture 'inspect'
} finally {
 $env:IMMICH_INSTALL_TEST_FAILURE=$priorFailure
 $env:IMMICH_R43_SCHEMA_FIXTURE_ROOT=$priorFixtureRoot
 $env:IMMICH_R43_SCHEMA_FIXTURE_STAGE=$priorFixtureStage
 if($setupAttempted -and (Get-Service ImmichClone -ErrorAction SilentlyContinue)){
  Stop-Service ImmichClone -Force -ErrorAction SilentlyContinue
  if(Test-Path -LiteralPath (Join-Path $app 'unins000.exe')){Uninstall}else{throw 'Owned schema-two service remains without its uninstaller'}
 }
 foreach($item in @(
  @{Path=$root;Parent=$env:RUNNER_TEMP;Leaf=$fixtureName},
  @{Path=$app;Parent=$env:ProgramFiles;Leaf=$fixtureName},
  @{Path=($app+'-upgrade-recovery');Parent=$env:ProgramFiles;Leaf=($fixtureName+'-upgrade-recovery')},
  @{Path=$initialState;Parent=$env:ProgramData;Leaf=$fixtureName},
  @{Path=$offlineState;Parent=$env:ProgramData;Leaf=($fixtureName+'.offline')},
  @{Path=$restoredState;Parent=$env:ProgramData;Leaf=($fixtureName+'-restored')}
 )){
  if(Test-Path -LiteralPath $item.Path){
   $target=[IO.Path]::GetFullPath($item.Path)
   $parent=[IO.Path]::GetFullPath($item.Parent).TrimEnd('\')
   if([IO.Path]::GetDirectoryName($target) -cne $parent -or [IO.Path]::GetFileName($target) -cne $item.Leaf){throw 'Schema-two Setup cleanup target refused'}
   Remove-Item -LiteralPath $target -Recurse -Force
  }
 }
}
