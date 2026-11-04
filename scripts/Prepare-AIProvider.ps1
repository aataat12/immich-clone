param([Parameter(Mandatory=$true)][string]$Destination)
$ErrorActionPreference='Stop'
$workerHash='15d1c2ca7ef3f526b34ac4823f9337c51280ad171ddf43df4f3af5d8128a7c58'
if ((Get-FileHash 'native-ai-candidate/ai-worker.exe').Hash.ToLowerInvariant() -ne $workerHash) { throw 'AI worker checksum mismatch' }
foreach ($file in @('OpenCV-LICENSE.txt','SOURCE-PATCHES.txt','opencv-module-sources.zip','protobuf-upstream-0299c030.patch','UPSTREAM-SOURCE.txt','CMakeCache.txt','pe-imports.txt','worker-version.txt','zlib-1.3.2-source.tar.gz','zlib-1.3.2-LICENSE.txt','zlib-operation-evidence.txt','opencv-security-patches.json','opencv-patched-source-SHA256.txt','opencv-security-operation-evidence.txt')) { if (!(Test-Path "native-ai-candidate/$file")) { throw 'AI provenance missing' } }
if ((Get-FileHash 'native-ai-candidate/zlib-1.3.2-source.tar.gz').Hash.ToLowerInvariant() -ne 'bb329a0a2cd0274d05519d61c667c062e06990d72e125ee2dfa8de64f0119d16') { throw 'Replacement zlib source checksum mismatch' }
$nativeVersion=Get-Content 'native-ai-candidate/worker-version.txt' -Raw | ConvertFrom-Json
if ($nativeVersion.zlib -ne '1.3.2' -or $nativeVersion.opencv -ne '4.12.0' -or $nativeVersion.security_backports -ne 'nr01-v1') { throw 'Native provenance version mismatch' }
$securityEvidence=Get-Content 'native-ai-candidate/opencv-security-operation-evidence.txt' -Raw | ConvertFrom-Json
if (!$securityEvidence.passed -or $securityEvidence.nary_cases -ne 64) { throw 'Native security operation evidence missing' }
foreach ($patch in (Get-Content 'experiments/native-ai/security-patches/manifest.json' -Raw | ConvertFrom-Json)) {
 if ((Get-FileHash (Join-Path 'native-ai-candidate' $patch.file)).Hash.ToLowerInvariant() -ne $patch.sha256) { throw 'Retained OpenCV patch checksum mismatch' }
}
New-Item -ItemType Directory -Force $Destination | Out-Null
Get-ChildItem native-ai-candidate | Where-Object Name -ne 'native-security-check.exe' | Copy-Item -Destination $Destination -Recurse -Force
if (Test-Path (Join-Path $Destination 'native-security-check.exe')) { throw 'Test executable must not be packaged' }
$base='https://media.githubusercontent.com/media/opencv/opencv_zoo/48b10fa505aeb51fbd127fc60a5c660040e823b7/models/'
$models=@(
 @('yunet.onnx','face_detection_yunet','face_detection_yunet_2023mar.onnx','8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4'),
 @('sface.onnx','face_recognition_sface','face_recognition_sface_2021dec.onnx','0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79'),
 @('yolox.onnx','object_detection_yolox','object_detection_yolox_2022nov.onnx','c5c2d13e59ae883e6af3b45daea64af4833a4951c92d116ec270d9ddbe998063')
)
$cache=Join-Path $env:RUNNER_TEMP 'verified-ai-model-cache'
New-Item -ItemType Directory -Force $cache | Out-Null
$origins=@('OpenCV Zoo source commit 48b10fa505aeb51fbd127fc60a5c660040e823b7. Unmodified model weights; CPU only.',"Native worker SHA256 $workerHash")
foreach ($model in $models) {
 $cached=Join-Path $cache $model[0]
 if (!(Test-Path $cached)) { Invoke-WebRequest ($base+$model[1]+'/'+$model[2]) -OutFile $cached }
 if ((Get-FileHash $cached).Hash.ToLowerInvariant() -ne $model[3]) { throw 'AI model checksum mismatch' }
 Copy-Item $cached (Join-Path $Destination $model[0]) -Force
 $noticeRoot=Join-Path $Destination ('model-notices/'+$model[1]);New-Item -ItemType Directory -Force $noticeRoot | Out-Null
 foreach ($notice in @('LICENSE','README.md')) { Invoke-WebRequest ('https://raw.githubusercontent.com/opencv/opencv_zoo/48b10fa505aeb51fbd127fc60a5c660040e823b7/models/'+$model[1]+'/'+$notice) -OutFile (Join-Path $noticeRoot $notice) }
 $origins+=($model[0]+' SHA256 '+$model[3]+' SOURCE '+$base+$model[1]+'/'+$model[2])
}
$origins | Set-Content (Join-Path $Destination 'MODEL-ORIGINS.txt')
