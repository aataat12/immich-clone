param(
    [Parameter(Mandatory=$true)][ValidateSet('Prepare','Install','Uninstall')][string]$Action,
    [Parameter(Mandatory=$true)][string]$AppDir,
    [Parameter(Mandatory=$true)][string]$DataDir,
    [string]$BoundaryHelper = '',
    [string]$PayloadSource = '',
    [string]$PayloadBackup = ''
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
$stoppedForAcl = $false
$owned = $null
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
    if ($Action -eq 'Uninstall') {
        # Bootstrap comes from the uninstaller's hash-verified private staging.
        # No ACL/log/state writes. Unsafe stationary trees refuse before stop.
        Checked $BoundaryHelper @('-action','audit-cleanup','-app-dir',$AppDir,'-data-dir',$DataDir)
        $owned = Own-Service
        if ($owned) { Stop-Service $name -Force }
        Checked $BoundaryHelper @('-action','cleanup','-app-dir',$AppDir,'-data-dir',$DataDir)
        if ($owned) { Checked sc.exe @('delete',$name) }
        exit 0
    }
    # During Setup this script/helper come from the new bundled payload. Refuse
    # writable retained code before any log/ACL write, service stop or snapshot.
    $stage = 'audit code permissions'
    Assert-TrustedCodeTree $PSScriptRoot
    Assert-TrustedCodeTree $AppDir
    Assert-TrustedCodeTree (Split-Path -Parent $BoundaryHelper)
    if ($Action -eq 'Prepare' -and (Test-Path -LiteralPath $AppDir)) { Checked $BoundaryHelper @('-action','audit-code','-app-dir',$AppDir) }
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
    # Unsafe stationary trees were refused above before any service stop. Native
    # ACL handles require exclusive write/delete sharing against active catalog
    # writers, so stop only our verified service before held-tree normalization.
    $owned = Own-Service
    if ($owned -and $owned.State -eq 'Running') {
        Stop-Service $name
        $stoppedForAcl = $true
    }
    $logIdentity = $identity
    if (Test-Path -LiteralPath $DataDir) { Set-PrivateDirectoryAcl -CreationHelper $BoundaryHelper -Path $DataDir -ReaderSid $identity -ServiceAccess }
    if (Test-Path -LiteralPath $rollback) { Set-PrivateDirectoryAcl -CreationHelper $BoundaryHelper -Path $rollback }
    Set-PrivateDirectoryAcl -CreationHelper $BoundaryHelper -Path $logRoot -ReaderSid $logIdentity -ServiceAccess
    # Verify actual inherited ACLs too; never assume the root change propagated.
    # Failure may leave partial ACL normalization and a stopped service. No
    # diagnostic, snapshot or data-content mutation has happened in this phase.
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
            Set-PrivateDirectoryAcl -CreationHelper $BoundaryHelper -Path $rollback
            Checked $BoundaryHelper @('-action','snapshot','-app-dir',$AppDir,'-data-dir',$DataDir)
        }
        Diagnostic 'installer_complete'
    exit 0
    }
    $stage = 'validate state location'
    if ([IO.Path]::GetFullPath($AppDir).TrimEnd('\') -eq [IO.Path]::GetFullPath($DataDir).TrimEnd('\')) { throw 'State and installation directories must differ' }
    foreach ($component in ($DataDir -split '[\\/]')) { if ($component -ieq 'Dropbox') { throw 'Application state cannot be inside a sync folder' } }
    if ((Test-Path $DataDir) -and !(Test-Path "$DataDir/state-format.txt") -and @(Get-ChildItem -LiteralPath $DataDir -Force).Count -gt 0) { throw 'Refusing an unrelated nonempty data directory' }
        if ($PayloadSource -and $PayloadBackup) {
            $stage = 'publish application payload'
            Checked $BoundaryHelper @('-action','ensure-private-root','-root-dir',$PayloadBackup)
            Checked $BoundaryHelper @('-action','publish-payload','-app-dir',$AppDir,'-source-dir',$PayloadSource,'-backup-dir',$PayloadBackup)
        }
    $stage = 'configure state ACL'
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    Checked $BoundaryHelper @('-action','grant-code-read','-app-dir',$AppDir)
    Set-PrivateDirectoryAcl -CreationHelper $BoundaryHelper -Path $DataDir -ReaderSid $identity -ServiceAccess
    # Initialization happens in the existing LocalService startup before health,
    # never by executing the application with this elevated installer token.
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
    $stage = 'start service'
    Start-Service $name
    $stage = 'verify service health'
    Health
    if ($env:IMMICH_INSTALL_TEST_FAILURE -eq '1') { throw 'Injected post-initialization upgrade failure' }
    Diagnostic 'installer_complete'
    exit 0
} catch {
    if ($Action -eq 'Uninstall') {
        Write-Error 'Uninstall refused or incomplete. Application data was preserved; review the service and registration before retrying.' -ErrorAction Continue
        exit 1
    }
    if (!$pathsValidated) {
        if ($stoppedForAcl) {
            try {
                Assert-PrivateDirectoryTree -Path $DataDir -ReaderSid $identity -ServiceAccess -VerifyInherited
                Assert-PrivateDirectoryTree -Path $logRoot -ReaderSid $identity -ServiceAccess -VerifyInherited
                Assert-PrivateDirectoryTree -Path $rollback -OwnerSid $identity -VerifyInherited
                Start-Service $name
                Health
                Write-Error 'Installation ACL normalization failed; prior service restarted after final permission verification. Data content was not changed.' -ErrorAction Continue
            } catch {
                Write-Error 'Installation ACL normalization failed; service remains stopped and administrator recovery is required. Data content was not changed.' -ErrorAction Continue
            }
        } else {
            Write-Error 'Installation storage validation failed; service and data content were not changed.' -ErrorAction Continue
        }
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
            if ($PayloadBackup) {
                Checked $BoundaryHelper @('-action','restore-payload','-app-dir',$AppDir,'-backup-dir',$PayloadBackup)
            }
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
        if ($PayloadBackup) {
            try { Checked $BoundaryHelper @('-action','restore-payload','-app-dir',$AppDir,'-backup-dir',$PayloadBackup) } catch { Write-Warning 'Fresh payload recovery requires administrator review' }
        }
    }
    Write-Error 'Installation action failed. User data was not deleted.' -ErrorAction Continue
    exit 1
}
