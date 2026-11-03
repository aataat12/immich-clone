param([Parameter(Mandatory=$true)][string]$Fixtures)
$ErrorActionPreference='Stop'
$root=Join-Path $env:RUNNER_TEMP ('immich-6a-app-'+[guid]::NewGuid().ToString('N'))
$name='ImmichClone6AApp'
function Checked([string]$exe,[string[]]$argsToUse) { & $exe @argsToUse; if($LASTEXITCODE -ne 0){throw '6A application command failed'} }
try {
 New-Item -ItemType Directory $root | Out-Null
 ./scripts/Prepare-MediaProvider.ps1 -Destination "$root/media/ffmpeg-8.1.3-7eb77d88e2e9"
 ./scripts/Prepare-AIProvider.ps1 -Destination "$root/ai/opencv-4.12.0-nr01-zlib-1.3.2-15d1c2ca7ef3"
 Checked go @('build','-trimpath','-ldflags','-X main.version=6a-candidate-prototype','-o',"$root/app.exe",'./cmd/immichclone')
 Checked "$root/app.exe" @('-mode','init','-data',"$root/state")
 Checked icacls @($root,'/grant','*S-1-5-19:(OI)(CI)RX')
 Checked icacls @("$root/state",'/grant','*S-1-5-19:(OI)(CI)M')
 $cred=[PSCredential]::new('NT AUTHORITY\LocalService',(ConvertTo-SecureString 'unused' -AsPlainText -Force))
 New-Service -Name $name -BinaryPathName ('"'+"$root/app.exe"+'" -mode service -data "'+"$root/state"+'"') -Credential $cred | Out-Null
 Checked icacls @((Join-Path $env:ProgramData 'ImmichCloneDiagnostics'),'/grant','*S-1-5-19:(OI)(CI)M')
 Start-Service $name
 $env:IMMICH_TEST_STATE="$root/state";$env:IMMICH_6A_FIXTURES=$Fixtures
 Checked node @('tests/media-ai.cjs','start')
 Restart-Service $name
 Checked node @('tests/media-ai.cjs','finish')
 # Capture an application-consistent archive, then remove access to the old state.
 $session=[Microsoft.PowerShell.Commands.WebRequestSession]::new()
 $code=(Get-Content "$root/state/access-code.txt" -Raw).Trim()
 $null=Invoke-RestMethod 'http://127.0.0.1:2283/api/login' -Method Post -Headers @{'X-Immich-Request'='1'} -Body (@{code=$code}|ConvertTo-Json) -WebSession $session
 Invoke-WebRequest 'http://127.0.0.1:2283/api/backup' -WebSession $session -OutFile "$root/backup.zip"
 Stop-Service $name
 Checked "$root/app.exe" @('-mode','restore','-data',"$root/restored",'-archive',"$root/backup.zip")
 Move-Item "$root/state" "$root/state.offline"
 $env:IMMICH_6A_RESTORE="$root/restored";$env:IMMICH_6A_MEDIA="$root/media/ffmpeg-8.1.3-7eb77d88e2e9"
 Checked go @('test','-count=1','-timeout=180s','./internal/library','-run','^TestMediaAIApplicationRestoredCheckpoint$')
 Write-Output 'PASS: 6A LocalService app, persisted controls, actual inference and independent restored frame regeneration'
} finally {
 if(Get-Service $name -ErrorAction SilentlyContinue){Stop-Service $name -ErrorAction SilentlyContinue;sc.exe delete $name}
 Remove-Item $root -Recurse -Force -ErrorAction SilentlyContinue
 foreach($key in @('IMMICH_TEST_STATE','IMMICH_6A_FIXTURES','IMMICH_6A_RESTORE','IMMICH_6A_MEDIA')){Remove-Item "Env:$key" -ErrorAction SilentlyContinue}
}
