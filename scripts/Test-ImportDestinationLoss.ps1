$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('immich-destination-loss-'+[guid]::NewGuid().ToString('N'))
$name='ImmichImportDestinationProbe'
function Checked([string]$exe,[string[]]$argv) { & $exe @argv; if($LASTEXITCODE -ne 0){throw 'Destination-loss acceptance command failed'} }
try {
    New-Item -ItemType Directory $root | Out-Null
    Checked go @('test','-mod=readonly','-count=1','-timeout=120s','./internal/library','-run','^TestImportOriginalsDirectoryUnavailableRecovery$')
    Checked go @('build','-o',"$root/app.exe",'./cmd/immichclone')
    New-Item -ItemType Directory -Path "$root/state" | Out-Null
    Checked icacls @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
    Checked icacls @("$root/state",'/grant','*S-1-5-19:(OI)(CI)M')
    New-Item -ItemType Directory -Force -Path (Join-Path $env:ProgramData 'ImmichCloneDiagnostics') | Out-Null
    Checked icacls @((Join-Path $env:ProgramData 'ImmichCloneDiagnostics'),'/grant','*S-1-5-19:(OI)(CI)M')
    $credential=[PSCredential]::new('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
    New-Service -Name $name -BinaryPathName ('"'+"$root/app.exe"+'" -mode service -data "'+"$root/state"+'"') -Credential $credential | Out-Null
    Start-Service $name
    $env:IMMICH_DESTINATION_ROOT=$root
    Checked node @('tests/import-destination-loss.cjs','prepare')
    Move-Item "$root/state/originals" "$root/state/originals.offline"
    try { Checked node @('tests/import-destination-loss.cjs','outage') }
    finally { Move-Item "$root/state/originals.offline" "$root/state/originals" }
    Restart-Service $name
    Checked node @('tests/import-destination-loss.cjs','resume')
    Stop-Service $name
    Checked "$root/app.exe" @('-mode','verify','-data',"$root/state")
} finally {
    if(Get-Service $name -ErrorAction SilentlyContinue){Stop-Service $name -ErrorAction SilentlyContinue;sc.exe delete $name}
    Remove-Item Env:IMMICH_DESTINATION_ROOT -ErrorAction SilentlyContinue
    Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
