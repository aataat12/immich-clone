param(
 [Parameter(Mandatory=$true)][string]$InstallerFiles,
 [Parameter(Mandatory=$true)][string]$NativeTests
)
$ErrorActionPreference='Stop'
function Assert([bool]$Condition,[string]$Label) { if(!$Condition){throw $Label}; Write-Host ('PASS: '+$Label) }
function Native([string]$Exe,[string[]]$Arguments) { & $Exe @Arguments | Out-Null; if($LASTEXITCODE -ne 0){throw 'Required fixture operation failed'} }
# A real separately logged-on account; no credentials in process arguments/logs.
Add-Type @'
using System;
using System.IO;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Security.Principal;
using System.Threading;
public sealed class MetadataActor : IDisposable {
 [DllImport("advapi32.dll",CharSet=CharSet.Unicode,SetLastError=true)] static extern bool LogonUser(string u,string d,string p,int t,int v,out IntPtr token);
 [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr h);
 [DllImport("kernel32.dll",CharSet=CharSet.Unicode,SetLastError=true)] static extern bool CreateHardLink(string n,string existing,IntPtr s);
 IntPtr token; Thread thread; ManualResetEvent stop=new ManualResetEvent(false),ready=new ManualResetEvent(false);
 readonly object gate=new object(); string[] targets=new string[0]; string links; FileStream held;
 public long Attempts,Allowed,Denied,Missing,Unexpected,PrimaryDenied; public bool Standard;
 public MetadataActor(string user,string password,string linkDirectory) {
  if(!LogonUser(user,".",password,2,0,out token))throw new Win32Exception(Marshal.GetLastWin32Error());
  links=linkDirectory;
  using(var id=new WindowsIdentity(token))using(var c=id.Impersonate()){Standard=!new WindowsPrincipal(id).IsInRole(WindowsBuiltInRole.Administrator);}
 }
 void Attempt(int index,Action action) {
  Interlocked.Increment(ref Attempts);
  try{action();Interlocked.Increment(ref Allowed);}
  catch(Exception e){
   int code=e is Win32Exception?((Win32Exception)e).NativeErrorCode:(e.HResult&65535);
   if(e is UnauthorizedAccessException||code==5||code==32){Interlocked.Increment(ref Denied);if(index==0)Interlocked.Increment(ref PrimaryDenied);}
   else if(code==2||code==3||code==80||code==183)Interlocked.Increment(ref Missing);
   else Interlocked.Increment(ref Unexpected);
  }
 }
 void Cycle(string[] current){
  for(int i=0;i<current.Length;i++){
   string target=current[i];int index=i;
   Attempt(index,delegate{using(var f=new FileStream(target,FileMode.Open,FileAccess.Write,FileShare.ReadWrite|FileShare.Delete)){f.Seek(0,SeekOrigin.End);f.WriteByte(82);}});
   Attempt(index,delegate{string moved=target+".actor-swap";File.Move(target,moved);File.Move(moved,target);});
   Attempt(index,delegate{string link=Path.Combine(links,"attempt-"+index+".bin");if(!CreateHardLink(link,target,IntPtr.Zero))throw new Win32Exception(Marshal.GetLastWin32Error());File.Delete(link);});
   Attempt(index,delegate{var acl=File.GetAccessControl(target);File.SetAccessControl(target,acl);});
  }
 }
 public void Positive(string target){using(var id=new WindowsIdentity(token))using(var c=id.Impersonate()){Cycle(new string[]{target});}}
 public void SetTargets(string[] values){lock(gate){targets=(string[])values.Clone();}}
 public void Start(){
  thread=new Thread(delegate(){
   try{using(var id=new WindowsIdentity(token))using(var c=id.Impersonate()){
    ready.Set();while(!stop.WaitOne(1)){string[] copy;lock(gate){copy=targets;}Cycle(copy);}
   }}catch{Interlocked.Increment(ref Unexpected);ready.Set();}
  });
  thread.IsBackground=true;thread.Start();if(!ready.WaitOne(5000))throw new Exception("Actor startup timed out");
 }
 public void Stop(){stop.Set();if(thread!=null&&!thread.Join(5000))throw new Exception("Actor reap timed out");thread=null;}
 public void HoldWriter(string target){using(var id=new WindowsIdentity(token))using(var c=id.Impersonate()){held=new FileStream(target,FileMode.Open,FileAccess.Write,FileShare.ReadWrite|FileShare.Delete);}}
 public void ReleaseWriter(){if(held!=null){held.Dispose();held=null;}}
 public void Dispose(){Stop();ReleaseWriter();if(token!=IntPtr.Zero){CloseHandle(token);token=IntPtr.Zero;}stop.Dispose();ready.Dispose();}
}
'@
$root=Join-Path $env:RUNNER_TEMP ('metadata concurrency '+[guid]::NewGuid().ToString('N'))
$actor=$null;$positiveActor=$null;$boundaryProcess=$null
$user='ICRace'+[guid]::NewGuid().ToString('N').Substring(0,8)
$password='Aa1!'+[guid]::NewGuid().ToString('N');$created=$false
function Start-NativeProcess([string]$Exe,[string[]]$Arguments){
 $info=[Diagnostics.ProcessStartInfo]::new();$info.FileName=$Exe
 $info.Arguments=($Arguments|ForEach-Object{'"'+$_+'"'})-join ' '
 $info.UseShellExecute=$false;$info.CreateNoWindow=$true
 $info.RedirectStandardOutput=$true;$info.RedirectStandardError=$true
 $process=[Diagnostics.Process]::new();$process.StartInfo=$info
 if(!$process.Start()){throw 'Native process did not start'}
 return @{Process=$process;Output=$process.StandardOutput.ReadToEndAsync();Error=$process.StandardError.ReadToEndAsync()}
}
function Finish-NativeProcess($Run,[int]$Seconds=15){
 if(!$Run.Process.WaitForExit($Seconds*1000)){$Run.Process.Kill();$Run.Process.WaitForExit();throw 'Native operation timed out and was reaped'}
 $null=$Run.Output.Result;$null=$Run.Error.Result
 $code=$Run.Process.ExitCode;return $code
}
$script:overlap=0
function Invoke-Action([string]$Action,[string]$App,[string]$Data,[string]$Transaction){
 $start=if($actor){$actor.Attempts}else{0}
 $run=Start-NativeProcess $InstallerFiles @('-action',$Action,'-app-dir',$App,'-data-dir',$Data,'-root-dir',$Transaction)
 $deadline=[DateTime]::UtcNow.AddSeconds(15)
 while(!$run.Process.HasExited -and [DateTime]::UtcNow -lt $deadline){
  if($actor -and $actor.Attempts -gt $start){$script:overlap++}
  Start-Sleep -Milliseconds 2
 }
 $code=Finish-NativeProcess $run;$run.Process.Dispose();return $code
}
function Snapshot([string]$Path){return ((Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash+'|'+(Get-Acl -LiteralPath $Path).Sddl)}
function Wait-Condition([scriptblock]$Condition,[int]$Seconds=10){
 $until=[DateTime]::UtcNow.AddSeconds($Seconds)
 while([DateTime]::UtcNow -lt $until){if(&$Condition){return $true};Start-Sleep -Milliseconds 10}
 return $false
}
try{
 New-Item -ItemType Directory -Path $root|Out-Null
 Native icacls.exe @($root,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F','*S-1-5-32-545:(OI)(CI)RX')
 New-LocalUser -Name $user -Password (ConvertTo-SecureString $password -AsPlainText -Force) -AccountNeverExpires|Out-Null
 $created=$true;Add-LocalGroupMember -SID 'S-1-5-32-545' -Member $user
 $sid=(Get-LocalUser -Name $user).SID.Value
 $permissive=Join-Path $root 'permissive';New-Item -ItemType Directory -Path $permissive|Out-Null
 Native icacls.exe @($permissive,'/grant','*S-1-5-32-545:(OI)(CI)F')
 $positive=Join-Path $permissive 'positive.txt';[IO.File]::WriteAllText($positive,'positive')
 $positiveActor=[MetadataActor]::new($user,$password,$permissive)
 Assert $positiveActor.Standard 'actor is a separately logged-on standard account'
 $positiveActor.Positive($positive)
 Assert ($positiveActor.Allowed -eq 4 -and $positiveActor.Denied -eq 0 -and $positiveActor.Unexpected -eq 0 -and $positiveActor.Missing -eq 0) 'same account positively writes renames creates hardlinks and applies ACLs in permissive control'
 Assert ([IO.File]::ReadAllText($positive) -eq 'positiveR') 'positive control actually changes bytes'
 $positiveActor.Dispose();$positiveActor=$null
 $app=Join-Path $root 'app';$data=Join-Path $root 'data';$source=Join-Path $root 'source';$transaction=$app+'-upgrade-recovery'
 foreach($dir in @($app,$data,$source)){Native $InstallerFiles @('-action','ensure-private-root','-root-dir',$dir)}
 $files=@('Manage-Service.ps1','Storage-Security.ps1','installer-files.exe','immichclone.exe','Export-Diagnostics.ps1','THIRD-PARTY-LICENSES.txt','prototype-notes.txt','media/runtime.bin','ai/model.bin')
 foreach($name in $files){foreach($dir in @($app,$source)){
  $path=Join-Path $dir $name;Native $InstallerFiles @('-action','ensure-private-root','-root-dir',(Split-Path -Parent $path))
  $prefix=if($dir -eq $app){'prior-'}else{'candidate-'};[IO.File]::WriteAllText($path,($prefix+$name))
 }}
 $outside=Join-Path $data 'outside.txt';[IO.File]::WriteAllText($outside,'outside sentinel')
 $catalog=Join-Path $data 'catalog.db';[IO.File]::WriteAllText($catalog,'prior catalog')
 $code=Join-Path $data 'access-code.txt';[IO.File]::WriteAllText($code,'synthetic code')
 $outsideBefore=Snapshot $outside;$catalogBefore=Snapshot $catalog;$codeBefore=Snapshot $code
 $pending=Join-Path $transaction 'upgrade-pending.json';$payload=Join-Path $transaction 'payload'
 $actor=[MetadataActor]::new($user,$password,$permissive);$actor.SetTargets([string[]]@($pending,$outside));$actor.Start()
 Assert ((Invoke-Action 'begin-upgrade' $app $data $transaction) -eq 0) 'real native begin runs with active separate-account actor'
 Assert ((Test-Path -LiteralPath $pending -PathType Leaf) -and (Get-Item -LiteralPath $pending).Length -gt 0) 'pending metadata exists and is nonempty'
 $pendingBefore=Snapshot $pending
 Assert (Wait-Condition {$actor.PrimaryDenied -ge 8}) 'actor actually attempts protected pending operations'
 for($i=0;$i -lt 12;$i++){Assert ((Invoke-Action 'audit-upgrade' $app $data $transaction) -eq 0) 'native audit succeeds during concurrent leaf replacement attempts'}
 Assert ($script:overlap -gt 0) 'attempt counters advance while native processes are active'
 $actor.Stop()
 Assert ($actor.Allowed -eq 0 -and $actor.Denied -gt 0 -and $actor.Unexpected -eq 0) 'all protected writes renames outgoing hardlinks and ACL changes are refused'
 Assert ((Snapshot $pending) -eq $pendingBefore) 'pending bytes and descriptor are preserved'
 # A preauthorized writable handle survives DACL removal. Sharing must exclude it.
 $saved=Get-Acl -LiteralPath $pending;$grant=Get-Acl -LiteralPath $pending
 $grant.AddAccessRule([Security.AccessControl.FileSystemAccessRule]::new([Security.Principal.SecurityIdentifier]::new($sid),'Modify','Allow'))
 Set-Acl -LiteralPath $pending -AclObject $grant
 try{$actor.HoldWriter($pending)}finally{Set-Acl -LiteralPath $pending -AclObject $saved}
 Assert ((Invoke-Action 'audit-upgrade' $app $data $transaction) -ne 0) 'audit refuses actual preheld standard-account writer after private DACL restoration'
 $actor.ReleaseWriter()
 Assert ((Invoke-Action 'audit-upgrade' $app $data $transaction) -eq 0 -and (Snapshot $pending) -eq $pendingBefore) 'writer release permits unchanged retry'
 $actor.Dispose();$actor=$null;$actor=[MetadataActor]::new($user,$password,$permissive)
 $actor.SetTargets([string[]]@($pending,(Join-Path $payload 'publication-intent.json'),(Join-Path $payload 'publication-ready.json'),(Join-Path $payload 'file-000000.bin'),(Join-Path $transaction 'upgrade-terminal.json'),$outside));$actor.Start()
 Native $InstallerFiles @('-action','publish-payload','-app-dir',$app,'-source-dir',$source,'-backup-dir',$payload)
 Assert ((Invoke-Action 'commit-upgrade' $app $data $transaction) -eq 0) 'real complete publication and commit run under separate-account attempts'
 Assert ((Test-Path -LiteralPath $payload -PathType Container) -and @(Get-ChildItem -LiteralPath $payload -Force).Count -eq 0 -and !(Test-Path $pending) -and !(Test-Path (Join-Path $transaction 'upgrade-terminal.json'))) 'commit retains only accepted empty recovery directories'
 foreach($name in $files){Assert ([IO.File]::ReadAllText((Join-Path $app $name)) -eq ('candidate-'+$name)) 'candidate payload bytes remain coherent'}
 # A compiled Go test supplies a deterministic barrier after all leaves are held.
 $boundary=Join-Path $root 'boundary';New-Item -ItemType Directory -Path $boundary|Out-Null
 $env:IMMICH_METADATA_FIXTURE=$boundary
 $boundaryProcess=Start-NativeProcess $NativeTests @('-test.run=^TestMetadataRetirementActorBoundary$','-test.v','-test.timeout=40s')
 $held=Join-Path $boundary 'held.json'
 Assert (Wait-Condition {Test-Path -LiteralPath $held} 15) 'test-only barrier reaches actual held retirement boundary'
 $info=Get-Content -LiteralPath $held -Raw|ConvertFrom-Json
 $actor.SetTargets([string[]]@((Join-Path $info.transaction 'upgrade-terminal.json'),(Join-Path $info.transaction 'upgrade-pending.json'),(Join-Path $info.transaction 'payload/file-000000.bin'),$outside))
 $before=$actor.PrimaryDenied
 Assert (Wait-Condition {$actor.PrimaryDenied -ge ($before+8)}) 'separate account attempts terminal replacement while retirement holds selected objects'
 [IO.File]::WriteAllText((Join-Path $boundary 'continue'),'continue')
 $boundaryCode=Finish-NativeProcess $boundaryProcess 40;$boundaryProcess.Process.Dispose();$boundaryProcess=$null
 Assert ($boundaryCode -eq 0) 'partial retirement retains terminal and retries without committed code rollback'
 $actor.Stop()
 Assert ($actor.Allowed -eq 0 -and $actor.Denied -gt 0 -and $actor.Unexpected -eq 0) 'publication and retirement actor never acquires protected mutation capability'
 Assert ((Snapshot $outside) -eq $outsideBefore -and (Snapshot $catalog) -eq $catalogBefore -and (Snapshot $code) -eq $codeBefore) 'outside catalog and access-code bytes and ACLs are unchanged'
 Assert ((Invoke-Action 'recover-upgrade' $app $data $transaction) -eq 0) 'post-commit recovery is idempotent'
}
finally{
 Remove-Item Env:IMMICH_METADATA_FIXTURE -ErrorAction SilentlyContinue
 if($boundaryProcess -and !$boundaryProcess.Process.HasExited){$boundaryProcess.Process.Kill();$boundaryProcess.Process.WaitForExit();$boundaryProcess.Process.Dispose()}
 if($actor){$actor.Dispose()};if($positiveActor){$positiveActor.Dispose()}
 if($created){Remove-LocalUser -Name $user -Confirm:$false -ErrorAction SilentlyContinue}
 if(Test-Path -LiteralPath $root){
  $resolved=[IO.Path]::GetFullPath($root);$parent=[IO.Path]::GetFullPath($env:RUNNER_TEMP).TrimEnd('\')+'\'
  if(!$resolved.StartsWith($parent,[StringComparison]::OrdinalIgnoreCase)){throw 'Fixture cleanup target refused'}
  Remove-Item -LiteralPath $resolved -Recurse -Force
 }
}
Write-Host 'PASS: native metadata concurrency checkpoint; full Setup concurrency still requires acceptance'
