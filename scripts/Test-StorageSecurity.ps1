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
 [DllImport("advapi32.dll", CharSet=CharSet.Unicode)] static extern uint SetNamedSecurityInfo(string name,int type,uint info,IntPtr owner,IntPtr group,IntPtr dacl,IntPtr sacl);
 public static void NullDacl(string path) {
  uint result=SetNamedSecurityInfo(path,1,0x80000004u,IntPtr.Zero,IntPtr.Zero,IntPtr.Zero,IntPtr.Zero);
  if(result!=0) throw new System.ComponentModel.Win32Exception((int)result);
 }
 public static void Impersonate(string user,string password,Action action) {
  IntPtr token;
  if(!LogonUser(user,".",password,2,0,out token)) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
  try { using(var identity=new WindowsIdentity(token)) using(var context=identity.Impersonate()) { action(); }} finally { CloseHandle(token); }
 }
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
 $ordinarySid=(Get-LocalUser -Name $user).SID.Value
 Add-LocalGroupMember -SID 'S-1-5-32-545' -Member $user
 $fixture=Join-Path $root 'acl'
 New-Item -ItemType Directory $fixture | Out-Null
 Native icacls @($fixture,'/grant','*S-1-1-0:(OI)(CI)F')
 $secret=Join-Path $fixture 'synthetic-secret.txt'
 [IO.File]::WriteAllText($secret,'synthetic secret')
 Assert ([StorageAccessProbe]::Access($user,$password,$secret,$false)) 'ordinary-user positive control can read permissive fixture'
 Native icacls @($fixture,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F','*S-1-5-19:(OI)(CI)M',('*'+$identity+':(OI)(CI)RX'))
 Assert ([StorageAccessProbe]::Access($user,$password,$secret,$false)) 'previous grant command reproduces retained explicit Everyone access'
 Assert-PrivateDirectoryTree -Path $fixture -ReaderSid $identity -ServiceAccess
 Assert $true 'normal inherited grants pass read-only descendant audit before root normalization'
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
 # Stationary descendant permissions: rejection must precede stop/snapshot/logging.
 $rollback=Join-Path $app 'rollback'
 Set-PrivateDirectoryAcl -Path $rollback
 $rollbackGuard=Join-Path $rollback 'guard.bin'
 [IO.File]::WriteAllText($rollbackGuard,'rollback guard')
 $logRoot=Join-Path $env:ProgramData 'ImmichCloneDiagnostics'
 $installerLog=Join-Path $logRoot 'installer.jsonl'
 $installDiagnostic=Join-Path $app 'install-diagnostic.txt'
 function New-StateFixture([string]$Label) {
  $candidate=Join-Path $root ('case-'+$Label)
  Set-PrivateDirectoryAcl -Path $candidate -ReaderSid $identity -ServiceAccess
  foreach($folder in @('originals','backups','nested')) {New-Item -ItemType Directory -Path (Join-Path $candidate $folder) | Out-Null}
  foreach($file in @('state-format.txt','catalog.db','access-code.txt','originals/original.bin','backups/backup.zip','nested/state.bin')) {
   [IO.File]::WriteAllText((Join-Path $candidate $file),'synthetic stationary fixture')
  }
  return $candidate
 }
 function Add-OrdinaryGrant([string]$Path,[bool]$Protected) {
  $a=Get-Acl -LiteralPath $Path
  if($Protected){$a.SetAccessRuleProtection($true,$true)}
  $flags=[Security.AccessControl.InheritanceFlags]::None
  if((Get-Item -LiteralPath $Path -Force).PSIsContainer){$flags=[Security.AccessControl.InheritanceFlags]'ObjectInherit,ContainerInherit'}
  $rule=New-Object Security.AccessControl.FileSystemAccessRule((New-Object Security.Principal.SecurityIdentifier($ordinarySid)),[Security.AccessControl.FileSystemRights]::Modify,$flags,[Security.AccessControl.PropagationFlags]::None,[Security.AccessControl.AccessControlType]::Allow)
  $a.AddAccessRule($rule);Set-Acl -LiteralPath $Path -AclObject $a
 }
 function Snapshot([string[]]$Paths) {
  $parts=@()
  foreach($path in $Paths) {
   if(!(Test-Path -LiteralPath $path)) {$parts+='absent';continue}
   $item=Get-Item -LiteralPath $path -Force
   $part=(Get-Acl -LiteralPath $path).Sddl
   if(!$item.PSIsContainer){$part+='|'+(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash}
   elseif(($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -eq 0) {$part+='|'+((@(Get-ChildItem -LiteralPath $path -Force | Select-Object -ExpandProperty Name | Sort-Object)) -join ',')}
   $parts+=$part
  }
  return $parts -join "`n"
 }
 function Rejected-Prepare([string]$Candidate,[string[]]$Watched,[string]$Label) {
  $beforeService=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
  $watch=@($Candidate,$rollback,$rollbackGuard,$installerLog,$installDiagnostic)+$Watched
  $before=Snapshot $watch
  & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File (Join-Path $app 'Manage-Service.ps1') -Action Prepare -AppDir $app -DataDir $Candidate *> $null
  $exit=$LASTEXITCODE
  $afterService=Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
  Assert ($exit -ne 0) ($Label+' rejected by production management preflight')
  Assert ($beforeService.State -eq 'Running' -and $afterService.State -eq 'Running' -and $beforeService.ProcessId -eq $afterService.ProcessId) ($Label+' rejected before service stop/restart')
  Assert ((Snapshot $watch) -eq $before) ($Label+' preserves watched bytes ACLs diagnostics and rollback')
 }
 foreach($kind in @('unprotected','protected')) {
  foreach($entry in @('nested/state.bin','originals/original.bin','catalog.db','access-code.txt','backups/backup.zip')) {
   $candidate=New-StateFixture ($kind+'-'+[guid]::NewGuid().ToString('N'))
   $bad=Join-Path $candidate $entry
   Add-OrdinaryGrant $bad ($kind -eq 'protected')
   Assert ([StorageAccessProbe]::Access($user,$password,$bad,$false)) 'ordinary-user descendant read positive control'
   Assert ([StorageAccessProbe]::Access($user,$password,$bad,$true)) 'ordinary-user descendant write positive control'
   Rejected-Prepare $candidate @($bad) ($kind+' data descendant')
  }
 }
 # A foreign inheritable grant on a directory is caught at that explicit ancestor.
 $candidate=New-StateFixture 'directory-grant'
 $bad=Join-Path $candidate 'originals'
 Add-OrdinaryGrant $bad $false
 $inherited=Join-Path $bad 'original.bin'
 Assert ([StorageAccessProbe]::Access($user,$password,$inherited,$false)) 'ordinary-user inherited child read positive control'
 Assert ([StorageAccessProbe]::Access($user,$password,$inherited,$true)) 'ordinary-user inherited child write positive control'
 Rejected-Prepare $candidate @($bad,$inherited) 'explicit directory grant with inherited descendants'
 foreach($kind in @('unprotected','protected')) {
  foreach($location in @('logs','rollback')) {
   if($location -eq 'logs') {$bad=Join-Path $logRoot ('permission-case-'+$kind+'.bin')}
   else {$bad=Join-Path $rollback ('permission-case-'+$kind+'.bin')}
   [IO.File]::WriteAllText($bad,'synthetic private fixture')
   Add-OrdinaryGrant $bad ($kind -eq 'protected')
   Assert ([StorageAccessProbe]::Access($user,$password,$bad,$false)) 'ordinary-user private descendant read positive control'
   Assert ([StorageAccessProbe]::Access($user,$password,$bad,$true)) 'ordinary-user private descendant write positive control'
   try {Rejected-Prepare $state @($bad) ($kind+' '+$location+' descendant')}
   finally {Remove-Item -LiteralPath $bad -Force}
  }
 }


 # Intended principals still cannot add broader rights or explicit denial rules.
 foreach($ruleKind in @('excess-allow','explicit-deny')) {
  $candidate=New-StateFixture $ruleKind
  $bad=Join-Path $candidate 'nested/state.bin'
  $a=Get-Acl -LiteralPath $bad
  $type=[Security.AccessControl.AccessControlType]::Allow
  $rights=[Security.AccessControl.FileSystemRights]::FullControl
  if($ruleKind -eq 'explicit-deny') {$type=[Security.AccessControl.AccessControlType]::Deny;$rights=[Security.AccessControl.FileSystemRights]::Write}
  $a.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule((New-Object Security.Principal.SecurityIdentifier('S-1-5-19')),$rights,$type)))
  Set-Acl -LiteralPath $bad -AclObject $a
  Rejected-Prepare $candidate @($bad) $ruleKind
 }
 # A protected NULL DACL grants everyone access and cannot be skipped merely
 # because an access-rule enumeration returns no ACEs.
 $candidate=New-StateFixture 'null-dacl'
 $nullAcl=Join-Path $candidate 'nested/null-dacl.bin'
 [IO.File]::WriteAllText($nullAcl,'synthetic null dacl fixture')
 [StorageAccessProbe]::NullDacl($nullAcl)
 Assert ([StorageAccessProbe]::Access($user,$password,$nullAcl,$false)) 'NULL DACL ordinary-user read positive control'
 Assert ([StorageAccessProbe]::Access($user,$password,$nullAcl,$true)) 'NULL DACL ordinary-user write positive control'
 Rejected-Prepare $candidate @($nullAcl) 'NULL DACL descendant'
 # An untrusted owner retains permission-changing authority even after an
 # inherited foreign grant is removed. Create under that real user token.
 $candidate=New-StateFixture 'untrusted-owner'
 $ownerParent=Join-Path $candidate 'nested'
 Add-OrdinaryGrant $ownerParent $false
 $untrustedOwner=Join-Path $ownerParent 'owned-by-ordinary-user.bin'
 Assert ([StorageAccessProbe]::Access($user,$password,$untrustedOwner,$true)) 'ordinary-user file creation owner positive control'
 $a=Get-Acl -LiteralPath $ownerParent
 $a.PurgeAccessRules((New-Object Security.Principal.SecurityIdentifier($ordinarySid)))
 Set-Acl -LiteralPath $ownerParent -AclObject $a
 $owner=(Get-Acl -LiteralPath $untrustedOwner).GetOwner([Security.Principal.SecurityIdentifier]).Value
 Assert ($owner -eq $ordinarySid) 'untrusted owner remains after permission inheritance repair'
 Rejected-Prepare $candidate @($ownerParent,$untrustedOwner) 'untrusted owner descendant'
 # Intended explicit rules and a complete protected directory remain compatible.
 $candidate=New-StateFixture 'normal-known-rules'
 $protected=Join-Path $candidate 'protected'
 Set-PrivateDirectoryAcl -Path $protected -ReaderSid $identity -ServiceAccess
 [IO.File]::WriteAllText((Join-Path $protected 'safe.bin'),'safe protected fixture')
 $normal=Join-Path $candidate 'nested/state.bin'
 $a=Get-Acl -LiteralPath $normal
 $a.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule((New-Object Security.Principal.SecurityIdentifier($identity)),[Security.AccessControl.FileSystemRights]::ReadAndExecute,[Security.AccessControl.AccessControlType]::Allow)))
 Set-Acl -LiteralPath $normal -AclObject $a
 Assert-PrivateDirectoryTree -Path $candidate -ReaderSid $identity -ServiceAccess
 Assert $true 'intended explicit rights and complete protected descendants accepted'
 foreach($entry in @('nested/state.bin','originals/original.bin','catalog.db','access-code.txt','backups/backup.zip','protected/safe.bin')) {
  $safe=Join-Path $candidate $entry
  Assert (![StorageAccessProbe]::Access($user,$password,$safe,$false)) 'configured descendant denies actual ordinary-user reads'
  Assert (![StorageAccessProbe]::Access($user,$password,$safe,$true)) 'configured descendant denies actual ordinary-user writes'
 }
 $safeLog=Join-Path $logRoot 'safe-permission-fixture.bin'
 [IO.File]::WriteAllText($safeLog,'safe log fixture')
 try {
  foreach($safe in @($safeLog,$rollbackGuard)) {
   Assert (![StorageAccessProbe]::Access($user,$password,$safe,$false)) 'configured private descendant denies actual ordinary-user reads'
   Assert (![StorageAccessProbe]::Access($user,$password,$safe,$true)) 'configured private descendant denies actual ordinary-user writes'
  }
 } finally {Remove-Item -LiteralPath $safeLog -Force}

 # An incomplete protected child must not silently deny required service access.
 $incomplete=Join-Path $candidate 'nested/incomplete.bin'
 [IO.File]::WriteAllText($incomplete,'incomplete protected fixture')
 $a=Get-Acl -LiteralPath $incomplete;$a.SetAccessRuleProtection($true,$false)
 $a.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule((New-Object Security.Principal.SecurityIdentifier('S-1-5-32-544')),[Security.AccessControl.FileSystemRights]::FullControl,[Security.AccessControl.AccessControlType]::Allow)))
 Set-Acl -LiteralPath $incomplete -AclObject $a
 Rejected-Prepare $candidate @($incomplete) 'incomplete protected descendant'
 # No-follow descendant junction, preserving target descriptor and data.
 $candidate=New-StateFixture 'child-junction'
 $target=Join-Path $root 'child-junction-target';New-Item -ItemType Directory $target | Out-Null
 $targetFile=Join-Path $target 'sentinel.bin';[IO.File]::WriteAllText($targetFile,'outside child-junction sentinel')
 $childLink=Join-Path $candidate 'nested/junction'
 New-Item -ItemType Junction -Path $childLink -Target $target | Out-Null
 try {Rejected-Prepare $candidate @($target,$targetFile) 'descendant junction'}
 finally {[IO.Directory]::Delete($childLink)}
 # Execute the exact audit using a real ordinary-user token. The normal tree
 # passes first; after a child becomes unreadable, enumeration/Get-Acl must fail.
 $denied=Join-Path $root 'audit-access-denied'
 Set-PrivateDirectoryAcl -Path $denied -ReaderSid $ordinarySid -ServiceAccess
 $deniedChild=Join-Path $denied 'child';New-Item -ItemType Directory $deniedChild | Out-Null
 [IO.File]::WriteAllText((Join-Path $deniedChild 'safe.bin'),'audit access fixture')
 [StorageAccessProbe]::Impersonate($user,$password,[Action]{Assert-PrivateDirectoryTree -Path $denied -ReaderSid $ordinarySid -ServiceAccess -OwnerSid $identity})
 Assert $true 'ordinary-user audit positive control reads complete normal tree'
 Set-PrivateDirectoryAcl -Path $deniedChild
 $deniedAudit=$false
 try {[StorageAccessProbe]::Impersonate($user,$password,[Action]{Assert-PrivateDirectoryTree -Path $denied -ReaderSid $ordinarySid -ServiceAccess -OwnerSid $identity})} catch {$deniedAudit=$true}
 Assert $deniedAudit 'actual audit access failure is rejected rather than skipped'
 Assert ((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') 'normal LocalService remains usable after rejected descendant fixtures'
 Assert ((Get-FileHash (Join-Path $state ('originals/'+$upload.asset.id))).Hash -eq (Get-FileHash $source).Hash) 'normal original unchanged after descendant permission tests'

} finally {
 if($serviceAttempted) {Stop-Service ImmichClone -Force -ErrorAction SilentlyContinue;& sc.exe delete ImmichClone | Out-Null}
 if($created) {Remove-LocalUser -Name $user -ErrorAction SilentlyContinue}
 $link=Join-Path $root 'junction';if(Test-Path -LiteralPath $link){[IO.Directory]::Delete($link)}
 if(Test-Path -LiteralPath $root){Remove-Item -LiteralPath $root -Recurse -Force}
}
