param(
    [Parameter(Mandatory=$true)][ValidateSet('Prepare','Install','Uninstall')][string]$Action,
    [Parameter(Mandatory=$true)][string]$AppDir,
    [Parameter(Mandatory=$true)][string]$DataDir
)
$ErrorActionPreference = 'Stop'
# Use the OS-provided Windows PowerShell modules even when setup is launched
# from PowerShell 7 with a different inherited module search path.
$env:PSModulePath = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\Modules'
Import-Module Microsoft.PowerShell.Management -Force
Import-Module Microsoft.PowerShell.Security -Force
. (Join-Path $PSScriptRoot 'Storage-Security.ps1')
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
        $file = Join-Path $logRoot 'installer.jsonl'
        if ((Test-Path $file) -and (Get-Item $file).Length -gt 4MB-1024) {
            Move-Item -LiteralPath $file -Destination ($file+'.previous') -Force
        }
        # These values are fixed internal labels and numbers, never exception text.
        $record = @{time_utc=[DateTime]::UtcNow.ToString('o');event=$event;action=$Action;stage=$stage;result_code=$result}
        [IO.File]::AppendAllText($file,($record | ConvertTo-Json -Compress)+[Environment]::NewLine,(New-Object Text.UTF8Encoding($false)))
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
    Assert-NoReparsePath $AppDir
    Assert-NoReparsePath $DataDir
    Assert-NoReparsePath $logRoot
    $pathsValidated = $true
    $logIdentity = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    Set-PrivateDirectoryAcl -Path $logRoot -ReaderSid $logIdentity -ServiceAccess
    $diagnosticsReady = $true
    Diagnostic 'installer_begin'
    $owned = Own-Service
    if ($Action -eq 'Prepare') {
        if ($owned) {
            $stage = 'snapshot previous installation'
            Diagnostic 'upgrade_prepare'
            Stop-Service $name
            Set-PrivateDirectoryAcl -Path $rollback
            Copy-Item $binary "$rollback/immichclone.exe" -Force
            Copy-Item "$DataDir/catalog.db" "$rollback/catalog.db" -Force
            Copy-Item "$DataDir/access-code.txt" "$rollback/access-code.txt" -Force
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
        Write-Error 'Installation paths could not be validated; no installation action performed.' -ErrorAction Continue
        exit 1
    }
    Diagnostic 'installer_failed' $_.Exception.HResult
    $inner = $_.Exception
    while ($null -ne $inner) {
        if ($inner -is [ComponentModel.Win32Exception]) { Diagnostic 'win32_error' $inner.NativeErrorCode }
        $inner = $inner.InnerException
    }
    # No paths, credentials, machine inventory or exception text in diagnostics.
    ($stage+'; '+$_.Exception.GetType().Name+'; HResult='+$_.Exception.HResult) | Set-Content (Join-Path $AppDir 'install-diagnostic.txt')
    # Schema 1 installer never mutates originals. Restore prior catalog and binary
    # after startup/configuration failure; retained orphan blobs are never deleted.
    if ($Action -eq 'Install' -and (Test-Path "$rollback/immichclone.exe")) {
        try {
            if (Get-Service $name -ErrorAction SilentlyContinue) { Stop-Service $name -Force -ErrorAction SilentlyContinue }
            Copy-Item "$rollback/immichclone.exe" $binary -Force
            Copy-Item "$rollback/catalog.db" "$DataDir/catalog.db" -Force
            Copy-Item "$rollback/access-code.txt" "$DataDir/access-code.txt" -Force
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
