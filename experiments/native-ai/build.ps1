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
# Backport the reviewed NR-01 fixes to the exact retained base, without new models/ABI.
$securityPatches = Get-Content experiments/native-ai/security-patches/manifest.json -Raw | ConvertFrom-Json
foreach ($item in $securityPatches) {
  $file = Join-Path 'experiments/native-ai/security-patches' $item.file
  if ((Get-FileHash $file).Hash.ToLowerInvariant() -ne $item.sha256) { throw 'OpenCV security patch checksum mismatch' }
  Checked git @('-C',$source,'apply','--check','--exclude=modules/*/test/*',(Resolve-Path $file).Path)
  Checked git @('-C',$source,'apply','--exclude=modules/*/test/*',(Resolve-Path $file).Path)
  Copy-Item $file $out
}
Copy-Item experiments/native-ai/security-patches/manifest.json "$out/opencv-security-patches.json"
$patchedPaths = @('modules/dnn/src/layers/nary_eltwise_layers.cpp','modules/core/src/system.cpp','modules/core/src/arithm.simd.hpp','modules/core/src/count_non_zero.simd.hpp','modules/core/include/opencv2/core/types.hpp','modules/core/src/matrix_expressions.cpp')
$patchedPaths | ForEach-Object { (Get-FileHash (Join-Path $source $_)).Hash.ToLowerInvariant()+'  '+$_ } | Set-Content "$out/opencv-patched-source-SHA256.txt"
# Official 3.19.x backport for CVE-2022-1941; only the three parser files.
$patch = Join-Path $out 'protobuf-upstream-0299c030.patch'
Invoke-WebRequest 'https://github.com/protocolbuffers/protobuf/commit/0299c03005fbfe086d8394fb7a873a8a21fe327f.patch' -OutFile $patch
if ((Get-FileHash $patch).Hash.ToLowerInvariant() -ne '5747ab6cf1dcdefc649fbfc6b8db1fdce546683c2ebad84519d3887944fd582e') { throw 'Backport checksum mismatch' }
$patchArgs = @('-C',"$source/3rdparty/protobuf",'apply','--include=src/google/protobuf/extension_set_inl.h','--include=src/google/protobuf/wire_format.cc','--include=src/google/protobuf/wire_format_lite.h')
Checked git ($patchArgs + @('--check',$patch))
Checked git ($patchArgs + @($patch))
'Bundled protobuf source modified by the checksum-pinned upstream CVE-2022-1941 backport in protobuf-upstream-0299c030.patch; three parser files only. Not a general security clearance.' | Set-Content "$out/SOURCE-PATCHES.txt"
'OpenCV security revision nr01-v1: upstream PRs 28272,28446,28579,28581,28959,29447; full pinned patches retained, production hunks applied, upstream test hunks excluded. Exact altered source digests in opencv-patched-source-SHA256.txt. Dedicated regression executable links the same selected modules.' | Add-Content "$out/SOURCE-PATCHES.txt"
# Prove the replacement compression library before compiling the worker.
$zlibArchive = Join-Path $out 'zlib-1.3.2-source.tar.gz'
Invoke-WebRequest 'https://zlib.net/zlib-1.3.2.tar.gz' -OutFile $zlibArchive
if ((Get-FileHash $zlibArchive).Hash.ToLowerInvariant() -ne 'bb329a0a2cd0274d05519d61c667c062e06990d72e125ee2dfa8de64f0119d16') { throw 'zlib source checksum mismatch' }
Checked tar @('-xzf',$zlibArchive,'-C',$root)
$zlibSource = Join-Path $root 'zlib-1.3.2'
$zlibBuild = Join-Path $root 'zlib-build'
$zlibInstall = Join-Path $root 'zlib-install'
Checked cmake @('-S',$zlibSource,'-B',$zlibBuild,'-G','Visual Studio 17 2022','-A','x64','-DZLIB_BUILD_SHARED=OFF','-DZLIB_BUILD_STATIC=ON','-DZLIB_BUILD_TESTING=OFF',"-DCMAKE_INSTALL_PREFIX=$zlibInstall",'-DCMAKE_MSVC_RUNTIME_LIBRARY=MultiThreaded')
Checked cmake @('--build',$zlibBuild,'--config','Release','--parallel','2')
Checked cmake @('--install',$zlibBuild,'--config','Release')
$checkBuild = Join-Path $root 'zlib-check'
Checked cmake @('-S','experiments/native-ai/zlib-check','-B',$checkBuild,'-G','Visual Studio 17 2022','-A','x64',"-DZLIB_INCLUDE_DIR=$zlibInstall/include","-DZLIB_LIBRARY=$zlibInstall/lib/zs.lib","-DZLIB_LIBRARY_RELEASE=$zlibInstall/lib/zs.lib")
Checked cmake @('--build',$checkBuild,'--config','Release','--parallel','2')
$check = & "$checkBuild/Release/zlib-check.exe"
if ($LASTEXITCODE -ne 0) { throw 'Actual zlib operations failed' }
$check | Set-Content "$out/zlib-operation-evidence.txt"
Copy-Item "$zlibSource/LICENSE" "$out/zlib-1.3.2-LICENSE.txt"
Copy-Item experiments/native-ai/zlib-check/zlib-smoke.cpp $out
'External zlib 1.3.2 source SHA256 bb329a0a2cd0274d05519d61c667c062e06990d72e125ee2dfa8de64f0119d16. OpenCV bundled zlib is disabled; static external library selected.' | Add-Content "$out/SOURCE-PATCHES.txt"
$build = Join-Path $root 'build'
$argsConfigure = @('-S','experiments/native-ai/worker','-B',$build,'-G','Visual Studio 17 2022','-A','x64',"-DOPENCV_SOURCE=$source","-DPINNED_ZLIB_ROOT=$zlibInstall",'-DCMAKE_CONFIGURATION_TYPES=Release',"-DOPENCV_DOWNLOAD_PATH=$root/downloads")
& cmake @argsConfigure *> "$out/configure.log"
if ($LASTEXITCODE -ne 0) { Get-Content "$out/configure.log" -Tail 100; throw 'Native configure failed' }
$job = Start-Job -ArgumentList $build,"$out/build.log" -ScriptBlock {
  param($directory,$log)
  & cmake --build $directory --config Release --target ai-worker native-security-check --parallel 2 *> $log
  $LASTEXITCODE
}
while ($job.State -eq 'Running' -or $job.State -eq 'NotStarted') {
  Start-Sleep -Seconds 30
  if (Test-Path "$out/build.log") { Get-Content "$out/build.log" -Tail 1 }
}
$compileResult = @(Receive-Job $job -Wait)
if ($job.State -ne 'Completed' -or $compileResult.Count -ne 1 -or $compileResult[0] -ne 0) { Get-Content "$out/build.log" -Tail 100; throw 'Native compile failed' }
Remove-Job $job
if (Test-Path "$build/opencv/3rdparty/lib/Release/zlib.lib") { throw 'Unexpected fallback to old bundled zlib' }
Copy-Item "$build/Release/ai-worker.exe","$build/Release/native-security-check.exe" $out
$securityResult = & "$out/native-security-check.exe"
if ($LASTEXITCODE -ne 0) { throw 'Actual OpenCV security regressions failed' }
$securityResult | Set-Content "$out/opencv-security-operation-evidence.txt"
Get-Content "$out/opencv-security-operation-evidence.txt"
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
Copy-Item experiments/native-ai/build.ps1,experiments/native-ai/worker/CMakeLists.txt,experiments/native-ai/worker/worker.cpp,experiments/native-ai/worker/security-check.cpp $out
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
$workerVersion = & "$out/ai-worker.exe" --version
if ($LASTEXITCODE -ne 0 -or (($workerVersion | ConvertFrom-Json).zlib -ne '1.3.2' -or ($workerVersion | ConvertFrom-Json).security_backports -ne 'nr01-v1')) { throw 'Worker linked zlib identity mismatch' }
$workerVersion | Set-Content "$out/worker-version.txt"
Get-FileHash "$out/ai-worker.exe" | ForEach-Object { $_.Hash.ToLowerInvariant()+'  '+(Split-Path $_.Path -Leaf) } | Set-Content "$out/SHA256SUMS.txt"
