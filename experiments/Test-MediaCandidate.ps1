# Bounded candidate experiment only. No production dependency acceptance implied.
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$root = Join-Path $env:RUNNER_TEMP ('media-probe-' + [guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($root) | Out-Null
$url = 'https://github.com/BtbN/FFmpeg-Builds/releases/download/autobuild-2026-10-31-12-59/ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip'
$expected = '5cef7bc48e513f3cee7c881e8b2999d835541edb1846c3e89443a67f8f8556ab'
$checks = [ordered]@{}
function Assert([bool]$Condition, [string]$Name) {
    if (!$Condition) { throw "Failed: $Name" }
    $checks[$Name] = 'passed'
}
function FF([string[]]$Arguments) {
    & $script:ffmpeg -hide_banner -loglevel error -nostdin -n @Arguments
    if ($LASTEXITCODE -ne 0) { throw "FFmpeg failed with exit code $LASTEXITCODE" }
}
function Probe([string]$Path) {
    $json = & $script:ffprobe -v error -show_streams -show_format -of json $Path
    if ($LASTEXITCODE -ne 0) { throw 'ffprobe failed' }
    $json | ConvertFrom-Json
}
try {
    $archive = Join-Path $root 'candidate.zip'
    Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $archive
    Assert ((Get-FileHash $archive -Algorithm SHA256).Hash.ToLowerInvariant() -eq $expected) 'pinned-archive-sha256'
    Expand-Archive -Path $archive -DestinationPath "$root/tool"
    $script:ffmpeg = (Get-ChildItem "$root/tool" -Filter ffmpeg.exe -Recurse | Select-Object -First 1).FullName
    $script:ffprobe = (Get-ChildItem "$root/tool" -Filter ffprobe.exe -Recurse | Select-Object -First 1).FullName
    $version = & $ffmpeg -version
    if ($LASTEXITCODE -ne 0) { throw 'Cannot execute candidate binary' }
    $license = & $ffmpeg -L
    if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect candidate license' }
    Assert (($version -join "`n") -notmatch '--enable-gpl|--enable-nonfree') 'no-gpl-or-nonfree-config-flags'

    $png = Join-Path $root 'fixture.png'
    FF @('-f','lavfi','-i','testsrc=size=320x240:rate=1','-frames:v','1','-threads','1',$png)
    Assert ((Probe $png).streams[0].width -eq 320) 'png-generation-and-metadata-read'
    $pngHash = (Get-FileHash $png).Hash
    $jpeg = Join-Path $root 'fixture.jpg'
    FF @('-i',$png,'-frames:v','1','-q:v','2',$jpeg)
    $jpegHash = (Get-FileHash $jpeg).Hash
    $thumb = Join-Path $root 'thumbnail.png'
    FF @('-i',$jpeg,'-vf','scale=80:-1','-frames:v','1',$thumb)
    $dimensions = (Probe $thumb).streams[0]
    Assert ($dimensions.width -eq 80 -and $dimensions.height -eq 60) 'jpeg-decode-and-thumbnail'

    $mp4 = Join-Path $root 'fixture.mp4'
    # MPEG-4 Part 2/AAC fixture: does not establish H.264/HEVC support.
    FF @('-f','lavfi','-i','testsrc=size=320x240:rate=10','-f','lavfi','-i','sine=frequency=440:sample_rate=44100','-t','1','-c:v','mpeg4','-pix_fmt','yuv420p','-c:a','aac','-metadata','title=Synthetic probe title',$mp4)
    $mp4Hash = (Get-FileHash $mp4).Hash
    $info = Probe $mp4
    Assert ($info.format.tags.title -eq 'Synthetic probe title' -and @($info.streams).Count -eq 2) 'video-audio-and-title-metadata'
    $frame = Join-Path $root 'frame.jpg'
    FF @('-ss','0.2','-i',$mp4,'-frames:v','1','-vf','scale=160:-1',$frame)
    Assert ((Probe $frame).streams[0].width -eq 160) 'video-decode-frame-extraction'
    FF @('-i',$mp4,'-f','null','-')
    Assert ($true) 'complete-video-audio-decode'
    Assert ((Get-FileHash $png).Hash -eq $pngHash -and (Get-FileHash $jpeg).Hash -eq $jpegHash -and (Get-FileHash $mp4).Hash -eq $mp4Hash) 'original-hashes-unchanged'

    $bad = Join-Path $root 'malformed.mp4'
    [IO.File]::WriteAllText($bad, 'deliberately invalid media')
    & $ffprobe -v quiet -show_format -of json $bad | Out-Null
    Assert ($LASTEXITCODE -ne 0) 'malformed-media-rejected'
    # Windows PowerShell Actions wrapper otherwise propagates the expected failure.
    $global:LASTEXITCODE = 0
    [ordered]@{
        scope = 'Candidate only; synthetic JPEG/PNG/MPEG4-Part2/AAC; no Windows 10, service-account, real-camera, HEIC, RAW, HDR, H264/HEVC or AI validation'
        os = (Get-CimInstance Win32_OperatingSystem).Caption
        version = $version
        license = $license
        bundledDlls = @(Get-ChildItem (Split-Path $ffmpeg) -Filter *.dll | Select-Object -ExpandProperty Name)
        sha256 = $expected
        checks = $checks
    } | ConvertTo-Json -Depth 5
} finally {
    Remove-Item $root -Recurse -Force -ErrorAction Continue
}
