# People and object organization: priority and acceptance

User deferred further import development/testing. Recognition/organization
usefulness is now the project value gate. Retain accepted 0.1.13 work unchanged.
AI stays local, optional and replaceable; failures cannot disable original access.
No personal media, names, source paths or PC details enter public CI.

## Intended behavior

People: detect faces, suggest recurring-person groups; user supplies names,
reviews uncertain matches and corrects assignments with merge/split controls.
Objects: searchable suggested tags/filters with user corrections. Initial detector
covers 80 COCO classes; broad scenes/natural-language search needs a separately
proven embedding model. Grouping never moves or rewrites original files. Manual
names/corrections must survive model replacement and independent backup/restore.

## Acceptance gates

1. Actual CPU inference: face boxes, five-landmark alignment, finite embeddings,
   positive/negative comparisons; object tensors/boxes/classes and correct input
   preprocessing. Execute under Windows LocalService/restart with bounded memory,
   deadline/output and unchanged input hashes. Compilation is insufficient.
2. Quality: predeclare labeled same/different-person pairs from multiple distinct
   photos per person, varied pose/light/age/occlusion and multiple/no-face scenes.
   Separate threshold calibration from held-out evaluation; report false merges,
   misses and retrieval results, with conservative unknown assignments. Transformed
   copies of one portrait are not cross-photo identity evidence. Public fixtures
   need usable rights; private validation stays on the PC.
3. Packaging: selected shipping runtime/DLLs/CPU baseline/minimum Windows, model
   terms/provenance/notices, security/update/replacement path, clean prerequisites,
   timeout/crash/model absence and actual target execution. Server/hosted Python
   availability does not establish Windows 10 or standalone packaging.
4. Small vertical slice: durable AI jobs, People page, naming/corrections, object
   tags/search, model-versioned rebuild and independent restore. User evaluates
   usefulness privately before returning to bulk import.

## Bounded candidate experiment

`experiments/ai-probe` is separate from application dependencies. OpenCV Zoo
commit `48b10fa505aeb51fbd127fc60a5c660040e823b7`; immutable model/fixture URLs
and SHA256 in `prepare.py`, wheel hashes in `requirements.txt`. No weights,
images, wheels or raw embeddings are committed or published as artifacts.

| Component | Candidate/published terms | Remaining gate |
|---|---|---|
| Face detector | YuNet 2023mar; model directory MIT | Broader face quality |
| Face embedding | SFace 2021dec; model directory Apache-2.0 | Cross-photo evaluation; upstream cosine 0.363 is not library-calibrated |
| Object detector | YOLOX 2022nov; model directory Apache-2.0 | 80 COCO categories; no scene/semantic-search claim |
| Inference | OpenCV DNN CPU, opencv-python-headless 4.12.0.88; packaging MIT, OpenCV Apache-2.0, transitive notices include LGPL FFmpeg | Native closure, distribution/security audit; not selected for app |
| Arrays | NumPy 2.2.6; BSD-3-Clause plus third-party notices | Not selected for app |
| Interpreter | Windows Python 3.12.10 cloud harness; Linux host 3.12.14 | Shipping runtime undecided; no local dependency request |

Windows probe reuses the tested creation-time Job Object launcher: 512MiB
commitment, one process, kill-on-close, 90-second experiment deadline, 1MiB
output. Resource limits are not a filesystem/token sandbox or throughput SLA.
OpenCV uses one thread, CPU target and no OpenCL. Downloads happen before inference;
probe inference makes no network calls.

Public fixtures: scikit-image astronaut (NASA/public-domain), Chelsea cat (CC0,
Stefan van der Walt), coffee (CC0, Rachel Michetti/Pikolo Espresso Bar), and
Matplotlib's Grace Hopper portrait (US Navy/James S. Davis, public-domain source).
SHA256 verified before/after inference. Never the user's photos.

Initial Linux cloud inference passed: resized/dimmed same-portrait cosine 0.941487
versus different-people 0.151917; coffee/cat produced no human-face detection.
Expected person/cat/cup objects were found (coffee also had dining-table class).
About 254ms for face operations and 435–675ms per object fixture on the disposable
host; not PC/throughput predictions. This does not clear broad recognition quality,
packaging or app integration. Windows service evidence pending.

## Sources and update policy

- https://github.com/opencv/opencv_zoo/tree/48b10fa505aeb51fbd127fc60a5c660040e823b7/models/face_detection_yunet
- https://github.com/opencv/opencv_zoo/tree/48b10fa505aeb51fbd127fc60a5c660040e823b7/models/face_recognition_sface
- https://github.com/opencv/opencv_zoo/tree/48b10fa505aeb51fbd127fc60a5c660040e823b7/models/object_detection_yolox
- https://pypi.org/project/opencv-python-headless/4.12.0.88/
- https://numpy.org/doc/2.2/license.html
- https://scikit-image.org/docs/0.25.x/api/skimage.data.html
- https://commons.wikimedia.org/wiki/File:Grace_Hopper.jpg (attribution/rights)

Directory licenses establish published terms, not an independent audit of all
training-data rights. Resolve specific distribution restrictions and retain exact
notices before shipping. No fixed model/runtime support SLA assumed: review
upstream security updates, pin replacements and rerun inference/quality/compatibility.
