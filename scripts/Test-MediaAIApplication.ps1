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
 # Seed partial observations, then take a durable paused baseline after restart.
 Checked node @('tests/media-ai-recovery.cjs','seed')
 Restart-Service $name
 Checked node @('tests/media-ai-recovery.cjs','baseline')
 $ffmpeg="$root/media/ffmpeg-8.1.3-7eb77d88e2e9/ffmpeg.exe"
 Move-Item $ffmpeg ($ffmpeg+'.held')
 try { Checked node @('tests/media-ai-recovery.cjs','decoder-failure') }
 finally { Move-Item ($ffmpeg+'.held') $ffmpeg }
 Restart-Service $name
 Checked node @('tests/media-ai-recovery.cjs','decoder-retry')
 # Deny only disposable derived-crop writes, not originals or the catalog.
 Checked icacls @("$root/state/faces",'/deny','*S-1-5-19:(OI)(CI)W')
 try { Checked node @('tests/media-ai-recovery.cjs','storage-failure') }
 finally { Checked icacls @("$root/state/faces",'/remove:d','*S-1-5-19') }
 $model="$root/ai/opencv-4.12.0-nr01-zlib-1.3.2-15d1c2ca7ef3/sface.onnx"
 Move-Item $model ($model+'.held')
 try { Checked node @('tests/media-ai-recovery.cjs','model-failure') }
 finally { Move-Item ($model+'.held') $model }
 Restart-Service $name
 Checked node @('tests/media-ai-recovery.cjs','resume-crash')
 # Terminate the actual service while a native child is running, without graceful
 # shutdown. Job-object cleanup must reap the child; the saved cursor must resume.
 $servicePid=(Get-CimInstance Win32_Service -Filter "Name='$name'").ProcessId
 $observed=$false
 $deadline=[DateTime]::UtcNow.AddSeconds(45)
 while([DateTime]::UtcNow -lt $deadline) {
   $child=Get-CimInstance Win32_Process -Filter "Name='ai-worker.exe' AND ParentProcessId=$servicePid"
   if($child) { $observed=$true;break }
   Start-Sleep -Milliseconds 75
 }
 if(!$observed){throw 'Actual in-flight native child was not observed; crash test not established'}
 Stop-Process -Id $servicePid -Force
 $deadline=[DateTime]::UtcNow.AddSeconds(15)
 while((Get-Service $name).Status -ne 'Stopped' -and [DateTime]::UtcNow -lt $deadline){Start-Sleep -Milliseconds 100}
 if(Get-CimInstance Win32_Process -Filter "Name='ai-worker.exe' AND ParentProcessId=$servicePid"){throw 'Native child survived service crash'}
 Start-Service $name
 Checked node @('tests/media-ai-recovery.cjs','recovered')
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
