$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('immich-source-loss-'+[guid]::NewGuid().ToString('N'))
$name='ImmichImportSourceProbe'
function Checked([string]$exe,[string[]]$argv) { & $exe @argv; if($LASTEXITCODE -ne 0){throw 'Source-loss acceptance command failed'} }
try {
    New-Item -ItemType Directory $root | Out-Null
    Checked go @('build','-o',"$root/app.exe",'./cmd/immichclone')
    New-Item -ItemType Directory -Path "$root/state" | Out-Null
    Checked icacls @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
    Checked icacls @("$root/state",'/grant','*S-1-5-19:(OI)(CI)M')
    $credential=[PSCredential]::new('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
    New-Service -Name $name -BinaryPathName ('"'+"$root/app.exe"+'" -mode service -data "'+"$root/state"+'"') -Credential $credential | Out-Null
    Checked icacls @((Join-Path $env:ProgramData 'ImmichCloneDiagnostics'),'/grant','*S-1-5-19:(OI)(CI)M')
    Start-Service $name
    $env:IMMICH_TEST_STATE="$root/state"
    $env:IMMICH_IMPORT_SOURCE_FIXTURES="$root/source"
    Checked node @('tests/import-source-loss.cjs')
    Stop-Service $name
    Checked "$root/app.exe" @('-mode','verify','-data',"$root/state")
} finally {
    if(Get-Service $name -ErrorAction SilentlyContinue){Stop-Service $name -ErrorAction SilentlyContinue;sc.exe delete $name}
    Remove-Item Env:IMMICH_TEST_STATE,Env:IMMICH_IMPORT_SOURCE_FIXTURES -ErrorAction SilentlyContinue
    Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
