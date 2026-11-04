param(
    [Parameter(Mandatory=$true)][ValidateSet('Prepare','Install','Uninstall')][string]$Action,
    [Parameter(Mandatory=$true)][string]$AppDir,
    [Parameter(Mandatory=$true)][string]$DataDir,
    [string]$BoundaryHelper = ''
)
$ErrorActionPreference = 'Stop'
# Use the OS-provided Windows PowerShell modules even when setup is launched
# from PowerShell 7 with a different inherited module search path.
$env:PSModulePath = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\Modules'
Import-Module Microsoft.PowerShell.Management -Force
Import-Module Microsoft.PowerShell.Security -Force
. (Join-Path $PSScriptRoot 'Storage-Security.ps1')
if (!$BoundaryHelper) { $BoundaryHelper = Join-Path $PSScriptRoot 'installer-files.exe' }
$name = 'ImmichClone'
$binary = Join-Path $AppDir 'immichclone.exe'
$rollback = Join-Path $AppDir 'rollback'
$stage = 'inspect service'
$pathsValidated = $false
$diagnosticsReady = $false
$logRoot = Join-Path $env:ProgramData 'ImmichCloneDiagnostics'
function Diagnostic([string]$event,[int]$result=0) {
    if (!$diagnosticsReady) { return }
    try {
        & $BoundaryHelper '-action' 'diagnostic' '-log-dir' $logRoot '-event' $event '-operation' $Action '-stage' $stage '-result' $result | Out-Null
        if ($LASTEXITCODE -ne 0) { throw 'Diagnostic operation refused' }
    } catch { Write-Warning 'A diagnostic event could not be saved' }
}

