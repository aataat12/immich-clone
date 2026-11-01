param([Parameter(Mandatory=$true)][string]$Destination)
$ErrorActionPreference = 'Stop'
$expected = @{
    'ffmpeg.exe'='7eb77d88e2e97afff0844d8df1408ec10f9ca66fac58f6971112cefd25fb6a0f'
    'ffprobe.exe'='0ba24b17e5b42e02de441fb053dacca08aa52ac4f7f38e0845bda7df6b73a6e6'
    'ffmpeg-8.1.3-source.tar.gz'='57421c6be6861f26d1dc9edf384a9d1ab6410cc0b247b2502f415df83443fe4b'
}
foreach ($name in $expected.Keys) {
    if ((Get-FileHash (Join-Path 'media-candidate' $name)).Hash.ToLowerInvariant() -ne $expected[$name]) { throw 'Pinned media provider checksum mismatch' }
}
foreach ($notice in @('COPYING.LGPLv2.1','LICENSE.md','compiler-copyright.txt','mingw-copyright.txt','configure-command.txt','build-tool-versions.txt','source-commit.txt')) {
    if (!(Test-Path (Join-Path 'media-candidate' $notice))) { throw 'Required media source/license evidence missing' }
}
New-Item -ItemType Directory -Force $Destination | Out-Null
Copy-Item media-candidate/* $Destination
Copy-Item experiments/media-worker/build.sh (Join-Path $Destination 'build.sh')
