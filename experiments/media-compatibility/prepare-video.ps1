param([Parameter(Mandatory=$true)][string]$Root)
$ErrorActionPreference='Stop'
$tool=Join-Path $env:RUNNER_TEMP ('compat-encoder-'+[guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory $tool | Out-Null
try {
    $archive=Join-Path $tool 'tool.zip'
    Invoke-WebRequest 'https://github.com/BtbN/FFmpeg-Builds/releases/download/autobuild-2026-10-31-12-59/ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip' -OutFile $archive
    if ((Get-FileHash $archive).Hash.ToLowerInvariant() -ne '5cef7bc48e513f3cee7c881e8b2999d835541edb1846c3e89443a67f8f8556ab') { throw 'Fixture encoder checksum mismatch' }
    Expand-Archive $archive "$tool/unpack"
    $ff=(Get-ChildItem "$tool/unpack" -Filter ffmpeg.exe -Recurse | Select-Object -First 1).FullName
    $h264=Join-Path $Root 'h264.mp4'
    [IO.File]::WriteAllBytes($h264,[Convert]::FromBase64String((Get-Content 'experiments/media-worker/fixtures/h264.mp4.b64' -Raw)))
    if ((Get-FileHash $h264).Hash.ToLowerInvariant() -ne '8dca451d45ef05a2fe5d4392bd9f1afcffb99f5236185dede9bd052f1cde52d9') { throw 'Fixture source checksum mismatch' }
    function Encode([string[]]$Arguments) { & $ff -hide_banner -loglevel error -nostdin -y @Arguments; if ($LASTEXITCODE -ne 0) { throw 'Compatibility fixture encoding failed' } }
    Encode @('-f','lavfi','-i','sine=frequency=440:sample_rate=44100','-i',$h264,'-map','0:a:0','-map','1:v:0','-t','1','-c:a','aac','-c:v','copy',(Join-Path $Root 'audio-first.mp4'))
    Encode @('-i',$h264,'-map','0:v:0','-c','copy','-output_ts_offset','5',(Join-Path $Root 'nonzero-start.mov'))
    # Hashes are generated after pinned input/encoder execution, before production decoding.
    $hashes=@{};Get-ChildItem $Root -File | Where-Object Extension -in @('.mp4','.mov') | ForEach-Object {$hashes[$_.Name]=(Get-FileHash $_.FullName).Hash.ToLowerInvariant()}
    $hashes | ConvertTo-Json | Set-Content (Join-Path $Root 'video-hashes.json')
    Write-Output 'PASS: independent audio-first stream and nonzero-timestamp fixtures generated with pinned CI-only encoder'
} finally { Remove-Item $tool -Recurse -Force -ErrorAction SilentlyContinue }
