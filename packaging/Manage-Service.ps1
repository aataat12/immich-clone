param(
    [Parameter(Mandatory=$true)][ValidateSet('Prepare','Install','Uninstall')][string]$Action,
    [Parameter(Mandatory=$true)][string]$AppDir,
    [Parameter(Mandatory=$true)][string]$DataDir
)
$ErrorActionPreference = 'Stop'
$name = 'ImmichClone'
$binary = Join-Path $AppDir 'immichclone.exe'
$rollback = Join-Path $AppDir 'rollback'
$stage = 'inspect service'
function Checked([string]$exe,[string[]]$arguments) { & $exe @arguments | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'A required installation operation failed' } }
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
    $owned = Own-Service
    if ($Action -eq 'Prepare') {
        if ($owned) {
            Stop-Service $name
            New-Item -ItemType Directory -Force $rollback | Out-Null
            Checked icacls @($rollback,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F')
            Copy-Item $binary "$rollback/immichclone.exe" -Force
            Copy-Item "$DataDir/catalog.db" "$rollback/catalog.db" -Force
            Copy-Item "$DataDir/access-code.txt" "$rollback/access-code.txt" -Force
        }
        exit 0
    }
    if ($Action -eq 'Uninstall') {
        if ($owned) { Stop-Service $name -Force; Checked sc.exe @('delete',$name) }
        # Application state and originals deliberately remain untouched.
        exit 0
    }
    $stage = 'validate state location'
    if ([IO.Path]::GetFullPath($AppDir).TrimEnd('\') -eq [IO.Path]::GetFullPath($DataDir).TrimEnd('\')) { throw 'State and installation directories must differ' }
    foreach ($component in ($DataDir -split '[\\/]')) { if ($component -ieq 'Dropbox') { throw 'Application state cannot be inside a sync folder' } }
    if ((Test-Path $DataDir) -and !(Test-Path "$DataDir/state-format.txt") -and @(Get-ChildItem -LiteralPath $DataDir -Force).Count -gt 0) { throw 'Refusing an unrelated nonempty data directory' }
    New-Item -ItemType Directory -Force $DataDir | Out-Null
    $stage = 'configure state ACL'
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    Checked icacls @($DataDir,'/inheritance:r','/grant:r','*S-1-5-18:(OI)(CI)F','*S-1-5-32-544:(OI)(CI)F','*S-1-5-19:(OI)(CI)M',('*'+$identity+':(OI)(CI)RX'))
    $stage = 'initialize state'
    Checked $binary @('-mode','init','-data',$DataDir)
    if (!$owned) {
        $stage = 'register LocalService'
        $credential = [PSCredential]::new('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
        New-Service -Name $name -DisplayName 'Immich Clone (prototype)' -BinaryPathName ('"'+$binary+'" -mode service -data "'+$DataDir+'"') -StartupType Automatic -Credential $credential | Out-Null
    }
    $stage = 'configure automatic startup'
    Checked sc.exe @('config',$name,'start=','delayed-auto')
    Checked sc.exe @('failure',$name,'reset=','86400','actions=','restart/5000/restart/15000/none/0')
    if ($env:IMMICH_INSTALL_TEST_FAILURE -eq '1') { throw 'Injected upgrade failure' }
    $stage = 'start service'
    Start-Service $name
    Health
    exit 0
} catch {
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