function Checked([string]$exe,[string[]]$arguments) { & $exe @arguments | Out-Null; if ($LASTEXITCODE -ne 0) { Diagnostic 'native_operation_failed' $LASTEXITCODE; throw 'A required installation operation failed' } }
function Own-Service {
    $service = Get-CimInstance Win32_Service -Filter "Name='$name'"
    if ($service -and !$service.PathName.StartsWith(('"'+$binary+'"'),[StringComparison]::OrdinalIgnoreCase)) { throw 'Service name belongs to a different installation' }
    return $service
}
function Health {
    for ($attempt=0;$attempt -lt 20;$attempt++) {
        try { if ((Invoke-RestMethod http://127.0.0.1:2283/health).status -eq 'ok') { return } } catch {}
        Start-Sleep -Milliseconds 500
    }
    throw 'Installed service did not become healthy'
}
try {
    # During Setup this script/helper come from the new bundled payload. Refuse
    # writable retained code before any log/ACL write, service stop or snapshot.
    $stage = 'audit code permissions'
    Assert-TrustedCodeTree $PSScriptRoot
    Assert-TrustedCodeTree $AppDir
    Assert-TrustedCodeTree (Split-Path -Parent $BoundaryHelper)
    Assert-NoReparsePath $AppDir
    Assert-NoReparsePath $DataDir
    Assert-NoReparsePath $logRoot
    $stage = 'audit storage permissions'
    if ([IO.Path]::GetFullPath($AppDir).TrimEnd('\') -eq [IO.Path]::GetFullPath($DataDir).TrimEnd('\')) { throw 'State and installation directories must differ' }
    foreach ($component in ($DataDir -split '[\\/]')) { if ($component -ieq 'Dropbox') { throw 'Application state cannot be inside a sync folder' } }
    if ((Test-Path -LiteralPath $DataDir) -and !(Test-Path -LiteralPath (Join-Path $DataDir 'state-format.txt')) -and @(Get-ChildItem -LiteralPath $DataDir -Force -ErrorAction Stop).Count -gt 0) { throw 'Refusing an unrelated nonempty data directory' }
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    # Audit every existing application-owned descendant before log/ACL writes,
    # service stop, rollback snapshot or failure-path mutation. Root inherited
    # grants are normalized only after all three read-only audits succeed.
    Assert-PrivateDirectoryTree -Path $DataDir -ReaderSid $identity -ServiceAccess
    Assert-PrivateDirectoryTree -Path $logRoot -ReaderSid $identity -ServiceAccess
    Assert-PrivateDirectoryTree -Path $rollback -OwnerSid $identity
    $logIdentity = $identity
    if (Test-Path -LiteralPath $DataDir) { Set-PrivateDirectoryAcl -Path $DataDir -ReaderSid $identity -ServiceAccess }
    if (Test-Path -LiteralPath $rollback) { Set-PrivateDirectoryAcl -Path $rollback }
    Set-PrivateDirectoryAcl -Path $logRoot -ReaderSid $logIdentity -ServiceAccess
    # Verify actual inherited ACLs too; never assume the root change propagated.
    # A failure here can leave safe root ACL normalization, but no service stop,
    # diagnostics, snapshot or data-content mutation has happened.
    Assert-PrivateDirectoryTree -Path $DataDir -ReaderSid $identity -ServiceAccess -VerifyInherited
    Assert-PrivateDirectoryTree -Path $logRoot -ReaderSid $identity -ServiceAccess -VerifyInherited
    Assert-PrivateDirectoryTree -Path $rollback -OwnerSid $identity -VerifyInherited
    $pathsValidated = $true
    $diagnosticsReady = $true
    Diagnostic 'installer_begin'
    $owned = Own-Service
    if ($Action -eq 'Prepare') {
        if ($owned) {
            $stage = 'snapshot previous installation'
            Diagnostic 'upgrade_prepare'
            Stop-Service $name
            Set-PrivateDirectoryAcl -Path $rollback
            Checked $BoundaryHelper @('-action','snapshot','-app-dir',$AppDir,'-data-dir',$DataDir)
        }
        Diagnostic 'installer_complete'
    exit 0
    }
    if ($Action -eq 'Uninstall') {
        if ($owned) { Stop-Service $name -Force; Checked sc.exe @('delete',$name) }
        # Application state and originals deliberately remain untouched.
        Diagnostic 'installer_complete'
    exit 0
    }
    $stage = 'validate state location'
    if ([IO.Path]::GetFullPath($AppDir).TrimEnd('\') -eq [IO.Path]::GetFullPath($DataDir).TrimEnd('\')) { throw 'State and installation directories must differ' }
    foreach ($component in ($DataDir -split '[\\/]')) { if ($component -ieq 'Dropbox') { throw 'Application state cannot be inside a sync folder' } }
    if ((Test-Path $DataDir) -and !(Test-Path "$DataDir/state-format.txt") -and @(Get-ChildItem -LiteralPath $DataDir -Force).Count -gt 0) { throw 'Refusing an unrelated nonempty data directory' }
    $stage = 'configure state ACL'
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    Checked icacls @($AppDir,'/grant','*S-1-5-19:(OI)(CI)RX')
    Set-PrivateDirectoryAcl -Path $DataDir -ReaderSid $identity -ServiceAccess
    $stage = 'initialize state'
    Checked $binary @('-mode','init','-data',$DataDir)
    if (!$owned) {
        $stage = 'register LocalService'
        $created = Invoke-CimMethod -ClassName Win32_Service -MethodName Create -Arguments @{
            Name=$name; DisplayName='Immich Clone (prototype)'; PathName=('"'+$binary+'" -mode service -data "'+$DataDir+'"')
            ServiceType=[byte]16; ErrorControl=[byte]1; DesktopInteract=$false
            StartMode='Automatic'; StartName='NT AUTHORITY\LocalService'; StartPassword=''
        }
        if ($created.ReturnValue -ne 0) { Diagnostic 'service_registration_failed' $created.ReturnValue; throw 'Native service registration failed' }
    }
    $stage = 'configure automatic startup'
    Checked sc.exe @('config',$name,'start=','delayed-auto')
    Checked sc.exe @('failure',$name,'reset=','86400','actions=','restart/5000/restart/15000/none/0')
    if ($env:IMMICH_INSTALL_TEST_FAILURE -eq '1') { throw 'Injected upgrade failure' }
    $stage = 'start service'
    Start-Service $name
    $stage = 'verify service health'
    Health
    Diagnostic 'installer_complete'
    exit 0
} catch {
    if (!$pathsValidated) {
        Write-Error 'Installation storage validation failed; service and data content were not changed.' -ErrorAction Continue
        exit 1
    }
    Diagnostic 'installer_failed' $_.Exception.HResult
    $inner = $_.Exception
    while ($null -ne $inner) {
        if ($inner -is [ComponentModel.Win32Exception]) { Diagnostic 'win32_error' $inner.NativeErrorCode }
        $inner = $inner.InnerException
    }
    # No paths, credentials, machine inventory or exception text in diagnostics.
    try {
        & $BoundaryHelper '-action' 'failure-note' '-app-dir' $AppDir '-stage' $stage '-result' $_.Exception.HResult | Out-Null
        if ($LASTEXITCODE -ne 0) { Write-Warning 'Installation failure note could not be saved' }
    } catch { Write-Warning 'Installation failure note could not be saved' }
    # Schema 1 installer never mutates originals. Restore prior catalog and binary
    # after startup/configuration failure; retained orphan blobs are never deleted.
    if ($Action -eq 'Install' -and (Test-Path "$rollback/immichclone.exe")) {
        try {
            if (Get-Service $name -ErrorAction SilentlyContinue) { Stop-Service $name -Force -ErrorAction SilentlyContinue }
            Checked $BoundaryHelper @('-action','restore','-app-dir',$AppDir,'-data-dir',$DataDir)
            Start-Service $name
            Health
            Diagnostic 'rollback_complete'
            Write-Error 'Upgrade failed; previous service restored and healthy' -ErrorAction Continue
        } catch { Write-Error 'Installation failed; rollback requires recovery from retained backup' -ErrorAction Continue }
    } elseif ($Action -eq 'Prepare' -and $owned) {
        Start-Service $name -ErrorAction SilentlyContinue
    } elseif ($Action -eq 'Install' -and !$owned) {
        if (Get-Service $name -ErrorAction SilentlyContinue) { Stop-Service $name -ErrorAction SilentlyContinue; & sc.exe delete $name | Out-Null }
    }
    Write-Error 'Installation action failed. User data was not deleted.' -ErrorAction Continue
    exit 1
}
