# Predeclared cross-photo candidate screen

Recorded before model inference on this corpus. Credit: AT&T Laboratories
Cambridge / Olivetti Research Laboratory. Original official archive and README:
https://www.cl.cam.ac.uk/research/dtg/attarchive/facedatabase.html .
The publisher makes this research database downloadable and requests credit.
This experiment downloads the original archive for evaluation only. Do not
redistribute the images, embeddings or archive, or include them in the app.
No claim of a general commercial dataset license. No user data is involved.

The ORL/AT&T corpus has 40 anonymous subjects, ten distinct photographs each,
92x112 grayscale, mostly upright/frontal, homogeneous backgrounds, limited
expression/light/glasses variation. No children/age progression, broad pose,
family resemblance, real-library diversity or no-face scenes are represented.
This is an initial screening set, not final recognition/value acceptance.

Frozen inputs: archive SHA256
509d763bd2276aa054d260d7f1000e8be212738134f2f83d8ff13bc2adf1c8ec;
existing pinned YuNet/SFace weights and OpenCV/NumPy versions from ai-probe.
Preprocessing: decode exact source shape, three-times linear resize, BGR, YuNet
confidence 0.9, NMS 0.3; exactly one face required, otherwise unresolved. SFace
five-landmark 112x112 alignment; normalize finite 128-dimensional vector. No
fallback to fabricated landmarks or manual crop. Original hashes stay unchanged.

Split by identity, never random images from the same subject across calibration
and evaluation. Calibration: subjects 1–10, all ten photographs. Choose the first
threshold from [0.363, 0.4, 0.45, ..., 0.9, 1.000001] above every different-person
calibration-pair cosine by at least 0.02. Freeze before evaluating subjects 11–40.
The final threshold rejects everything if calibration cannot separate negatives.
No threshold/input changes in response to held-out results.

Evaluation: subjects 11–30 enrolled with photograph 1 only; photographs 2–10
are 180 known-person queries. Subjects 31–40 are never enrolled; all 100 images
are unknown-person queries. Missing enrollment/query detections remain unresolved
and count against end-to-end known-query accuracy. A match needs both the frozen
threshold and a 0.05 lead over the runner-up enrolled identity. No per-photo choice
of a better enrollment image. Report wrong IDs, unknown false accepts, unresolved,
coverage and scored positive/negative pair counts, not just successful detections.

Initial screen passes only with >=90% detection over all 400 images, >=80% correct
known queries out of all 180, zero wrong known IDs and zero unknown false accepts.
These thresholds gate this narrow screen only. Report the result even on failure;
a green workflow means evaluation executed, not necessarily that quality passed.
No further integration claim until the separate packaging and broader-quality gates.

Do not persist raw embeddings, portrait thumbnails or individual predictions in
CI artifacts. Aggregate anonymous metrics only. No import regression tests needed.
