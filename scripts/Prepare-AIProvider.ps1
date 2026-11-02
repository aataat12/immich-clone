param([Parameter(Mandatory=$true)][string]$Destination)
$ErrorActionPreference='Stop'
$workerHash='a83dcef02ebef33beccea8a3b21b88ed2bc18b298040f592be31ee9e72191ec6'
if ((Get-FileHash 'native-ai-candidate/ai-worker.exe').Hash.ToLowerInvariant() -ne $workerHash) { throw 'AI worker checksum mismatch' }
foreach ($file in @('OpenCV-LICENSE.txt','SOURCE-PATCHES.txt','opencv-module-sources.zip','protobuf-upstream-0299c030.patch','UPSTREAM-SOURCE.txt','CMakeCache.txt','pe-imports.txt')) { if (!(Test-Path "native-ai-candidate/$file")) { throw 'AI provenance missing' } }
New-Item -ItemType Directory -Force $Destination | Out-Null
Copy-Item native-ai-candidate/* $Destination -Recurse -Force
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
