param([Parameter(Mandatory=$true)][string]$Root)
$ErrorActionPreference='Stop'
# Same already-executed, hash-pinned test encoder. Not distributed with the app.
$tool=Join-Path $env:RUNNER_TEMP '6a-test-encoder'
New-Item -ItemType Directory $tool -Force | Out-Null
try {
 $archive=Join-Path $tool 'tool.zip'
 Invoke-WebRequest 'https://github.com/BtbN/FFmpeg-Builds/releases/download/autobuild-2026-10-31-12-59/ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip' -OutFile $archive
 if ((Get-FileHash $archive).Hash.ToLowerInvariant() -ne '5cef7bc48e513f3cee7c881e8b2999d835541edb1846c3e89443a67f8f8556ab') {throw 'Fixture encoder checksum mismatch'}
 Expand-Archive $archive "$tool/unpack"
 $ff=(Get-ChildItem "$tool/unpack" -Filter ffmpeg.exe -Recurse | Select-Object -First 1).FullName
 function Encode([string[]]$ArgsToUse) { & $ff -hide_banner -loglevel error -nostdin -y @ArgsToUse; if($LASTEXITCODE -ne 0){throw '6A fixture encoding failed'} }
 Encode @('-framerate','8','-i',"$Root/frames/%03d.png",'-c:v','libopenh264','-b:v','2000k','-pix_fmt','yuv420p',"$Root/video.mp4")
 Encode @('-i',"$Root/video.mp4",'-c','copy',"$Root/recovery.mov")
 Encode @('-display_rotation:v:0','90','-i',"$Root/video.mp4",'-c','copy',"$Root/rotated-face.mov")
 Encode @('-i',"$Root/video.mp4",'-t','0.125','-c','copy',"$Root/short.mp4")
 @("file 'frames/000.png'","duration 0.25","file 'frames/031.png'","duration 0.75","file 'frames/000.png'","duration 0.125","file 'frames/031.png'","duration 0.875","file 'frames/031.png'") | Set-Content "$Root/vfr.txt"
 Encode @('-f','concat','-safe','0','-i',"$Root/vfr.txt",'-fps_mode','vfr','-c:v','libopenh264','-b:v','2000k','-pix_fmt','yuv420p',"$Root/vfr.mp4")
 Encode @('-f','lavfi' ,'-i','sine=frequency=440:sample_rate=44100','-t','1','-c:a','aac',"$Root/audio.mp4")
 [IO.File]::WriteAllBytes("$Root/hevc.mp4",[Convert]::FromBase64String((Get-Content 'experiments/media-worker/fixtures/hevc.mp4.b64' -Raw)))
 # Preserve fixture identities for before/after checks; never output private content.
 $hashes=@{};Get-ChildItem $Root -File | Where-Object Extension -in @('.png','.heic','.mp4','.mov') | ForEach-Object {$hashes[$_.Name]=(Get-FileHash $_.FullName).Hash.ToLowerInvariant()}
 $hashes|ConvertTo-Json|Set-Content "$Root/fixture-hashes.json"
 Write-Output 'PASS: controlled H264 face/object sequence, rotated MOV, short clip, no-video and existing HEVC fixtures prepared'
} finally {Remove-Item $tool -Recurse -Force -ErrorAction SilentlyContinue}
