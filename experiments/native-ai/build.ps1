$ErrorActionPreference = 'Stop'
function Checked([string]$exe,[string[]]$a) { & $exe @a; if ($LASTEXITCODE -ne 0) { throw 'Native build command failed' } }
$root = Join-Path $env:RUNNER_TEMP 'native-ai-build'
$out = Join-Path $env:GITHUB_WORKSPACE 'native-ai-candidate'
New-Item -ItemType Directory $root,$out -Force | Out-Null
$archive = Join-Path $out 'opencv-4.12.0-source.zip'
Invoke-WebRequest 'https://github.com/opencv/opencv/archive/49486f61fb25722cbcf586b7f4320921d46fb38e.zip' -OutFile $archive
if ((Get-FileHash $archive).Hash.ToLowerInvariant() -ne 'a7382f4e2fd33b987f09f89866c50416a7b3647a95bab207e0dc271bab607999') { throw 'OpenCV source checksum mismatch' }
Expand-Archive $archive "$root/source"
$source = Join-Path $root 'source/opencv-49486f61fb25722cbcf586b7f4320921d46fb38e'
$build = Join-Path $root 'build'
$argsConfigure = @('-S','experiments/native-ai/worker','-B',$build,'-G','Visual Studio 17 2022','-A','x64',"-DOPENCV_SOURCE=$source",'-DCMAKE_CONFIGURATION_TYPES=Release',"-DOPENCV_DOWNLOAD_PATH=$root/downloads")
& cmake @argsConfigure *> "$out/configure.log"
if ($LASTEXITCODE -ne 0) { Get-Content "$out/configure.log" -Tail 100; throw 'Native configure failed' }
& cmake --build $build --config Release --target ai-worker --parallel 2 *> "$out/build.log"
if ($LASTEXITCODE -ne 0) { Get-Content "$out/build.log" -Tail 100; throw 'Native compile failed' }
Copy-Item "$build/Release/ai-worker.exe" $out
Copy-Item "$build/CMakeCache.txt" "$out/CMakeCache.txt"
Copy-Item "$source/LICENSE" "$out/OpenCV-LICENSE.txt"
# Retain transitive notices without a redundant large source ZIP artifact.
Get-ChildItem "$source/3rdparty" -Recurse -File | Where-Object { $_.Name -match '(?i)license|copying|copyright' } | ForEach-Object {
  $relative = $_.FullName.Substring($source.Length+1)
  $destination = Join-Path "$out/notices" $relative
  New-Item -ItemType Directory (Split-Path $destination) -Force | Out-Null
  Copy-Item $_.FullName $destination
  $relative
} | Set-Content "$out/third-party-license-index.txt"
'https://github.com/opencv/opencv/archive/49486f61fb25722cbcf586b7f4320921d46fb38e.zip SHA256 a7382f4e2fd33b987f09f89866c50416a7b3647a95bab207e0dc271bab607999' | Set-Content "$out/UPSTREAM-SOURCE.txt"
# Some compiled modules embed BSD/SoftFloat/FDLIBM/FLANN notices in source
# headers instead of standalone LICENSE files. Retain those complete modules.
$moduleSources = @('core','imgproc','dnn','objdetect','calib3d','features2d','flann') | ForEach-Object { Join-Path "$source/modules" $_ }
Compress-Archive -Path $moduleSources -DestinationPath "$out/opencv-module-sources.zip"
Copy-Item experiments/native-ai/build.ps1,experiments/native-ai/worker/CMakeLists.txt,experiments/native-ai/worker/worker.cpp $out
$vs = & "${env:ProgramFiles(x86)}/Microsoft Visual Studio/Installer/vswhere.exe" -latest -products '*' -property installationPath
$dumpbin = Get-ChildItem "$vs/VC/Tools/MSVC/*/bin/Hostx64/x64/dumpbin.exe" | Sort-Object FullName | Select-Object -Last 1 -ExpandProperty FullName
$imports = & $dumpbin /dependents "$out/ai-worker.exe"
if ($LASTEXITCODE -ne 0) { throw 'PE audit failed' }
$dependencies = @($imports | ForEach-Object { if ($_ -match '^\s+([A-Za-z0-9_.-]+\.dll)\s*$') { $Matches[1].ToLowerInvariant() } } | Sort-Object -Unique)
if ($dependencies.Count -eq 0) { throw 'Empty PE audit' }
$allowed = @('kernel32.dll','user32.dll','advapi32.dll','ole32.dll','oleaut32.dll','ws2_32.dll','shell32.dll','bcrypt.dll','ntdll.dll','gdi32.dll','comdlg32.dll','comctl32.dll','winmm.dll','version.dll','shlwapi.dll')
foreach ($dll in $dependencies) { if ($dll -notin $allowed) { throw "Unexpected native dependency: $dll" } }
$dependencies | Set-Content "$out/pe-imports.txt"
(& cmake --version) | Set-Content "$out/build-tool-versions.txt"
(& $dumpbin /headers "$out/ai-worker.exe" | Select-Object -First 12) | Add-Content "$out/build-tool-versions.txt"
Get-Content "$out/pe-imports.txt"
Checked "$out/ai-worker.exe" @('--version')
Get-FileHash "$out/ai-worker.exe" | ForEach-Object { $_.Hash.ToLowerInvariant()+'  '+(Split-Path $_.Path -Leaf) } | Set-Content "$out/SHA256SUMS.txt"
