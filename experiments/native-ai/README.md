# Native AI worker candidate

Separate feasibility experiment, not linked into the app or an installer.
Same YuNet/SFace/YOLOX weight IDs and hashes as `../ai-probe/prepare.py`.
OpenCV 4.12.0 source commit 49486f61fb25722cbcf586b7f4320921d46fb38e,
ZIP SHA256 a7382f4e2fd33b987f09f89866c50416a7b3647a95bab207e0dc271bab607999.
This version preserves parity with the accepted model experiments. Current patch/
security review remains a gate before choosing a shipping dependency; not a claim
that an old release has a support SLA. No latest/floating source or binary input.

Build on disposable Windows Server 2022 with Visual Studio 2022, CMake, /MT
static C/C++ runtime, static OpenCV core/imgproc/dnn/objdetect and required internal
modules. SSE2 baseline, dispatch disabled, no IPP/OpenCL/CUDA/OpenVINO/GPU, GUI,
Python bindings or media codecs. The worker reads a bounded working-frame format;
not the user's compressed original. No image/video parsing dependencies are added.
Pure-Go/accepted-media preprocessing into these frames remains an integration gate.

Protocol: `ai-worker.exe faces|objects MODEL_DIRECTORY FRAME_FILE`.
ICAI input: four ASCII magic bytes, uint32 little-endian width and height, then
exactly width*height*3 BGR8 bytes, 32..1600 each dimension, no trailing bytes.
This dimension bound is not a guarantee that every frame fits the memory budget.
All inference is CPU/one OpenCV thread. Model reads use buffers and native wide
paths; output is protocol-1 JSON with model results, never paths. No original
writes or inferred names. Face output includes normalized 128-dimensional vectors;
those are private data in future use and must not enter diagnostic logs.

Build script verifies the exact upstream archive. The compact artifact retains
its immutable source URL/SHA256, all third-party license texts, top-level OpenCV
Apache-2.0 notice, complete compiled module sources preserving embedded BSD/
SoftFloat/FDLIBM/FLANN notices, CMake cache, logs, worker source,
compiler/build provenance, PE imports and checksums. Bundled protobuf/zlib/etc.
notices and shipping redistribution/security audit remain to be completed; no
model weights, fixture media, raw embeddings or personal data enter the artifact.
No fixed vendor support horizon is assumed; review pinned updates and rerun proofs.

Acceptance: native face vectors vs independent Python goldens, positive/negative
faces and object classes on public fixtures, malformed frames, absent models,
worker cancellation/recovery, unchanged input hashes and LocalService restart.
Use existing creation-time 512MiB/one-process Windows Job Object and bounded
output/deadline launcher. Native child environment has only its own directory in
PATH and SystemRoot; no Python/development runtime assumed by inference.
A PE import audit must find only explicitly allowed Windows system DLLs.
This is not a clean Windows 10 or full security/filesystem sandbox claim.

References: https://docs.opencv.org/4.13.0/db/d05/tutorial_config_reference.html
and https://github.com/opencv/opencv/wiki/CPU-optimizations-build-options .
Native build and actual operations are pending until linked evidence is recorded.
Build checkpoint artifacts are explicitly not accepted; only a passed separate
service job can establish operation evidence. Retain builds to avoid recompiling
unchanged dependencies for a service-harness-only correction.


Security-review notes (not shipping clearance): upstream 4.14.0 is available,
so this 4.12.0 parity baseline is not an update policy. The documented 2025
OpenCV JPEG2000 bug was fixed in 4.12 and imgcodecs is excluded here. Bundled
protobuf identifies as 3.19.1; the three parser files receive official backport
0299c03005fbfe086d8394fb7a873a8a21fe327f (CVE-2022-1941), checksum-pinned and
recorded in SOURCE-PATCHES.txt. Clean applicability was tested in the Linux cloud
workspace. Further current patch/advisory review remains a shipping gate. This experiment uses only previously SHA256-pinned
trusted ONNX inputs under the existing job limits, never user-supplied models.
Do not extend the candidate to arbitrary model loading or app distribution until
these review gates are resolved. References:
https://github.com/opencv/opencv/wiki/OpenCV-Change-Logs
https://securitylab.github.com/advisories/GHSL-2025-057_OpenCV/
https://github.com/protocolbuffers/protobuf/security/advisories/GHSA-8gq9-2x98-w8hf
