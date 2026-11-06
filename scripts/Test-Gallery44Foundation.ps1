$ErrorActionPreference='Stop'
function Checked([string]$Exe,[string[]]$Arguments){ & $Exe @Arguments; if($LASTEXITCODE -ne 0){throw 'Feature fixture operation failed'} }
$root=Join-Path $env:RUNNER_TEMP ('gallery44-foundation-'+[guid]::NewGuid().ToString('N'))
$owned=$false
try {
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
} finally {
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
