$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('immich-mixed-'+[guid]::NewGuid().ToString('N'))
$name='ImmichMixedImportProbe'
function Checked([string]$exe,[string[]]$argv) { & $exe @argv; if($LASTEXITCODE -ne 0){throw 'Mixed import acceptance command failed'} }
try {
    New-Item -ItemType Directory $root | Out-Null
    ./scripts/Prepare-MediaProvider.ps1 -Destination "$root/media/ffmpeg-8.1.3-7eb77d88e2e9"
    Checked go @('build','-o',"$root/app.exe",'./cmd/immichclone')
    New-Item -ItemType Directory -Path "$root/state" | Out-Null
    Checked icacls @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
    Checked icacls @("$root/state",'/grant','*S-1-5-19:(OI)(CI)M')
    New-Item -ItemType Directory -Force -Path (Join-Path $env:ProgramData 'ImmichCloneDiagnostics') | Out-Null
    Checked icacls @((Join-Path $env:ProgramData 'ImmichCloneDiagnostics'),'/grant','*S-1-5-19:(OI)(CI)M')
    $credential=[PSCredential]::new('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
    New-Service -Name $name -BinaryPathName ('"'+"$root/app.exe"+'" -mode service -data "'+"$root/state"+'"') -Credential $credential | Out-Null
    Start-Service $name
    $env:IMMICH_MIXED_ROOT=$root
    $node=(Get-Command node).Source
    $probe=Start-Process $node -ArgumentList @('tests/import-mixed.cjs','start') -PassThru -RedirectStandardOutput "$root/probe.out" -RedirectStandardError "$root/probe.err"
    try {
        $deadline=[DateTime]::UtcNow.AddSeconds(120)
        while(!(Test-Path "$root/crash-ready") -and !$probe.HasExited -and [DateTime]::UtcNow -lt $deadline){Start-Sleep -Milliseconds 100}
        if(!(Test-Path "$root/crash-ready")){Get-Content "$root/probe.err";throw 'Partial transfer crash barrier not reached'}
        $servicePID=(Get-CimInstance Win32_Service -Filter "Name='$name'").ProcessId
        Stop-Process -Id $servicePID -Force
        if(!$probe.WaitForExit(30000)){throw 'Browser did not finish interrupted-transfer recovery'}
        Get-Content "$root/probe.out"
        if($probe.ExitCode -ne 0){Get-Content "$root/probe.err";throw 'Interrupted mixed import browser phase failed'}
    } finally {if(!$probe.HasExited){Stop-Process -Id $probe.Id -Force}}
    $deadline=[DateTime]::UtcNow.AddSeconds(15)
    while((Get-Service $name).Status -ne 'Stopped' -and [DateTime]::UtcNow -lt $deadline){Start-Sleep -Milliseconds 100}
    Start-Service $name
    Checked node @('tests/import-mixed.cjs','finish')
    Stop-Service $name
    Checked "$root/app.exe" @('-mode','verify','-data',"$root/state")
    Checked "$root/app.exe" @('-mode','restore','-data',"$root/restored",'-archive',"$root/mixed.zip")
    Move-Item "$root/state" "$root/state.offline"
    $env:IMMICH_MIXED_RESTORE="$root/restored"
    $env:IMMICH_MIXED_MEDIA="$root/media/ffmpeg-8.1.3-7eb77d88e2e9"
    Checked go @('test','-count=1','-timeout=180s','./internal/library','-run','^TestMixedImportArchiveRestored$')
} finally {
    if(Get-Service $name -ErrorAction SilentlyContinue){Stop-Service $name -ErrorAction SilentlyContinue;sc.exe delete $name}
    foreach($key in @('IMMICH_MIXED_ROOT','IMMICH_MIXED_RESTORE','IMMICH_MIXED_MEDIA')){Remove-Item "Env:$key" -ErrorAction SilentlyContinue}
    Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
}
