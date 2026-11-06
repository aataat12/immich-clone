$ErrorActionPreference='Stop'
$PSNativeCommandUseErrorActionPreference=$false
if($env:GITHUB_ACTIONS -ne 'true' -or (Get-Service ImmichClone -ErrorAction SilentlyContinue) -or (Test-Path 'HKLM:\SOFTWARE\ImmichClone')){throw 'Schema compatibility requires a clean disposable CI host'}
$name='ImmichClone-schema-'+[guid]::NewGuid().ToString('N')
$root=Join-Path $env:RUNNER_TEMP $name
$app=Join-Path $env:ProgramFiles $name
$initialState=Join-Path $env:ProgramData $name
foreach($path in @($root,$app,$initialState)){if(Test-Path -LiteralPath $path){throw 'Schema fixture paths must be new'}}
$setupAttempted=$false;$initialServicePath=$null
function Checked([string]$Exe,[string[]]$Arguments){& $Exe @Arguments;if($LASTEXITCODE -ne 0){throw 'Schema fixture operation failed'}}
function Assert([bool]$Condition,[string]$Message){if(!$Condition){throw $Message};Write-Host ('PASS: '+$Message)}
function Fixture([string]$Stage){
 $env:IMMICH_R43_SCHEMA_FIXTURE_STAGE=$Stage
 Checked go @('test','-mod=readonly','-count=1','-timeout=90s','./internal/library','-run','^TestAutomaticSchemaPublishedWriterFixture$')
}
function StateSnapshot([string]$State){
 $lines=@(Get-ChildItem -LiteralPath $State -Recurse -Force|ForEach-Object {
  $relative=[IO.Path]::GetRelativePath($State,$_.FullName)
  if($_.PSIsContainer){'D|'+$relative}else{'F|'+$relative+'|'+$_.Length+'|'+(Get-FileHash -LiteralPath $_.FullName).Hash}
 })
 return (($lines|Sort-Object)-join "`n")
}
function ServiceCommand([string]$Binary,[string]$State,[string]$Logs){return ('"'+$Binary+'" -mode service -data "'+$State+'" -logs "'+$Logs+'"')}
function WaitHealthy([string]$Version){
 for($i=0;$i -lt 60;$i++){
  try{if((Invoke-RestMethod 'http://127.0.0.1:2283/health' -TimeoutSec 2).build -eq $Version){return $true}}catch{}
  Start-Sleep -Milliseconds 200
 }
 return $false
}
try {
 New-Item -ItemType Directory -Path $root|Out-Null
 $sid=[Security.Principal.WindowsIdentity]::GetCurrent().User.Value
 Checked icacls.exe @($root,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F',('*'+$sid+':(OI)(CI)F'),'*S-1-5-19:(OI)(CI)RX')
 $installer=Join-Path $root 'published-0.1.42-setup.exe'
 Invoke-WebRequest 'https://github.com/aataat12/immich-clone/releases/download/v0.1.42-prototype/ImmichClone-0.1.42-prototype-x64-setup.exe' -OutFile $installer
 Assert ((Get-FileHash -LiteralPath $installer).Hash.ToLowerInvariant() -eq '18dbc670ce4d46809e2a90d5831d17b31f633050907290082f1e793294df8b87') 'published 0.1.42 installer matches accepted immutable checksum'
 $setupAttempted=$true
 $install=Start-Process $installer -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$app+'"'),('/STATE="'+$initialState+'"'),('/LOG="'+(Join-Path $root 'setup.log')+'"')) -Wait -PassThru -WindowStyle Hidden
 Assert ($install.ExitCode -eq 0) 'published baseline installs on the disposable runner'
 $old=Join-Path $app 'immichclone.exe'
 Assert ((& $old -mode version) -eq '0.1.42-prototype') 'schema compatibility uses the genuine published old executable'
 $oldHash=(Get-FileHash -LiteralPath $old).Hash
 $service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
 Assert ($service.StartName -eq 'NT AUTHORITY\LocalService') 'actual old writer runs through LocalService and SCM'
 $initialServicePath=$service.PathName
 Stop-Service ImmichClone

 $state=Join-Path $root 'synthetic-state';$logs=Join-Path $root 'synthetic-logs'
 New-Item -ItemType Directory -Path $state,$logs|Out-Null
 foreach($folder in @($state,$logs)){Checked icacls.exe @($folder,'/grant','*S-1-5-19:(OI)(CI)M')}
 $env:IMMICH_R43_SCHEMA_FIXTURE_ROOT=$state
 Fixture 'create'
 Checked sc.exe @('config','ImmichClone','binPath=',(ServiceCommand $old $state $logs))
 Start-Service ImmichClone
 Assert (WaitHealthy '0.1.42-prototype') 'genuine old writer opens the same synthetic schema-one library as a positive control'
 Stop-Service ImmichClone
 Fixture 'promote'
 $before=StateSnapshot $state

 $oldVerification=@(& $old -mode verify -data $state 2>&1)
 Assert ($LASTEXITCODE -ne 0) 'genuine old read-only verifier refuses schema two'
 Assert ((StateSnapshot $state) -ceq $before) 'old read-only refusal preserves source namespace and every file hash'
 try{Start-Service ImmichClone -ErrorAction Stop}catch{}
 for($i=0;$i -lt 60;$i++){
  $current=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
  if($current.State -eq 'Stopped' -and $current.ProcessId -eq 0){break}
  Start-Sleep -Milliseconds 100
 }
 Assert ($current.State -eq 'Stopped' -and $current.ProcessId -eq 0) 'genuine old LocalService writer refuses the promoted catalog'
 Assert ((StateSnapshot $state) -ceq $before) 'old writer refusal preserves catalog, provenance, marker, access code, originals and other source files'
 Assert ((Get-FileHash -LiteralPath $old).Hash -eq $oldHash) 'old writer checks used the unchanged installed published executable'

 $candidate=Join-Path $root 'candidate.exe'
 Checked go @('build','-mod=readonly','-trimpath','-ldflags','-X main.version=0.1.43-schema-development','-o',$candidate,'./cmd/immichclone')
 Checked sc.exe @('config','ImmichClone','binPath=',(ServiceCommand $candidate $state $logs))
 Start-Service ImmichClone
 Assert (WaitHealthy '0.1.43-schema-development') 'new LocalService writer reopens the identical promoted library'
 $session=[Microsoft.PowerShell.Commands.WebRequestSession]::new()
 $code=(Get-Content -LiteralPath (Join-Path $state 'access-code.txt') -Raw).Trim()
 $null=Invoke-RestMethod 'http://127.0.0.1:2283/api/login' -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
 $org=Invoke-RestMethod 'http://127.0.0.1:2283/api/organization' -WebSession $session
 Assert (@($org.faces|Where-Object {$_.name_source -eq 'automatic' -and $_.name -eq 'Schema person'}).Count -eq 1) 'new service retains automatic provenance after actual old-writer refusal'
 Stop-Service ImmichClone
 Fixture 'inspect'
 Checked $candidate @('-mode','verify','-data',$state)
 Assert $true 'new read-only verifier supports the promoted library'
 Checked sc.exe @('config','ImmichClone','binPath=',$initialServicePath)
} finally {
 Remove-Item Env:IMMICH_R43_SCHEMA_FIXTURE_ROOT,Env:IMMICH_R43_SCHEMA_FIXTURE_STAGE -ErrorAction SilentlyContinue
 if($setupAttempted){
  Stop-Service ImmichClone -Force -ErrorAction SilentlyContinue
  if($initialServicePath -and (Get-Service ImmichClone -ErrorAction SilentlyContinue)){& sc.exe config ImmichClone binPath= $initialServicePath|Out-Null}
  $uninstaller=Join-Path $app 'unins000.exe'
  if(Test-Path -LiteralPath $uninstaller){
   $uninstall=Start-Process $uninstaller -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru -WindowStyle Hidden
   if($uninstall.ExitCode -ne 0){throw 'Disposable published baseline uninstall failed'}
  }
  if(Get-Service ImmichClone -ErrorAction SilentlyContinue){throw 'Owned compatibility service was not removed by uninstall'}
 }
 foreach($item in @(@{Path=$root;Parent=$env:RUNNER_TEMP},@{Path=$app;Parent=$env:ProgramFiles},@{Path=$initialState;Parent=$env:ProgramData})){
  if(Test-Path -LiteralPath $item.Path){
   $target=[IO.Path]::GetFullPath($item.Path)
   $parent=[IO.Path]::GetFullPath($item.Parent).TrimEnd('\')
   if([IO.Path]::GetDirectoryName($target) -cne $parent -or [IO.Path]::GetFileName($target) -cne $name){throw 'Disposable schema cleanup target refused'}
   Remove-Item -LiteralPath $target -Recurse -Force
  }
 }
}
