param([Parameter(Mandatory=$true)][string]$Destination)
# CI-only synthetic fixture generator. No binary is bundled with the application.
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$root = Join-Path $env:RUNNER_TEMP ('media-fixture-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory $root | Out-Null
try {
    $archive = Join-Path $root 'tool.zip'
    Invoke-WebRequest https://github.com/BtbN/FFmpeg-Builds/releases/download/autobuild-2026-10-31-12-59/ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip -OutFile $archive
    if ((Get-FileHash $archive).Hash.ToLowerInvariant() -ne '5cef7bc48e513f3cee7c881e8b2999d835541edb1846c3e89443a67f8f8556ab') { throw 'Fixture tool checksum mismatch' }
    Expand-Archive $archive "$root/tool"
    $ffmpeg = (Get-ChildItem "$root/tool" -Filter ffmpeg.exe -Recurse | Select-Object -First 1).FullName
    & $ffmpeg -hide_banner -loglevel error -nostdin -n -f lavfi -i 'testsrc=size=160x120:rate=10' -f lavfi -i 'sine=frequency=440:sample_rate=44100' -t 1 -c:v libopenh264 -b:v 200k -pix_fmt yuv420p -c:a aac -movflags +faststart $Destination
    if ($LASTEXITCODE -ne 0) { throw 'Actual H264/AAC fixture generation failed' }
    Write-Output 'PASS: pinned FFmpeg executed H264/AAC encoding for a synthetic browser fixture.'
} finally {
    $resolved=[IO.Path]::GetFullPath($root)
    $parent=[IO.Path]::GetFullPath($env:RUNNER_TEMP).TrimEnd('\')+'\'
    if(!$resolved.StartsWith($parent,[StringComparison]::OrdinalIgnoreCase)){throw 'Fixture cleanup target refused'}
    Remove-Item -LiteralPath $resolved -Recurse -Force -ErrorAction Continue
}
