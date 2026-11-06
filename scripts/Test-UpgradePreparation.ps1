param(
    [Parameter(Mandatory=$true)][string]$Binary,
    [string]$Compiler = ''
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot '../packaging/Storage-Security.ps1')
$ProgressPreference = 'SilentlyContinue'
function Assert([bool]$Condition,[string]$Label) { if (!$Condition) { throw $Label }; Write-Host "PASS: $Label" }
function Native([string]$Exe,[string[]]$Arguments) { & $Exe @Arguments | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'Preparation probe native operation failed' } }
function Wait-Health {
    for ($i=0;$i -lt 30;$i++) {
        try { if ((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') { return } } catch { }
        Start-Sleep -Milliseconds 500
    }
    throw 'Preparation probe service did not become healthy'
}
function Run-Setup([string]$Application,[string]$State,[switch]$ObserveStaging,[switch]$InterruptPublication) {
    $process = Start-Process $script:setup -WindowStyle Hidden -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$Application+'"'),('/STATE="'+$State+'"')) -PassThru
    $observed = $false
    while (!$process.HasExited) {
        if ($InterruptPublication) {
            $intent = Join-Path ($Application+'-upgrade-recovery') 'payload/publication-intent.json'
            $ready = Join-Path ($Application+'-upgrade-recovery') 'payload/publication-ready.json'
            if ((Test-Path -LiteralPath $intent) -and (Test-Path -LiteralPath $ready) -and (Get-Item -LiteralPath $ready).Length -eq 0) {
                $helpers = @(Get-CimInstance Win32_Process -Filter "Name='installer-files.exe'" | Where-Object { $_.CommandLine -and $_.CommandLine.Contains($Application) -and $_.CommandLine.Contains('publish-payload') })
                if ($helpers.Count -eq 1 -and (Get-Item -LiteralPath $ready).Length -eq 0) {
                    $service = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'"
                    Assert ($service.State -eq 'Stopped' -and $service.StartMode -eq 'Disabled') 'actual publication keeps service stopped and disabled before process-tree termination'
                    Native taskkill.exe @('/PID',[string]$process.Id,'/T','/F')
                    $process.WaitForExit()
                    Assert (!(Get-Process -Id $helpers[0].ProcessId -ErrorAction SilentlyContinue)) 'actual Setup publication helper was killed with its parent process tree'
                    Assert ((Test-Path -LiteralPath $intent) -and (Get-Item -LiteralPath $ready).Length -eq 0) 'terminated Setup retains prewrite intent without a completion receipt'
                    return 73
                }
            }
        }
        if ($ObserveStaging -and !$observed) {
            $children = @(Get-CimInstance Win32_Process -Filter "Name='powershell.exe'" | Where-Object { $_.CommandLine -and $_.CommandLine.Contains($Application) -and $_.CommandLine.Contains('-Action ') })
            foreach ($child in $children) {
                $match = [regex]::Match($child.CommandLine,'-File "([^"\r\n]+Manage-Service\.ps1)"')
                if (!$match.Success) { continue }
                $staged = $match.Groups[1].Value
                Assert-TrustedCodeTree (Split-Path $staged -Parent)
                Assert $true 'actual staging directory passes the production code trust audit'
                # No paths/command lines are printed. Check both exact extracted
                # helpers against writes by even the current elevated account.
                foreach ($file in @($staged,(Join-Path (Split-Path $staged -Parent) 'Storage-Security.ps1'),(Join-Path (Split-Path $staged -Parent) 'installer-files.exe'))) {
                    $locked = $false
                    try { $f = [IO.File]::Open($file,[IO.FileMode]::Open,[IO.FileAccess]::Write,[IO.FileShare]::ReadWrite -bor [IO.FileShare]::Delete); $f.Dispose() }
                    catch [IO.IOException] { $locked = (($_.Exception.HResult -band 0xffff) -eq 32) }
                    Assert $locked 'actual extracted helper denies writes while Setup holds its sharing lock'
                    Assert (![UpgradeWriteProbe]::CanWrite($ordinaryUser,$ordinaryPassword,$file)) 'actual extracted helper denies separate standard-user writes'
                }
                $observed = $true
                break
            }
        }
        Start-Sleep -Milliseconds 30
        $process.Refresh()
    }
    $process.WaitForExit()
    if ($ObserveStaging) { Assert $observed 'actual staged helper process observed during production Setup' }
    if ($InterruptPublication) { throw 'Setup completed before the required publication interruption' }
    return $process.ExitCode
}
function Write-RetainedHelper([string]$Application) {
    @'
param([string]$Action,[string]$AppDir,[string]$DataDir)
[IO.File]::WriteAllText((Join-Path $PSScriptRoot 'retained-preparer-ran.txt'),'unexpected retained preparer execution')
exit 0
'@ | Set-Content -LiteralPath (Join-Path $Application 'Manage-Service.ps1')
}
function Run-SetupMetadataRace([string]$Application,[string]$State) {
    $transaction = $Application+'-upgrade-recovery'
    $control = Join-Path $root 'metadata-race-control'
    New-Item -ItemType Directory -Path $control | Out-Null
    Native icacls @($control,'/grant',('*'+$ordinarySid+':(OI)(CI)M'))
    $outside = Join-Path $root 'metadata-race-outside.txt'
    $actor = Join-Path $control 'metadata-race-actor.ps1'
    $result = Join-Path $control 'metadata-race-result.json'
    [IO.File]::WriteAllText($outside,'outside-sentinel')
    if (Test-Path -LiteralPath $transaction) { Remove-Item -LiteralPath $transaction -Recurse -Force }
    $pending = Join-Path $transaction 'upgrade-pending.json'
    $payload = Join-Path $transaction 'payload'
    $p = $pending.Replace("'","''"); $o = $outside.Replace("'","''"); $r = $result.Replace("'","''"); $u = $ordinarySid.Replace("'","''")
    $script = @'
$attempts=0; $writeSuccess=0; $renameSuccess=0; $linkSuccess=0; $aclSuccess=0
while (Get-Process -Id $args[0] -ErrorAction SilentlyContinue) {
  $attempts++
    try { Add-Content -LiteralPath '__P__' -Value 'race' -ErrorAction Stop; $writeSuccess++ } catch {}
    try {
      Rename-Item -LiteralPath '__P__' -NewName 'upgrade-pending.swap' -ErrorAction Stop
      Rename-Item -LiteralPath (Join-Path (Split-Path '__P__') 'upgrade-pending.swap') -NewName 'upgrade-pending.json' -ErrorAction Stop
      $renameSuccess++
    } catch {}
    try {
      & fsutil.exe hardlink create (Join-Path (Split-Path '__P__') 'payload/actor-link.bin') '__P__' | Out-Null
      if ($LASTEXITCODE -eq 0) { $linkSuccess++ }
    } catch {}
    try {
      & icacls.exe '__P__' /grant ('*'+ '__U__'+':F') | Out-Null
      if ($LASTEXITCODE -eq 0) { $aclSuccess++ }
    } catch {}
  try { Add-Content -LiteralPath '__O__' -Value 'race' -ErrorAction Stop } catch {}
  Start-Sleep -Milliseconds 10
}
[ordered]@{attempts=$attempts;writeSuccess=$writeSuccess;renameSuccess=$renameSuccess;linkSuccess=$linkSuccess;aclSuccess=$aclSuccess} | ConvertTo-Json | Set-Content -LiteralPath '__R__'
'@
    $script.Replace('__P__',$p).Replace('__O__',$o).Replace('__R__',$r).Replace('__U__',$u) | Set-Content -LiteralPath $actor
    $credential = [PSCredential]::new(($env:COMPUTERNAME+'\'+$ordinaryUser),(ConvertTo-SecureString $ordinaryPassword -AsPlainText -Force))
    $positive = Join-Path $control 'permissive-parent'
    New-Item -ItemType Directory -Path $positive | Out-Null
    Native icacls @($positive,'/grant',('*'+$ordinarySid+':(OI)(CI)M'))
    $positiveFile = Join-Path $positive 'positive.txt'
    $positiveSource = Join-Path $positive 'hardlink-source.txt'
    $positiveLink = Join-Path $positive 'hardlink-link.txt'
    [IO.File]::WriteAllText($positiveFile,'positive')
    [IO.File]::WriteAllText($positiveSource,'hardlink')
    $positiveScript = Join-Path $control 'permissive-actor.ps1'
    $pp = $positiveFile.Replace("'","''"); $ps = $positiveSource.Replace("'","''"); $pl = $positiveLink.Replace("'","''"); $pr = $result.Replace("'","''")
    @'
try {
  Add-Content -LiteralPath '__PP__' -Value 'write' -ErrorAction Stop
  Rename-Item -LiteralPath '__PP__' -NewName 'renamed.txt' -ErrorAction Stop
  & fsutil.exe hardlink create '__PL__' '__PS__' | Out-Null
  if ($LASTEXITCODE -ne 0) { throw 'hardlink' }
  Set-Content -LiteralPath '__PR__' -Value 'write-rename-link'
} catch { Set-Content -LiteralPath '__PR__' -Value 'positive-control-failed' }
'@.Replace('__PP__',$pp).Replace('__PS__',$ps).Replace('__PL__',$pl).Replace('__PR__',$pr) | Set-Content -LiteralPath $positiveScript
    $positiveProcess = Start-Process powershell.exe -Credential $credential -ArgumentList @('-NoProfile','-File',('"'+$positiveScript+'"')) -Wait -PassThru
    Assert ($positiveProcess.ExitCode -eq 0 -and (Get-Content -LiteralPath $result -Raw).Trim() -eq 'write-rename-link') 'separate account permissive parent write rename and hardlink positive control'
    Remove-Item -LiteralPath $result -Force
    $process = Start-Process $script:setup -WindowStyle Hidden -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$Application+'"'),('/STATE="'+$State+'"')) -PassThru
    $actorProcess = Start-Process powershell.exe -Credential $credential -ArgumentList @('-NoProfile','-File',('"'+$actor+'"'),[string]$process.Id) -PassThru
    try {
        $process.WaitForExit()
        $process.Refresh()
        if (!$actorProcess.WaitForExit(30000)) { $actorProcess.Kill(); $actorProcess.WaitForExit(); throw 'metadata actor timeout' }
        Assert ($process.ExitCode -eq 0) 'real Setup completes while separate account races recovery metadata'
        Assert (Test-Path -LiteralPath $result) 'separate-account Setup race emitted result'
        $counts = Get-Content -LiteralPath $result -Raw | ConvertFrom-Json
        Assert ($counts.attempts -gt 0 -and $counts.writeSuccess -eq 0 -and $counts.renameSuccess -eq 0 -and $counts.linkSuccess -eq 0 -and $counts.aclSuccess -eq 0) 'protected Setup metadata rejects all separate-account mutation attempts'
        Assert ([IO.File]::ReadAllText($outside) -eq 'outside-sentinel') 'Setup metadata race preserves outside bytes'
        Assert (!(Test-Path -LiteralPath (Join-Path $transaction 'upgrade-pending.json')) -and !(Test-Path -LiteralPath (Join-Path $transaction 'upgrade-terminal.json'))) 'successful Setup retires transaction metadata without stale markers'
    } finally {
        if (!$process.HasExited) { $process.Kill(); $process.WaitForExit() }
        if (!$actorProcess.HasExited) { $actorProcess.Kill(); $actorProcess.WaitForExit() }
    }
}
function Run-UninstallMetadataRace([string]$Application) {
    $control = Join-Path $root 'uninstall-metadata-control'
    $actor = Join-Path $control 'uninstall-metadata-actor.ps1'
    $result = Join-Path $control 'uninstall-metadata-result.json'
    $target = Join-Path $Application 'Manage-Service.ps1'
    $journal = Join-Path $Application 'unins000.dat'
    New-Item -ItemType Directory -Path $control | Out-Null
    Native icacls @($control,'/grant',('*'+$ordinarySid+':(OI)(CI)M'))
    $outside = Join-Path $root 'uninstall-race-outside.txt'
    [IO.File]::WriteAllText($outside,'outside-sentinel')
    $t = $target.Replace("'","''"); $j = $journal.Replace("'","''"); $o = $outside.Replace("'","''"); $r = $result.Replace("'","''")
    $script = @'
$attempts=0; $writeSuccess=0; $renameSuccess=0; $journalSuccess=0; $outsideSuccess=0
while (Get-Process -Id $args[0] -ErrorAction SilentlyContinue) {
  $attempts++
  try { Add-Content -LiteralPath '__T__' -Value 'race' -ErrorAction Stop; $writeSuccess++ } catch {}
  try { Rename-Item -LiteralPath '__T__' -NewName 'Manage-Service.swap' -ErrorAction Stop; $renameSuccess++ } catch {}
  try { Add-Content -LiteralPath '__J__' -Value 'race' -ErrorAction Stop; $journalSuccess++ } catch {}
  try { Add-Content -LiteralPath '__O__' -Value 'race' -ErrorAction Stop; $outsideSuccess++ } catch {}
  Start-Sleep -Milliseconds 10
}
[ordered]@{attempts=$attempts;writeSuccess=$writeSuccess;renameSuccess=$renameSuccess;journalSuccess=$journalSuccess;outsideSuccess=$outsideSuccess} | ConvertTo-Json | Set-Content -LiteralPath '__R__'
'@
    $script.Replace('__T__',$t).Replace('__J__',$j).Replace('__O__',$o).Replace('__R__',$r) | Set-Content -LiteralPath $actor
    $credential = [PSCredential]::new(($env:COMPUTERNAME+'\'+$ordinaryUser),(ConvertTo-SecureString $ordinaryPassword -AsPlainText -Force))
    $trace = Join-Path $root 'uninstall-boundary.log'
    $script:uninstallRaceTrace = $trace
    $process = Start-Process (Join-Path $Application 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/LOG="'+$trace+'"')) -PassThru
    $actorProcess = Start-Process powershell.exe -Credential $credential -ArgumentList @('-NoProfile','-File',('"'+$actor+'"'),[string]$process.Id) -PassThru
    try {
        $process.WaitForExit()
        $process.Refresh()
        $script:uninstallRaceExitCode = $process.ExitCode
        if (!$actorProcess.WaitForExit(30000)) { $actorProcess.Kill(); $actorProcess.WaitForExit(); throw 'uninstall metadata actor timeout' }
        Assert ($process.ExitCode -eq 0) 'real uninstall completes while separate account races metadata'
        $counts = Get-Content -LiteralPath $result -Raw | ConvertFrom-Json
        Assert ($counts.attempts -gt 0 -and $counts.writeSuccess -eq 0 -and $counts.renameSuccess -eq 0 -and $counts.journalSuccess -eq 0 -and $counts.outsideSuccess -eq 0) 'uninstall metadata race rejects separate-account mutations'
        Assert ([IO.File]::ReadAllText($outside) -eq 'outside-sentinel') 'uninstall metadata race preserves outside bytes'
    } finally {
        if (!$process.HasExited) { $process.Kill(); $process.WaitForExit() }
        if (!$actorProcess.HasExited) { $actorProcess.Kill(); $actorProcess.WaitForExit() }
    }
}

$root = Join-Path $env:RUNNER_TEMP ('upgrade preparation '+[guid]::NewGuid().ToString('N')+' '+[char]0x00e9)
$repo = Split-Path $PSScriptRoot -Parent
$currentApp = ''
$programApp = ''
$freshState = ''
$junction = ''
$serviceOwned = $false
$ordinaryUser = 'ICUpgrade'+[guid]::NewGuid().ToString('N').Substring(0,8)
$ordinaryPassword = 'Aa1!'+[guid]::NewGuid().ToString('N')
$ordinaryCreated = $false
Add-Type @'
using System;
using System.IO;
using System.Runtime.InteropServices;
using System.Security.Principal;
public static class UpgradeWriteProbe {
 [DllImport("advapi32.dll", CharSet=CharSet.Unicode, SetLastError=true)] static extern bool LogonUser(string u,string d,string p,int t,int v,out IntPtr token);
 [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr token);
 public static bool CreateOnlyBoundary(string user,string password,string parent,string managed) {
  IntPtr token;
  if(!LogonUser(user,".",password,2,0,out token)) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
  try { using(var identity=new WindowsIdentity(token)) using(var context=identity.Impersonate()) {
   string own=Path.Combine(parent,"actor-owned");
   Directory.CreateDirectory(own);
   File.WriteAllText(Path.Combine(own,"marker"),"actor");
   Directory.Move(own,own+"-renamed");
   try { Directory.Move(managed,managed+"-replaced"); return false; }
   catch(UnauthorizedAccessException) { return true; }
   catch(IOException e) { if((e.HResult & 65535)==5) return true; throw; }
  }} finally { CloseHandle(token); }
 }
 public static bool CanWrite(string user,string password,string path) {
  IntPtr token;
  if(!LogonUser(user,".",password,2,0,out token)) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
  try { using(var identity=new WindowsIdentity(token)) using(var context=identity.Impersonate()) {
   try { using(var f=File.Open(path,FileMode.Open,FileAccess.Write,FileShare.ReadWrite|FileShare.Delete)) {} return true; }
   catch(UnauthorizedAccessException) { return false; }
   catch(IOException e) { if((e.HResult & 65535)==32) return false; throw; }
  }} finally { CloseHandle(token); }
 }
}
'@
try {
    Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue)) 'upgrade preparation probe refuses an existing service'
    Assert (!(Test-Path 'HKLM:\SOFTWARE\ImmichClone')) 'upgrade preparation probe refuses existing application registration'
    Assert (!(Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{5DF7680B-6F42-4BAF-AD97-5A3095728DFA}_is1')) 'upgrade preparation probe refuses existing installer registration'
    New-Item -ItemType Directory -Path $root | Out-Null
    New-LocalUser -Name $ordinaryUser -Password (ConvertTo-SecureString $ordinaryPassword -AsPlainText -Force) -AccountNeverExpires | Out-Null
    $ordinaryCreated = $true
    Add-LocalGroupMember -SID 'S-1-5-32-545' -Member $ordinaryUser
    $ordinarySid = (Get-LocalUser -Name $ordinaryUser).SID.Value
    $installerSid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    Native icacls @($root,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F',('*'+$installerSid+':(OI)(CI)F'),'*S-1-5-32-545:(OI)(CI)RX','*S-1-5-19:(OI)(CI)RX')
    Native icacls @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
    if (!$Compiler) {
        $compilerSetup = Join-Path $root 'innosetup-6.7.4.exe'
        Invoke-WebRequest 'https://github.com/jrsoftware/issrc/releases/download/is-6_7_4/innosetup-6.7.4.exe' -OutFile $compilerSetup
        Assert ((Get-FileHash $compilerSetup).Hash.ToLowerInvariant() -eq 'a549929319f9120716677c685535ca4671e4b937b5fdc87e2585f32d8eb30e8c') 'pinned Inno 6.7.4 compiler archive hash verified'
        $compilerDir = Join-Path $root 'inno-6.7.4'
        $process = Start-Process $compilerSetup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$compilerDir+'"')) -Wait -PassThru -WindowStyle Hidden
        Assert ($process.ExitCode -eq 0) 'pinned Inno compiler provisioned for disposable probe'
        $Compiler = Join-Path $compilerDir 'ISCC.exe'
    }
    # Compile the actual production .iss, not a duplicate of its Pascal logic.
    # Minimal fixture providers are never delivered or uploaded as release artifacts.
    $source = Join-Path $root 'source'
    foreach ($directory in @('packaging','docs','dist','dist/media','dist/ai')) { New-Item -ItemType Directory -Force (Join-Path $source $directory) | Out-Null }
    foreach ($file in @('immichclone.iss','Manage-Service.ps1','Storage-Security.ps1','Export-Diagnostics.ps1')) { Copy-Item -LiteralPath (Join-Path $repo ('packaging/'+$file)) -Destination (Join-Path $source ('packaging/'+$file)) }
    Copy-Item -LiteralPath (Join-Path $repo 'docs/release-notes-0.1.43.md') -Destination (Join-Path $source 'dist/prototype-notes.txt')
    Copy-Item -LiteralPath $Binary -Destination (Join-Path $source 'dist/immichclone.exe')
    foreach ($file in @('dist/THIRD-PARTY-LICENSES.txt','dist/media/preparation-probe.txt','dist/ai/preparation-probe.txt')) { [IO.File]::WriteAllText((Join-Path $source $file),'Disposable upgrade-preparation fixture only; not a distributable package.') }
    Native go @('build','-trimpath','-o',(Join-Path $source 'dist/installer-files.exe'),'./cmd/installerfiles')
    Native $Compiler @('/DAppVersion=0.0.0',(Join-Path $source 'packaging/immichclone.iss'))
    $script:setup = Join-Path $source 'dist/ImmichClone-0.0.0-prototype-x64-setup.exe'
    Assert (Test-Path -LiteralPath $script:setup) 'production Inno script compiled with minimal disposable payload'

    # A static junction must abort before touching retained app files. The old
    # helper would write a harmless marker if it were mistakenly invoked.
    $negativeApp = Join-Path $root 'rejected application'
    $outside = Join-Path $root 'outside'
    New-Item -ItemType Directory -Path $negativeApp,$outside | Out-Null
    Write-RetainedHelper $negativeApp
    $oldBinary = Join-Path $negativeApp 'immichclone.exe'
    [IO.File]::WriteAllText($oldBinary,'retained binary sentinel')
    $oldBinaryHash = (Get-FileHash $oldBinary).Hash
    $oldHelperHash = (Get-FileHash (Join-Path $negativeApp 'Manage-Service.ps1')).Hash
    $sentinel = Join-Path $outside 'sentinel.txt'; [IO.File]::WriteAllText($sentinel,'outside sentinel')
    $outsideAcl = (Get-Acl -LiteralPath $outside).Sddl
    $junction = Join-Path $root 'redirected state'
    New-Item -ItemType Junction -Path $junction -Target $outside | Out-Null
    Assert ((Run-Setup $negativeApp $junction) -ne 0) 'actual Setup rejects junction state during bundled preparation'
    Assert (!(Test-Path (Join-Path $negativeApp 'retained-preparer-ran.txt'))) 'rejected preparation never executes retained old helper'
    Assert ((Get-FileHash $oldBinary).Hash -eq $oldBinaryHash -and (Get-FileHash (Join-Path $negativeApp 'Manage-Service.ps1')).Hash -eq $oldHelperHash -and !(Test-Path (Join-Path $negativeApp 'Storage-Security.ps1'))) 'preparation failure publishes no replacement application files'
    Assert ((Get-Acl -LiteralPath $outside).Sddl -eq $outsideAcl -and [IO.File]::ReadAllText($sentinel) -eq 'outside sentinel' -and @(Get-ChildItem -LiteralPath $outside -Force).Count -eq 1) 'rejected Setup leaves junction target ACL and bytes unchanged'
    Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue) -and !(Test-Path 'HKLM:\SOFTWARE\ImmichClone')) 'rejected preparation registers no service or state'
    [IO.Directory]::Delete($junction); $junction = ''

    # An actor-writable parent is an invalid installer namespace. The actual
    # Inno Setup must refuse before creating a managed app, state or service.
    $unsafeParent = Join-Path $root 'actor-writable-parent'
    $unsafeApp = Join-Path $unsafeParent 'unsafe application'
    $unsafeState = Join-Path $unsafeParent 'unsafe state'
    New-Item -ItemType Directory -Path $unsafeParent | Out-Null
    Native icacls @($unsafeParent,'/grant',('*'+$ordinarySid+':(OI)(CI)M'))
    $unsafeResult = Run-Setup $unsafeApp $unsafeState
    Assert ($unsafeResult -ne 0) 'actual Setup refuses an actor-writable application namespace before privileged writes'
    Assert (!(Test-Path -LiteralPath (Join-Path $unsafeApp 'immichclone.exe')) -and !(Test-Path -LiteralPath $unsafeState) -and !(Test-Path -LiteralPath ($unsafeApp+'-upgrade-recovery')) -and !(Get-Service ImmichClone -ErrorAction SilentlyContinue)) 'unsafe namespace refusal leaves no managed app, state, recovery metadata or service'

    # Standard ProgramData-like creation rights permit actor-owned siblings but
    # grant no replacement authority over an existing private managed child.
    $createParent = Join-Path $root 'create-only-parent'
    New-Item -ItemType Directory -Path $createParent | Out-Null
    $createAcl = New-Object Security.AccessControl.DirectorySecurity
    $createAcl.SetSecurityDescriptorSddlForm(('D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;'+$installerSid+')(A;;0x1201bf;;;BU)(A;OICIIO;GA;;;CO)'))
    Set-Acl -LiteralPath $createParent -AclObject $createAcl
    $createChild = Join-Path $createParent 'private-managed'
    $nativeHelper = Join-Path $source 'dist/installer-files.exe'
    Native $nativeHelper @('-action','audit-namespace','-app-dir',$createChild,'-data-dir',(Join-Path $createParent 'state'),'-root-dir',(Join-Path $createParent 'recovery'))
    Native $nativeHelper @('-action','ensure-private-root','-root-dir',$createChild)
    Assert ([UpgradeWriteProbe]::CreateOnlyBoundary($ordinaryUser,$ordinaryPassword,$createParent,$createChild)) 'separate standard account creates and renames its sibling but cannot replace private managed child'
    Assert (Test-Path -LiteralPath $createChild) 'create-only actor control retains managed root identity'

    # Create an actual service-owned existing library without using the retained
    # helper. It deliberately represents old files, not an older runtime version.
    $currentApp = Join-Path $root 'existing application'
    $state = Join-Path $root 'existing state'
    New-Item -ItemType Directory -Path $currentApp | Out-Null
    Copy-Item -LiteralPath $Binary -Destination (Join-Path $currentApp 'immichclone.exe')
    Write-RetainedHelper $currentApp
    New-Item -ItemType Directory -Path $state | Out-Null
    Native icacls @($state,'/grant','*S-1-5-19:(OI)(CI)M')
    $credential = New-Object PSCredential('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
    New-Service -Name ImmichClone -BinaryPathName ('"'+(Join-Path $currentApp 'immichclone.exe')+'" -mode service -data "'+$state+'"') -Credential $credential | Out-Null
    $serviceOwned = $true
    Start-Service ImmichClone
    Wait-Health
    $code = ([IO.File]::ReadAllText((Join-Path $state 'access-code.txt'))).Trim()
    $codeHash = (Get-FileHash (Join-Path $state 'access-code.txt')).Hash
    $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    $original = Join-Path $root 'synthetic-original.bin'; [IO.File]::WriteAllText($original,'upgrade preparation synthetic original')
    $asset = (Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=fixture.bin' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $original -WebSession $session).asset
    $originalHash = (Get-FileHash $original).Hash
    $priorBinaryHash = (Get-FileHash (Join-Path $currentApp 'immichclone.exe')).Hash
    Assert-TrustedCodeTree $currentApp
    Assert $true 'existing application passes the production code trust audit'
    Assert ((Run-Setup $currentApp $state -ObserveStaging) -eq 0) 'actual Setup succeeds using bundled preparation against existing LocalService state'
    Wait-Health
    Assert (!(Test-Path (Join-Path $currentApp 'retained-preparer-ran.txt'))) 'successful preparation never executes retained old helper'
    Assert ((Get-FileHash (Join-Path $currentApp 'rollback/immichclone.exe')).Hash -eq $priorBinaryHash -and (Get-FileHash (Join-Path $currentApp 'rollback/access-code.txt')).Hash -eq $codeHash -and (Test-Path (Join-Path $currentApp 'rollback/catalog.db'))) 'bundled preparation snapshots real prior binary catalog and access code'
    Assert ((Get-FileHash (Join-Path $currentApp 'Manage-Service.ps1')).Hash -eq (Get-FileHash (Join-Path $source 'packaging/Manage-Service.ps1')).Hash -and (Get-FileHash (Join-Path $currentApp 'Storage-Security.ps1')).Hash -eq (Get-FileHash (Join-Path $source 'packaging/Storage-Security.ps1')).Hash) 'Setup installs exact candidate management and sibling helper bytes'
    Assert ((Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash -and (Get-FileHash (Join-Path $state 'access-code.txt')).Hash -eq $codeHash) 'bundled preparation preserves managed original and access code'
    Assert ((Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").StartName -eq 'NT AUTHORITY\LocalService') 'prepared service remains LocalService'
    Assert-PrivateDirectoryTree -Path (Join-Path $currentApp 'rollback') -OwnerSid $installerSid -VerifyInherited
    Assert $true 'actual Setup code read grant preserves admin-only rollback snapshots and access-code privacy'
    Assert ((Get-ItemProperty 'HKLM:\SOFTWARE\ImmichClone').State -eq $state) 'Setup retains real application data location'
    $payloadNames = @('Manage-Service.ps1','Storage-Security.ps1','installer-files.exe','immichclone.exe','Export-Diagnostics.ps1','THIRD-PARTY-LICENSES.txt','prototype-notes.txt','media/preparation-probe.txt','ai/preparation-probe.txt')
    $installedPayloadHashes = @{}
    foreach ($leaf in $payloadNames) { $installedPayloadHashes[$leaf] = (Get-FileHash -LiteralPath (Join-Path $currentApp $leaf)).Hash }
    Assert $true 'actual Setup publishes every fixed root and nested provider fixture through the native payload helper'
    # Keep installed source ACLs untouched. An exact byte copy of its stopped
    # library is separately configured RX-only for a real non-administrator.
    # Administrator CLI commands also exercise the original installed library.
    $offlineInstalledExe = Join-Path $currentApp 'immichclone.exe'
    $offlineExe = Join-Path $root 'offline-current-runtime.exe'
    $offlineSource = Join-Path $root 'ordinary readonly library copy'
    $offlineOutput = Join-Path $root 'ordinary offline output'
    $offlineLogs = Join-Path $offlineOutput 'logs must remain absent'
    $offlineArchive = Join-Path $offlineOutput 'independent-readonly.zip'
    $offlineAdminArchive = Join-Path $offlineOutput 'administrator-original.zip'
    $offlineRestored = Join-Path $root 'independent readonly recovery'
    $offlineInstalledExeAcl = (Get-Acl -LiteralPath $offlineInstalledExe).Sddl
    $offlineCredential = [Management.Automation.PSCredential]::new(($env:COMPUTERNAME+'\'+$ordinaryUser),(ConvertTo-SecureString $ordinaryPassword -AsPlainText -Force))
    $script:offlineAttempt = 0
    function Offline-StateSnapshot([string]$Path) {
        $records = @()
        $items = @((Get-Item -LiteralPath $Path -Force)) + @(Get-ChildItem -LiteralPath $Path -Force -Recurse)
        foreach ($item in ($items | Sort-Object FullName)) {
            if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) { throw 'Read-only installed fixture unexpectedly contains a reparse point' }
            $record = $item.FullName+'|'+(Get-Acl -LiteralPath $item.FullName).Sddl
            if (!$item.PSIsContainer) { $record += '|'+(Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash }
            $records += $record
        }
        return $records -join "`n"
    }
    function Run-OfflineProbe([string]$Mode,[string]$SelectedState,[string]$SelectedArchive,[Management.Automation.PSCredential]$Credential) {
        $script:offlineAttempt++
        $stdout = Join-Path $offlineOutput ('stdout-'+$script:offlineAttempt+'.txt')
        $stderr = Join-Path $offlineOutput ('stderr-'+$script:offlineAttempt+'.txt')
        $arguments = @('-mode',$Mode,'-data',('"'+$SelectedState+'"'),'-logs',('"'+$offlineLogs+'"'))
        if ($SelectedArchive) { $arguments += @('-archive',('"'+$SelectedArchive+'"')) }
        $options = @{FilePath=$offlineExe;ArgumentList=$arguments;PassThru=$true;WorkingDirectory=$offlineOutput;RedirectStandardOutput=$stdout;RedirectStandardError=$stderr}
        if ($Credential) { $options.Credential = $Credential }
        $process = Start-Process @options
        $null = $process.Handle
        try {
            if (!$process.WaitForExit(60000)) { $process.Kill(); $process.WaitForExit(); throw 'Offline installed command exceeded the bounded fixture timeout' }
            $process.Refresh(); $process.WaitForExit()
            $status = $process.ExitCode
            if ($null -eq $status) { throw 'Offline installed command completed without a captured exit status' }
            return @{Code=$status;Output=[IO.File]::ReadAllText($stdout).Trim();Error=[IO.File]::ReadAllText($stderr).Trim()}
        } finally { $process.Dispose() }
    }
    Stop-Service ImmichClone
    (Get-Service ImmichClone).WaitForStatus('Stopped',[TimeSpan]::FromSeconds(30))
    $offlineInstalledBefore = Offline-StateSnapshot $state
    try {
        New-Item -ItemType Directory -Path $offlineOutput | Out-Null
        Native icacls @($offlineOutput,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F',('*'+$ordinarySid+':(OI)(CI)F'))
        Copy-Item -LiteralPath $offlineInstalledExe -Destination $offlineExe
        Assert ((Get-FileHash -LiteralPath $offlineExe).Hash -eq (Get-FileHash -LiteralPath $offlineInstalledExe).Hash) 'offline process uses exact installed runtime bytes without changing installed executable permissions'
        Native icacls @($offlineExe,'/grant',('*'+$ordinarySid+':RX'))
        $adminVerified = Run-OfflineProbe 'verify' $state '' $null
        Assert ($adminVerified.Code -eq 0 -and $adminVerified.Output -eq 'All catalogued originals passed SHA256 verification' -and !$adminVerified.Error) 'administrator verifies the original installed library without source repair or writes'
        $adminBackedUp = Run-OfflineProbe 'backup' $state $offlineAdminArchive $null
        Assert ($adminBackedUp.Code -eq 0 -and $adminBackedUp.Output -eq 'Consistent backup created' -and !$adminBackedUp.Error) 'administrator backs up the original installed library without modifying source state'
        Assert ((Offline-StateSnapshot $state) -eq $offlineInstalledBefore -and !(Test-Path -LiteralPath $offlineLogs)) 'administrator verify and backup preserve original installed source hashes descriptors namespace and selected logs'
        Copy-Item -LiteralPath $state -Destination $offlineSource -Recurse
        $installedItems = @((Get-Item -LiteralPath $state -Force)) + @(Get-ChildItem -LiteralPath $state -Force -Recurse)
        $copiedItems = @((Get-Item -LiteralPath $offlineSource -Force)) + @(Get-ChildItem -LiteralPath $offlineSource -Force -Recurse)
        Assert ($installedItems.Count -eq $copiedItems.Count) 'RX fixture copy preserves the complete installed library namespace'
        foreach ($item in $installedItems) {
            $relative = $item.FullName.Substring($state.Length).TrimStart('\')
            $copyPath = $offlineSource
            if ($relative) { $copyPath = Join-Path $offlineSource $relative }
            Assert ((Test-Path -LiteralPath $copyPath) -and (Get-Item -LiteralPath $copyPath -Force).PSIsContainer -eq $item.PSIsContainer) 'RX fixture copy preserves each installed library object type'
            if (!$item.PSIsContainer) { Assert ((Get-FileHash -LiteralPath $copyPath).Hash -eq (Get-FileHash -LiteralPath $item.FullName).Hash) 'RX fixture copy preserves each installed library file byte before permission setup' }
        }
        foreach ($item in (@((Get-Item -LiteralPath $offlineSource -Force)) + @(Get-ChildItem -LiteralPath $offlineSource -Force -Recurse) | Sort-Object FullName)) {
            $cloneAcl = Get-Acl -LiteralPath $item.FullName
            if ($item.PSIsContainer) { $clonePolicy = 'D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;'+$ordinarySid+')' }
            else { $clonePolicy = 'D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;0x1200a9;;;'+$ordinarySid+')' }
            $cloneAcl.SetSecurityDescriptorSddlForm($clonePolicy,[Security.AccessControl.AccessControlSections]::Access)
            Set-Acl -LiteralPath $item.FullName -AclObject $cloneAcl
        }
        Assert (![UpgradeWriteProbe]::CanWrite($ordinaryUser,$ordinaryPassword,(Join-Path $offlineSource 'catalog.db')) -and ![UpgradeWriteProbe]::CanWrite($ordinaryUser,$ordinaryPassword,(Join-Path $offlineSource ('originals/'+$asset.id)))) 'actual offline account cannot modify copied source catalog or original with its RX-only policy'
        $offlineBefore = Offline-StateSnapshot $offlineSource
        $verified = Run-OfflineProbe 'verify' $offlineSource '' $offlineCredential
        Assert ($verified.Code -eq 0 -and $verified.Output -eq 'All catalogued originals passed SHA256 verification' -and !$verified.Error) 'separate RX-only non-administrator verifies the copied installed library successfully'
        $backedUp = Run-OfflineProbe 'backup' $offlineSource $offlineArchive $offlineCredential
        Assert ($backedUp.Code -eq 0 -and $backedUp.Output -eq 'Consistent backup created' -and !$backedUp.Error) 'separate RX-only non-administrator backs up the copied installed library without source write authority'
        Assert ((Offline-StateSnapshot $offlineSource) -eq $offlineBefore -and (Offline-StateSnapshot $state) -eq $offlineInstalledBefore -and !(Test-Path -LiteralPath $offlineLogs)) 'actual RX verify and backup preserve all source hashes descriptors namespace and create no selected diagnostic directory'
        $archiveAcl = Get-Acl -LiteralPath $offlineArchive
        $archiveRaw = [Security.AccessControl.RawSecurityDescriptor]::new($archiveAcl.GetSecurityDescriptorBinaryForm(),0)
        Assert (($archiveRaw.ControlFlags -band [Security.AccessControl.ControlFlags]::DiscretionaryAclProtected) -ne 0) 'ordinary-user-created archive has a protected private descriptor'
        foreach ($ace in $archiveRaw.DiscretionaryAcl) {
            Assert (($ace -is [Security.AccessControl.CommonAce]) -and $ace.AceQualifier -eq [Security.AccessControl.AceQualifier]::AccessAllowed -and $ace.SecurityIdentifier.Value -in @('S-1-5-18','S-1-5-32-544',$ordinarySid)) 'private archive grants access only to its creating user administrators and SYSTEM'
        }
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $zip = [IO.Compression.ZipFile]::OpenRead($offlineArchive)
        try {
            $manifestEntry = $zip.GetEntry('manifest.json')
            Assert ($null -ne $manifestEntry) 'ordinary-user backup has an independent recovery manifest'
            $manifestReader = [IO.StreamReader]::new($manifestEntry.Open())
            try { $manifest = $manifestReader.ReadToEnd() | ConvertFrom-Json } finally { $manifestReader.Dispose() }
            Assert ($manifest.schema -eq 1 -and @($manifest.sha256.PSObject.Properties).Count -eq 4 -and $zip.Entries.Count -eq 5) 'ordinary-user backup contains exactly the known installed library recovery set'
            foreach ($property in $manifest.sha256.PSObject.Properties) {
                $entry = $zip.GetEntry($property.Name)
                Assert ($null -ne $entry) 'manifest names an existing archived recovery entry'
                $stream = $entry.Open(); $algorithm = [Security.Cryptography.SHA256]::Create()
                try { $hash = [BitConverter]::ToString($algorithm.ComputeHash($stream)).Replace('-','').ToLowerInvariant() } finally { $algorithm.Dispose(); $stream.Dispose() }
                Assert ($hash -eq $property.Value) 'independent ZIP readback matches the archived recovery checksum'
            }
            Assert ($manifest.sha256.('originals/'+$asset.id) -eq $originalHash.ToLowerInvariant()) 'independent ZIP readback retains the exact uploaded original checksum'
        } finally { $zip.Dispose() }
        $archiveHashBeforeRestore = (Get-FileHash -LiteralPath $offlineArchive).Hash
        $restored = Run-OfflineProbe 'restore' $offlineRestored $offlineArchive $null
        Assert ($restored.Code -eq 0 -and $restored.Output -eq 'Restore verified and published; previews queued for regeneration') 'existing administrator recovery workflow restores the ordinary-user read-only backup into a distinct new library'
        $restoredVerified = Run-OfflineProbe 'verify' $offlineRestored '' $null
        Assert ($restoredVerified.Code -eq 0 -and $restoredVerified.Output -eq 'All catalogued originals passed SHA256 verification') 'administrator recovery verification remains compatible with the read-only command path'
        Assert ((Get-FileHash -LiteralPath $offlineArchive).Hash -eq $archiveHashBeforeRestore -and (Get-FileHash -LiteralPath (Join-Path $offlineRestored ('originals/'+$asset.id))).Hash -eq $originalHash -and (Offline-StateSnapshot $offlineSource) -eq $offlineBefore -and (Offline-StateSnapshot $state) -eq $offlineInstalledBefore) 'independent restore preserves the archive original installed library RX source copy and exact restored original bytes'
    } finally {
        Assert ((Offline-StateSnapshot $state) -eq $offlineInstalledBefore -and (Get-Acl -LiteralPath $offlineInstalledExe).Sddl -eq $offlineInstalledExeAcl) 'offline fixture leaves every original installed source descriptor hash directory entry and executable permission unchanged before resuming service'
        Start-Service ImmichClone
        Wait-Health
    }
    # Stationary metadata aliases must be refused BEFORE Inno overwrites its
    # retained journal, not merely noticed by the later code permission phase.
    $journal = Join-Path $currentApp 'unins000.dat'
    $journalOutside = Join-Path $root 'outside-valid-uninstaller-journal.dat'
    $journalAcl = Get-Acl -LiteralPath $journal
    Copy-Item -LiteralPath $journal -Destination $journalOutside
    $journalHash = (Get-FileHash -LiteralPath $journalOutside).Hash
    $metadataPid = (Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").ProcessId
    Remove-Item -LiteralPath $journal
    New-Item -ItemType HardLink -Path $journal -Target $journalOutside | Out-Null
    try {
        Assert ((Run-Setup $currentApp $state) -ne 0) 'actual Setup refuses a hard-linked retained Inno metadata journal before publication'
        Assert ((Get-FileHash -LiteralPath $journalOutside).Hash -eq $journalHash -and (Get-FileHash -LiteralPath $journal).Hash -eq $journalHash) 'refused metadata preparation leaves outside and selected journal bytes unchanged'
        Assert ((Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").ProcessId -eq $metadataPid) 'hard-linked metadata refusal precedes service stop and restart'
        Assert ((Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash) 'metadata alias refusal preserves managed original bytes'
    } finally {
        Remove-Item -LiteralPath $journal
        Copy-Item -LiteralPath $journalOutside -Destination $journal
        Set-Acl -LiteralPath $journal -AclObject $journalAcl
    }
    # A stopped-service native ACL acquisition can still meet an unrelated held
    # writer. Refuse before permission writes, verify final policy, and restart
    # the prior service honestly rather than leaving a healthy installation down.
    $aclWriterPath = Join-Path $state 'acl-acquisition-writer.tmp'
    [IO.File]::WriteAllText($aclWriterPath,'synthetic ACL sharing control')
    $aclRootBefore = (Get-Acl -LiteralPath $state).Sddl
    $aclWriter = [IO.File]::Open($aclWriterPath,[IO.FileMode]::Open,[IO.FileAccess]::ReadWrite,[IO.FileShare]::None)
    try {
        Assert ((Run-Setup $currentApp $state) -ne 0) 'actual Setup refuses native ACL acquisition against an unrelated exclusive writer'
        Wait-Health
        Assert ((Get-Acl -LiteralPath $state).Sddl -eq $aclRootBefore) 'failed native ACL acquisition preserves root descriptor and restarts safe prior service'
        Assert ((Get-FileHash (Join-Path $currentApp 'immichclone.exe')).Hash -eq $priorBinaryHash -and (Get-FileHash (Join-Path $state 'access-code.txt')).Hash -eq $codeHash -and (Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash) 'ACL phase refusal preserves prior code access code and original bytes'
    } finally { $aclWriter.Dispose(); Remove-Item -LiteralPath $aclWriterPath }

    # Exercise the real management catch path too: snapshot must be complete,
    # and only the verified native restore permits restart/rollback_complete.
    $installerLog = Join-Path $env:ProgramData 'ImmichCloneDiagnostics/installer.jsonl'
    $rollbackBeforeCount = @(Get-Content -LiteralPath $installerLog | Where-Object { $_ -match '"event":"rollback_complete"' }).Count
    # A hard-linked optional failure note must be refused without suppressing
    # the real restore path or changing the outside sentinel.
    $noteOutside = Join-Path $root 'outside-note-sentinel.txt'
    [IO.File]::WriteAllText($noteOutside,'outside diagnostic sentinel')
    Set-Acl -LiteralPath $noteOutside -AclObject (Get-Acl -LiteralPath (Join-Path $currentApp 'immichclone.exe'))
    $noteHash = (Get-FileHash -LiteralPath $noteOutside).Hash
    $notePath = Join-Path $currentApp 'install-diagnostic.txt'
    New-Item -ItemType HardLink -Path $notePath -Target $noteOutside | Out-Null
    $notePid = (Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").ProcessId
    Assert ((Run-Setup $currentApp $state) -ne 0) 'actual preparation refuses a hard-linked optional note before service stop'
    Assert ((Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").ProcessId -eq $notePid -and (Get-FileHash -LiteralPath $noteOutside).Hash -eq $noteHash) 'early optional-note alias refusal preserves prior service and outside bytes'
    Remove-Item -LiteralPath $notePath
    Run-SetupMetadataRace $currentApp $state
    $healthFailureBefore = @(Get-Content -LiteralPath $installerLog | ForEach-Object { $_ | ConvertFrom-Json } | Where-Object { $_.event -eq 'installer_failed' -and $_.stage -eq 'verify service health' }).Count
    try {
        $env:IMMICH_INSTALL_TEST_FAILURE = '1'
        Assert ((Run-Setup $currentApp $state) -ne 0) 'actual candidate Setup rejects injected post-initialization failure'
    } finally { Remove-Item Env:IMMICH_INSTALL_TEST_FAILURE -ErrorAction SilentlyContinue }
    Wait-Health
    Assert ((Get-FileHash -LiteralPath $noteOutside).Hash -eq $noteHash) 'injected post-health rollback preserves prior outside sentinel bytes'
    $healthFailureAfter = @(Get-Content -LiteralPath $installerLog | ForEach-Object { $_ | ConvertFrom-Json } | Where-Object { $_.event -eq 'installer_failed' -and $_.stage -eq 'verify service health' }).Count
    Assert ($healthFailureAfter -eq $healthFailureBefore+1) 'actual injected failure reaches verified service health before rollback rather than passing on an earlier refusal'
    $rollbackAfterCount = @(Get-Content -LiteralPath $installerLog | Where-Object { $_ -match '"event":"rollback_complete"' }).Count
    Assert ($rollbackAfterCount -eq $rollbackBeforeCount+1) 'verified native restore completes before management reports rollback and service health'
    Assert ((Get-FileHash (Join-Path $currentApp 'immichclone.exe')).Hash -eq $priorBinaryHash -and (Get-FileHash (Join-Path $state 'access-code.txt')).Hash -eq $codeHash -and (Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash) 'actual failed Setup restores code and preserves access code and original bytes'
    foreach ($leaf in $payloadNames) { Assert ((Get-FileHash -LiteralPath (Join-Path $currentApp $leaf)).Hash -eq $installedPayloadHashes[$leaf]) 'failed Setup retains verified prior scripts runtime and provider bytes' }
    $snapshotReady = Get-Content -LiteralPath (Join-Path $currentApp 'rollback/snapshot-ready.json') -Raw | ConvertFrom-Json
    Assert ($snapshotReady.Version -eq 1 -and @($snapshotReady.Files).Count -eq 3) 'real Setup rollback uses a complete three-file readiness receipt'

    # Execute the real candidate installer against an existing healthy service.
    # Opening a file for write is the standard-user positive control; no code
    # bytes are changed or arbitrary elevated payload executed.
    $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
    # Wait only for this fixture's existing preview to settle before comparing
    # consistent logical exports. Do not read bbolt's live exclusively locked file.
    for ($i=0;$i -lt 30;$i++) {
        $listed = Invoke-RestMethod http://127.0.0.1:2283/api/assets -WebSession $session
        $assetState = @($listed | Where-Object { $_.id -eq $asset.id })
        if ($assetState.Count -eq 1 -and $assetState[0].preview -ne 'queued') { break }
        Start-Sleep -Milliseconds 200
    }
    Assert ($assetState.Count -eq 1 -and $assetState[0].preview -ne 'queued') 'fixture preview settled before logical catalog preservation checks'
    $codeProbe = Join-Path $currentApp 'ai/trust-probe.exe'
    Copy-Item -LiteralPath $Binary -Destination $codeProbe
    foreach ($entry in @('', 'Manage-Service.ps1', 'ai/trust-probe.exe', 'media')) {
        $unsafe = $currentApp
        if ($entry) { $unsafe = Join-Path $currentApp $entry }
        $savedAcl = Get-Acl -LiteralPath $unsafe
        try {
            $acl = Get-Acl -LiteralPath $unsafe
            $flags = [Security.AccessControl.InheritanceFlags]::None
            if ((Get-Item -LiteralPath $unsafe).PSIsContainer) { $flags = [Security.AccessControl.InheritanceFlags]'ContainerInherit,ObjectInherit' }
            $rule = New-Object Security.AccessControl.FileSystemAccessRule((New-Object Security.Principal.SecurityIdentifier($ordinarySid)),[Security.AccessControl.FileSystemRights]::Modify,$flags,[Security.AccessControl.PropagationFlags]::None,[Security.AccessControl.AccessControlType]::Allow)
            $acl.AddAccessRule($rule)
            Set-Acl -LiteralPath $unsafe -AclObject $acl
            $writable = $unsafe
            if ((Get-Item -LiteralPath $unsafe).PSIsContainer) {
                $writable = Join-Path $unsafe 'ordinary-write-control.bin'
                [IO.File]::WriteAllText($writable,'synthetic code-tree permission fixture')
            }
            Assert ([UpgradeWriteProbe]::CanWrite($ordinaryUser,$ordinaryPassword,$writable)) 'real standard-user write positive control for unsafe code permissions'
            $pidBefore = (Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").ProcessId
            $binaryBefore = (Get-FileHash (Join-Path $currentApp 'immichclone.exe')).Hash
            $catalogBefore = (Invoke-WebRequest http://127.0.0.1:2283/api/export -WebSession $session -UseBasicParsing).Content
            $rollbackBefore = (Get-FileHash (Join-Path $currentApp 'rollback/catalog.db')).Hash
            Assert ((Run-Setup $currentApp $state) -ne 0) 'actual Setup refuses unsafe root script executable or provider permissions'
            Assert ((Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").ProcessId -eq $pidBefore) 'unsafe code refusal precedes service stop or restart'
            Assert ((Get-FileHash (Join-Path $currentApp 'immichclone.exe')).Hash -eq $binaryBefore -and (Invoke-WebRequest http://127.0.0.1:2283/api/export -WebSession $session -UseBasicParsing).Content -eq $catalogBefore -and (Get-FileHash (Join-Path $currentApp 'rollback/catalog.db')).Hash -eq $rollbackBefore) 'unsafe code refusal preserves executable consistent catalog export and rollback bytes'
        } finally {
            Set-Acl -LiteralPath $unsafe -AclObject $savedAcl
            if ($writable -and $writable.EndsWith('ordinary-write-control.bin')) { Remove-Item -LiteralPath $writable -Force -ErrorAction SilentlyContinue }
        }
    }
    Remove-Item -LiteralPath $codeProbe -Force
    Assert (![UpgradeWriteProbe]::CanWrite($ordinaryUser,$ordinaryPassword,(Join-Path $currentApp 'Manage-Service.ps1'))) 'normal installed helper denies real standard-user modification'
    Assert ((Run-Setup $currentApp $state -ObserveStaging) -eq 0) 'protected custom application remains upgradeable after negative controls'
    Wait-Health
    # Enlarge only the final fixture candidate to expose a real native copy
    # window. This is synthetic zero-filled test data, never a release provider.
    $copyWindow = Join-Path $source 'dist/ai/recovery-window.bin'
    $stream = [IO.File]::Open($copyWindow,[IO.FileMode]::CreateNew,[IO.FileAccess]::Write,[IO.FileShare]::None)
    try { $stream.SetLength(1GB); $stream.Flush($true) } finally { $stream.Dispose() }
    Native $Compiler @('/DAppVersion=0.0.0',(Join-Path $source 'packaging/immichclone.iss'))
    Assert ((Run-Setup $currentApp $state -InterruptPublication) -eq 73) 'actual production Setup was terminated during native payload publication'
    $pendingRoot = $currentApp+'-upgrade-recovery'
    $pendingIntent = Join-Path $pendingRoot 'payload/publication-intent.json'
    Assert (![UpgradeWriteProbe]::CanWrite($ordinaryUser,$ordinaryPassword,$pendingIntent)) 'persistent interrupted-recovery intent denies actual standard-user modification'
    $savedCatalogHash = (Get-FileHash -LiteralPath (Join-Path $currentApp 'rollback/catalog.db')).Hash
    $savedExecutableHash = (Get-FileHash -LiteralPath (Join-Path $currentApp 'rollback/immichclone.exe')).Hash
    Assert ((Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").StartMode -eq 'Disabled') 'terminated upgrade cannot automatically start its mixed code on reboot'
    Assert ((Run-Setup $currentApp $state) -eq 0) 'new bundled Setup recovers pending publication before taking a new snapshot and completes retry'
    Wait-Health
    Assert ((Get-FileHash -LiteralPath (Join-Path $currentApp 'rollback/catalog.db')).Hash -eq $savedCatalogHash -and (Get-FileHash -LiteralPath (Join-Path $currentApp 'rollback/immichclone.exe')).Hash -eq $savedExecutableHash) 'retry snapshots restored prior catalog and executable rather than interrupted candidate bytes'
    Assert ((Get-FileHash -LiteralPath (Join-Path $state 'access-code.txt')).Hash -eq $codeHash -and (Get-FileHash -LiteralPath (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash) 'killed Setup and retry preserve independent access-code and original hashes'
    Assert ((Get-FileHash -LiteralPath (Join-Path $currentApp 'ai/recovery-window.bin')).Hash -eq (Get-FileHash -LiteralPath $copyWindow).Hash) 'retry publishes and verifies the complete synthetic large payload'
    Assert (@(Get-ChildItem -LiteralPath (Join-Path $pendingRoot 'payload') -Force).Count -eq 0 -and !(Test-Path (Join-Path $pendingRoot 'upgrade-pending.json')) -and !(Test-Path (Join-Path $pendingRoot 'upgrade-terminal.json'))) 'healthy commit retires saved payload and transaction metadata'
    Assert ((Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").StartMode -eq 'Auto') 'automatic startup returns only after healthy transaction commit'
    # Real uninstaller refusals must happen before service stop, metadata removal,
    # retained-script execution or any app/data deletion.
    $uninstallMarker = Join-Path $currentApp 'unexpected-uninstall-script.txt'
    $helperPath = Join-Path $currentApp 'Manage-Service.ps1'
    $helperBytes = [IO.File]::ReadAllBytes($helperPath)
    $providerPath = Join-Path $currentApp 'ai/preparation-probe.txt'
    $providerBytes = [IO.File]::ReadAllBytes($providerPath)
    $providerAcl = Get-Acl -LiteralPath $providerPath
    foreach ($kind in @('changed-script','provider-hardlink','provider-writer')) {
        $pidBefore = (Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").ProcessId
        $uninstallLogHash = (Get-FileHash -LiteralPath (Join-Path $currentApp 'unins000.dat')).Hash
        try {
            if ($kind -eq 'changed-script') {
                [IO.File]::WriteAllText($helperPath,('[IO.File]::WriteAllText("'+$uninstallMarker+'","unexpected execution")'))
            } elseif ($kind -eq 'provider-hardlink') {
                Remove-Item -LiteralPath $providerPath
                New-Item -ItemType HardLink -Path $providerPath -Target $noteOutside | Out-Null
            } else {
                Native icacls @($providerPath,'/grant','*S-1-5-32-545:F')
                Assert ([UpgradeWriteProbe]::CanWrite($ordinaryUser,$ordinaryPassword,$providerPath)) 'uninstall unsafe provider has actual standard-user write positive control'
            }
            $refused = Start-Process (Join-Path $currentApp 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru
            Assert ($refused.ExitCode -ne 0) 'actual uninstaller refuses modified bootstrap unsafe ACL or hard-linked provider'
            Assert ((Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").ProcessId -eq $pidBefore) 'unsafe uninstall refuses before service stop or registration deletion'
            Assert (!(Test-Path -LiteralPath $uninstallMarker) -and (Test-Path 'HKLM:\SOFTWARE\ImmichClone') -and (Get-FileHash -LiteralPath (Join-Path $currentApp 'unins000.dat')).Hash -eq $uninstallLogHash) 'refused uninstall executes no changed script and preserves uninstall metadata and registration'
            Assert ((Get-FileHash -LiteralPath $noteOutside).Hash -eq $noteHash -and (Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash -and (Test-Path (Join-Path $currentApp 'immichclone.exe'))) 'refused uninstall preserves outside bytes original and earlier payload files'
        } finally {
            [IO.File]::WriteAllBytes($helperPath,$helperBytes)
            if ($kind -eq 'provider-hardlink') { Remove-Item -LiteralPath $providerPath; [IO.File]::WriteAllBytes($providerPath,$providerBytes) }
            Set-Acl -LiteralPath $providerPath -AclObject $providerAcl
        }
    }
    $unknownRoot = Join-Path $currentApp 'unrelated-preserved.txt'
    $unknownRollback = Join-Path $currentApp 'rollback/unrelated-preserved.txt'
    foreach ($path in @($unknownRoot,$unknownRollback)) { [IO.File]::WriteAllText($path,'unrelated retained bytes') }
    Run-UninstallMetadataRace $currentApp
    $uninstallTrace = Join-Path $root 'uninstall-boundary.log'
    $uninstallExitCode = $script:uninstallRaceExitCode
    if ($uninstallExitCode -ne 0) {
        Write-Host ('Uninstall boundary: exit_code='+$uninstallExitCode)
        if (Test-Path -LiteralPath $uninstallTrace) { Get-Content -LiteralPath $uninstallTrace | Where-Object { $_ -match 'Uninstall boundary:' } | ForEach-Object { Write-Host ($_.Substring($_.IndexOf('Uninstall boundary:'))) } }
        $remainingService = Get-Service ImmichClone -ErrorAction SilentlyContinue
        Write-Host ('Uninstall boundary: service_present='+[bool]$remainingService)
        if ($remainingService) { Write-Host ('Uninstall boundary: service_state='+$remainingService.Status) }
        foreach ($leaf in @('Manage-Service.ps1','immichclone.exe','ai','media','rollback/catalog.db')) { Write-Host ('Uninstall boundary: selected_remaining='+[bool](Test-Path -LiteralPath (Join-Path $currentApp $leaf))) }
    }
 
 
 
 
    Assert ($uninstallExitCode -eq 0) 'disposable preparation installer uninstalls cleanly'
    foreach ($path in @($unknownRoot,$unknownRollback)) { Assert ([IO.File]::ReadAllText($path) -eq 'unrelated retained bytes') 'native uninstall preserves unrelated root and rollback contents' }
    Assert (!(Get-Service ImmichClone -ErrorAction SilentlyContinue) -and !(Test-Path 'HKLM:\SOFTWARE\ImmichClone')) 'completed native uninstall removes owned service and product registration'
    foreach ($leaf in $payloadNames) { Assert (!(Test-Path -LiteralPath (Join-Path $currentApp $leaf))) 'uninstall removes fixed native-published payload and provider fixture files' }
    Assert ((Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash) 'probe uninstall retains managed original bytes'
    # Use actual default Program Files inheritance too, rather than inferring
    # compatibility from a hand-protected temporary root.
    $programApp = Join-Path $env:ProgramFiles ('ImmichClone-trust-'+[guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $programApp | Out-Null
    Assert-TrustedCodeTree $programApp
    $currentApp = $programApp
    $freshState = Join-Path $env:ProgramData ('ImmichClone-state-'+[guid]::NewGuid().ToString('N'))
    Assert (!(Test-Path -LiteralPath $freshState)) 'fresh Setup fixture starts without state or application metadata'
    $commandLog = Join-Path $env:ProgramData 'ImmichCloneDiagnostics/commands.jsonl'
    $commandLogBefore = 'absent'
    if (Test-Path -LiteralPath $commandLog) { $commandLogBefore = (Get-FileHash -LiteralPath $commandLog).Hash }
    $freshSetupResult = Run-Setup $currentApp $freshState
    if ($freshSetupResult -ne 0) {
        # Fixed-schema diagnostic fields only; never output paths/exception text.
        $failureLog = Join-Path $env:ProgramData 'ImmichCloneDiagnostics/installer.jsonl'
        if (Test-Path -LiteralPath $failureLog) {
            foreach ($line in (Get-Content -LiteralPath $failureLog -Tail 8)) {
                $record = $line | ConvertFrom-Json
                Write-Host ('Setup diagnostic event='+$record.event+' stage='+$record.stage+' result='+$record.result_code)
            }
        }
    }
    Assert ($freshSetupResult -eq 0) 'actual fresh Setup works with default Program Files and ProgramData parent permissions'
    Wait-Health
    Assert (![UpgradeWriteProbe]::CanWrite($ordinaryUser,$ordinaryPassword,(Join-Path $currentApp 'Manage-Service.ps1'))) 'Program Files installed helper denies actual standard-user modification'
    Assert ((Get-CimInstance Win32_Service -Filter "Name='ImmichClone'").StartName -eq 'NT AUTHORITY\LocalService') 'fresh installer initializes through the existing LocalService account'
    foreach ($leaf in @('state-format.txt','catalog.db','access-code.txt','originals','staging','previews','faces')) {
        $owner = (Get-Acl -LiteralPath (Join-Path $freshState $leaf)).GetOwner([Security.Principal.SecurityIdentifier]).Value
        Assert ($owner -eq 'S-1-5-19') 'fresh state metadata and storage directories are created by LocalService rather than the installer identity'
    }
    $commandLogAfter = 'absent'
    if (Test-Path -LiteralPath $commandLog) { $commandLogAfter = (Get-FileHash -LiteralPath $commandLog).Hash }
    Assert ($commandLogAfter -eq $commandLogBefore) 'fresh Setup performs no elevated offline initialization command logging'
    $freshCode = ([IO.File]::ReadAllText((Join-Path $freshState 'access-code.txt'))).Trim()
    $freshSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $null = Invoke-RestMethod http://127.0.0.1:2283/api/login -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$freshCode}|ConvertTo-Json) -WebSession $freshSession
    $freshAsset = (Invoke-RestMethod 'http://127.0.0.1:2283/api/upload?name=fresh-fixture.bin' -Method Post -Headers @{'X-Immich-Request'='1'} -InFile $original -WebSession $freshSession).asset
    Assert ((Get-FileHash (Join-Path $freshState ('originals/'+$freshAsset.id))).Hash -eq $originalHash) 'LocalService-initialized fresh library accepts and preserves actual original bytes'
    $process = Start-Process (Join-Path $currentApp 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru
    foreach ($leaf in $payloadNames) { Assert (!(Test-Path -LiteralPath (Join-Path $currentApp $leaf))) 'Program Files uninstall removes native-published payload fixture files' }
    Assert ($process.ExitCode -eq 0 -and (Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash -and (Get-FileHash (Join-Path $freshState ('originals/'+$freshAsset.id))).Hash -eq $originalHash) 'Program Files uninstall preserves previous and fresh library original bytes'

} finally {
    # Never remove an unrelated service/registration if this fixture did not own it.
    if ($serviceOwned) {
        $service = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'" -ErrorAction SilentlyContinue
        if ($service -and $service.PathName.StartsWith(('"'+(Join-Path $currentApp 'immichclone.exe')+'"'),[StringComparison]::OrdinalIgnoreCase)) { Stop-Service ImmichClone -Force -ErrorAction SilentlyContinue; & sc.exe delete ImmichClone | Out-Null }
    }
    if ($currentApp -and (Test-Path (Join-Path $currentApp 'unins000.exe'))) { $null = Start-Process (Join-Path $currentApp 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru }
    if ($junction -and (Test-Path -LiteralPath $junction)) { [IO.Directory]::Delete($junction) }
    foreach ($target in @($programApp,$freshState)) {
        if (!$target) { continue }
        $absolute = [IO.Path]::GetFullPath($target)
        $parent = Split-Path $absolute -Parent
        $leaf = Split-Path $absolute -Leaf
        if (!(($parent -eq $env:ProgramFiles -and $leaf -match '^ImmichClone-trust-[0-9a-f]{32}$') -or ($parent -eq $env:ProgramData -and $leaf -match '^ImmichClone-state-[0-9a-f]{32}$'))) { throw 'Default-layout cleanup escaped disposable fixture' }
        if (Test-Path -LiteralPath $absolute) { Remove-Item -LiteralPath $absolute -Recurse -Force }
    }
    if ($ordinaryCreated) { Remove-LocalUser -Name $ordinaryUser -ErrorAction SilentlyContinue }
    if (Test-Path -LiteralPath $root) { Remove-Item -LiteralPath $root -Recurse -Force }
}
