param([Parameter(Mandatory=$true)][string]$Binary)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot '../packaging/Storage-Security.ps1')
function Assert([bool]$Value,[string]$Label) { if (!$Value) { throw $Label }; Write-Output "PASS: $Label" }
function Native([string]$Exe,[string[]]$Arguments) { & $Exe @Arguments | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'Fixture native operation failed' } }
Add-Type @'
using System;
using System.IO;
using System.Runtime.InteropServices;
using System.Security.Principal;
public static class StorageAccessProbe {
 [DllImport("advapi32.dll", CharSet=CharSet.Unicode, SetLastError=true)] static extern bool LogonUser(string u,string d,string p,int t,int v,out IntPtr token);
 [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr h);
 public static bool Access(string user,string password,string path,bool write) {
  IntPtr token;
  if(!LogonUser(user,".",password,2,0,out token)) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
  try { using(var identity=new WindowsIdentity(token)) using(var context=identity.Impersonate()) {
   try { if(write) File.WriteAllText(path,"synthetic access probe"); else File.ReadAllText(path); return true; }
   catch(UnauthorizedAccessException) { return false; }
  }} finally { CloseHandle(token); }
 }
}
'@
$root=Join-Path $env:RUNNER_TEMP ('storage-security-'+[guid]::NewGuid().ToString('N'))
$user='ICSec'+[guid]::NewGuid().ToString('N').Substring(0,10)
$password='Aa1!'+[guid]::NewGuid().ToString('N')
$identity=[Security.Principal.WindowsIdentity]::GetCurrent().User.Value
$created=$false
$serviceAttempted=$false
try {
 New-Item -ItemType Directory $root | Out-Null
 Native icacls @($root,'/grant','*S-1-1-0:(OI)(CI)RX','*S-1-5-19:(OI)(CI)RX')
 New-LocalUser -Name $user -Password (ConvertTo-SecureString $password -AsPlainText -Force) -AccountNeverExpires | Out-Null
 $created=$true
 Add-LocalGroupMember -SID 'S-1-5-32-545' -Member $user
 $fixture=Join-Path $root 'acl'
 New-Item -ItemType Directory $fixture | Out-Null
 Native icacls @($fixture,'/grant','*S-1-1-0:(OI)(CI)F')
 $secret=Join-Path $fixture 'synthetic-secret.txt'
 [IO.File]::WriteAllText($secret,'synthetic secret')
 Assert ([StorageAccessProbe]::Access($user,$password,$secret,$false)) 'ordinary-user positive control can read permissive fixture'
 Native icacls @($fixture,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F','*S-1-5-19:(OI)(CI)M',('*'+$identity+':(OI)(CI)RX'))
 Assert ([StorageAccessProbe]::Access($user,$password,$secret,$false)) 'previous grant command reproduces retained explicit Everyone access'
 Set-PrivateDirectoryAcl -Path $fixture -ReaderSid $identity -ServiceAccess
 Assert (![StorageAccessProbe]::Access($user,$password,$secret,$false)) 'replacement DACL denies ordinary-user secret read'
 Assert (![StorageAccessProbe]::Access($user,$password,(Join-Path $fixture 'forbidden.txt'),$true)) 'replacement DACL denies ordinary-user write'
 $acl=Get-Acl -LiteralPath $fixture
 $allowed=@('S-1-5-18','S-1-5-32-544','S-1-5-19',$identity)
 Assert ($acl.AreAccessRulesProtected -and @($acl.Access | Where-Object { $_.IdentityReference.Translate([Security.Principal.SecurityIdentifier]).Value -notin $allowed }).Count -eq 0) 'protected root DACL contains only intended principals'
 $outside=Join-Path $root 'outside'
 New-Item -ItemType Directory $outside | Out-Null
 $sentinel=Join-Path $outside 'sentinel.txt';[IO.File]::WriteAllText($sentinel,'untouched')
 $before=(Get-Acl -LiteralPath $outside).Sddl
 $link=Join-Path $root 'junction'
 New-Item -ItemType Junction -Path $link -Target $outside | Out-Null
 foreach($path in @($link,(Join-Path $link 'nested'))) {
  $rejected=$false;try { Set-PrivateDirectoryAcl -Path $path -ServiceAccess } catch { $rejected=$true }
  Assert $rejected 'junction or junction ancestor rejected before ACL/create'
 }
 Assert ((Get-Acl -LiteralPath $outside).Sddl -eq $before -and [IO.File]::ReadAllText($sentinel) -eq 'untouched' -and !(Test-Path (Join-Path $outside 'nested'))) 'junction target ACL and bytes remain untouched'
 [IO.Directory]::Delete($link)
 $app=Join-Path $root 'app';New-Item -ItemType Directory $app | Out-Null
 Copy-Item -LiteralPath $Binary -Destination (Join-Path $app 'immichclone.exe')
 Copy-Item -LiteralPath (Join-Path $PSScriptRoot '../packaging/Storage-Security.ps1') -Destination $app
 Copy-Item -LiteralPath (Join-Path $PSScriptRoot '../packaging/Manage-Service.ps1') -Destination $app
 $state=Join-Path $root 'state';New-Item -ItemType Directory $state | Out-Null
 Native icacls @($state,'/grant','*S-1-1-0:(OI)(CI)F')
 if(Get-Service ImmichClone -ErrorAction SilentlyContinue) { throw 'Fixture refuses an existing service' }
 $serviceAttempted=$true
 & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File (Join-Path $app 'Manage-Service.ps1') -Action Install -AppDir $app -DataDir $state
 Assert ($LASTEXITCODE -eq 0) 'production management helper installs with corrected ACLs'
 $service=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
 Assert ($service.StartName -eq 'NT AUTHORITY\LocalService' -and $service.State -eq 'Running') 'actual application runs as LocalService after ACL replacement'
 Assert (![StorageAccessProbe]::Access($user,$password,(Join-Path $state 'access-code.txt'),$false)) 'ordinary user cannot read initialized access code'
 $code=([IO.File]::ReadAllText((Join-Path $state 'access-code.txt'))).Trim()
 $session=New-Object Microsoft.PowerShell.Commands.WebRequestSession
 $null=Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
 $source=Join-Path $root 'original.bin';[IO.File]::WriteAllText($source,'service original fixture')
 $upload=Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=fixture.bin' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $source -WebSession $session
 Assert ($upload.asset.id -eq (Get-FileHash $source).Hash.ToLowerInvariant()) 'LocalService can commit original and catalog under corrected ACLs'
 Restart-Service ImmichClone
 $healthy=$false
 for($i=0;$i -lt 20;$i++) {try {$healthy=(Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok'} catch {};if($healthy){break};Start-Sleep -Milliseconds 500}
 Assert $healthy 'LocalService restart retains usable state'
 Assert ((Get-FileHash (Join-Path $state ('originals/'+$upload.asset.id))).Hash -eq (Get-FileHash $source).Hash) 'committed original bytes survive restart'
} finally {
 if($serviceAttempted) {Stop-Service ImmichClone -Force -ErrorAction SilentlyContinue;& sc.exe delete ImmichClone | Out-Null}
 if($created) {Remove-LocalUser -Name $user -ErrorAction SilentlyContinue}
 $link=Join-Path $root 'junction';if(Test-Path -LiteralPath $link){[IO.Directory]::Delete($link)}
 if(Test-Path -LiteralPath $root){Remove-Item -LiteralPath $root -Recurse -Force}
}
