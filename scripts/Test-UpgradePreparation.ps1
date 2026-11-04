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
function Run-Setup([string]$Application,[string]$State,[switch]$ObserveStaging) {
    $process = Start-Process $script:setup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$Application+'"'),('/STATE="'+$State+'"')) -PassThru
    $observed = $false
    while (!$process.HasExited) {
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
                foreach ($file in @($staged,(Join-Path (Split-Path $staged -Parent) 'Storage-Security.ps1'))) {
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
    return $process.ExitCode
}
function Write-RetainedHelper([string]$Application) {
    @'
param([string]$Action,[string]$AppDir,[string]$DataDir)
[IO.File]::WriteAllText((Join-Path $PSScriptRoot 'retained-preparer-ran.txt'),'unexpected retained preparer execution')
exit 0
'@ | Set-Content -LiteralPath (Join-Path $Application 'Manage-Service.ps1')
}
$root = Join-Path $env:RUNNER_TEMP ('upgrade preparation '+[guid]::NewGuid().ToString('N')+' é')
$repo = Split-Path $PSScriptRoot -Parent
$currentApp = ''
$programApp = ''
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
        $process = Start-Process $compilerSetup -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/DIR="'+$compilerDir+'"')) -Wait -PassThru
        Assert ($process.ExitCode -eq 0) 'pinned Inno compiler provisioned for disposable probe'
        $Compiler = Join-Path $compilerDir 'ISCC.exe'
    }
    # Compile the actual production .iss, not a duplicate of its Pascal logic.
    # Minimal fixture providers are never delivered or uploaded as release artifacts.
    $source = Join-Path $root 'source'
    foreach ($directory in @('packaging','docs','dist','dist/media','dist/ai')) { New-Item -ItemType Directory -Force (Join-Path $source $directory) | Out-Null }
    foreach ($file in @('immichclone.iss','Manage-Service.ps1','Storage-Security.ps1','Export-Diagnostics.ps1')) { Copy-Item -LiteralPath (Join-Path $repo ('packaging/'+$file)) -Destination (Join-Path $source ('packaging/'+$file)) }
    Copy-Item -LiteralPath (Join-Path $repo 'docs/prototype-notes.txt') -Destination (Join-Path $source 'docs/prototype-notes.txt')
    Copy-Item -LiteralPath $Binary -Destination (Join-Path $source 'dist/immichclone.exe')
    foreach ($file in @('dist/THIRD-PARTY-LICENSES.txt','dist/media/preparation-probe.txt','dist/ai/preparation-probe.txt')) { [IO.File]::WriteAllText((Join-Path $source $file),'Disposable upgrade-preparation fixture only; not a distributable package.') }
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

    # Create an actual service-owned existing library without using the retained
    # helper. It deliberately represents old files, not an older runtime version.
    $currentApp = Join-Path $root 'existing application'
    $state = Join-Path $root 'existing state'
    New-Item -ItemType Directory -Path $currentApp | Out-Null
    Copy-Item -LiteralPath $Binary -Destination (Join-Path $currentApp 'immichclone.exe')
    Write-RetainedHelper $currentApp
    Native (Join-Path $currentApp 'immichclone.exe') @('-mode','init','-data',$state)
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
    Assert ((Get-ItemProperty 'HKLM:\SOFTWARE\ImmichClone').State -eq $state) 'Setup retains real application data location'
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
    $process = Start-Process (Join-Path $currentApp 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru
    Assert ($process.ExitCode -eq 0) 'disposable preparation installer uninstalls cleanly'
    Assert ((Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash) 'probe uninstall retains managed original bytes'
    # Use actual default Program Files inheritance too, rather than inferring
    # compatibility from a hand-protected temporary root.
    $programApp = Join-Path $env:ProgramFiles ('ImmichClone-trust-'+[guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $programApp | Out-Null
    Assert-TrustedCodeTree $programApp
    $currentApp = $programApp
    Assert ((Run-Setup $currentApp $state) -eq 0) 'actual Setup works with default Program Files code inheritance'
    Wait-Health
    Assert (![UpgradeWriteProbe]::CanWrite($ordinaryUser,$ordinaryPassword,(Join-Path $currentApp 'Manage-Service.ps1'))) 'Program Files installed helper denies actual standard-user modification'
    $process = Start-Process (Join-Path $currentApp 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru
    Assert ($process.ExitCode -eq 0 -and (Get-FileHash (Join-Path $state ('originals/'+$asset.id))).Hash -eq $originalHash) 'Program Files uninstall preserves managed original bytes'

} finally {
    # Never remove an unrelated service/registration if this fixture did not own it.
    if ($serviceOwned) {
        $service = Get-CimInstance Win32_Service -Filter "Name='ImmichClone'" -ErrorAction SilentlyContinue
        if ($service -and $service.PathName.StartsWith(('"'+(Join-Path $currentApp 'immichclone.exe')+'"'),[StringComparison]::OrdinalIgnoreCase)) { Stop-Service ImmichClone -Force -ErrorAction SilentlyContinue; & sc.exe delete ImmichClone | Out-Null }
    }
    if ($currentApp -and (Test-Path (Join-Path $currentApp 'unins000.exe'))) { $null = Start-Process (Join-Path $currentApp 'unins000.exe') -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -Wait -PassThru }
    if ($junction -and (Test-Path -LiteralPath $junction)) { [IO.Directory]::Delete($junction) }
    if ($programApp -and (Test-Path -LiteralPath $programApp)) { Remove-Item -LiteralPath $programApp -Recurse -Force }
    if ($ordinaryCreated) { Remove-LocalUser -Name $ordinaryUser -ErrorAction SilentlyContinue }
    if (Test-Path -LiteralPath $root) { Remove-Item -LiteralPath $root -Recurse -Force }
}
