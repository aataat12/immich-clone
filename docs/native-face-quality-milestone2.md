# Milestone 2 — current production Go/native cross-photo quality screen

This is a bounded, identity-disjoint quality evaluation of the current production recognizer and suggestion helper, not a general accuracy guarantee, model upgrade, retraining or automatic naming feature. It extends the earlier Python model-only screen through the shipped Go preprocessing, hash-verified patched native provider and production suggestion threshold/margin. Corrections still require user review.

## Frozen protocol

The official AT&T Laboratories Cambridge / Olivetti Research Laboratory (ORL) archive, URL, SHA256 and terms/credit are retained in `experiments/face-quality/PROTOCOL.md`. This experiment downloads it solely for research evaluation; no portraits, archive, raw embeddings or individual predictions are committed, packaged or uploaded as artifacts. The archive's exact SHA256 is checked before and after all inference.

- Exactly400 images,40 subjects with10 distinct photos each, original92×112 grayscale. Decode the fixed PGM shape and perform a declared3× Go `golang.org/x/image/draw.BiLinear` resize to276×336 RGBA, then call the actual `recognition.Provider.Analyze` pipeline, including production Frame/BGR conversion and both native face/object tasks. No manual face crops or fallback landmarks.
- The3× interpolation uses Go's Bilinear implementation rather than OpenCV INTER_LINEAR. This declared preprocessing difference is part of the current Go screen; results are not passed off as bit-identical to the earlier Python run. No preprocessing changes in reaction to held-out results.
- Subjects1–10 are the separate calibration cohort. Compute every available different-subject cosine; require its maximum plus0.02 to remain at or below the already selected production threshold0.363. The current threshold is fixed before the run and is never recalibrated from held-out results.
- Enroll only photograph1 for subjects11–30. Their photographs2–10 form180 known-person queries. Subjects31–40 are never enrolled and supply100 unknown queries. Missing detections remain in the denominators; no better enrollment photo is selected after failure.
- Invoke the production `suggestReviewFace` helper with the current0.363 threshold and0.05 lead over the runner-up identity. The same named-reference aggregation and provider-compatibility rule applies. No suggestion is applied as a person name.
- Report aggregate detection counts, calibration pairs/max-negative, all180 known outcomes, all100 unknown outcomes, enrollment coverage, scored/unscored same/different pairs and pair threshold errors. Zero/one/multiple face outcomes are not silently omitted.
- Existing narrow gate: at least360/400 exactly-one-face detections, at least144/180 correct known queries, zero wrong known identities and zero unknown false accepts; calibration separation must pass. Three flat grayscale no-face controls and the pinned natural coffee scene must also produce no face detections.

No threshold, margin, model, input selection or acceptance gate is changed to force a passing result. The provider ID remains `opencv-4.12.0-nr01-zlib-1.3.2-15d1c2ca7ef3-bgr960-v1`, with existing weights and worker hashes checked by production code. Root's extracted suggestion helper preserves the preexisting threshold/margin semantics.

## Executed Windows result

[Run41767731927](https://github.com/aataat12/immich-clone/actions/runs/41767731927), source `039b96dc25baeba3962e2abc84a755eb3603ea83`, job `126369964093`: PASS first execution, Windows Server2022, Go1.27.3 windows/amd64, CGO disabled, GOAMD64=v1. Actual production provider and root's unchanged production helper were executed, not replaced by a Python approximation or mocked embeddings.

| Measure | Observed result |
|---|---|
| Corpus detections |400/400 exactly one face:100 calibration,300 held out |
| Calibration negative pairs |4500, maximum cosine0.32311472554031656; plus0.02 remains below fixed0.363 |
| Known queries |180/180 correct,0 wrong,0 unresolved |
| Unknown queries |100/100 rejected,0 false accepts,0 unprocessed |
| Enrollment coverage |20/20 fixed first-photo enrollments |
| Same-person held-out pairs |1350 scored,0 below0.363,0 unscored |
| Different-person held-out pairs |43500 scored,**14 above0.363**,0 unscored |
| No-face controls |3/3 flat gray controls and the pinned natural coffee scene yielded no face detection |
| Original/scratch preservation |Archive unchanged; analysis scratch empty |
| Full test including cleanup |562.38seconds; reported inference/evaluation elapsed562.325seconds |

The14 accepted different-person pairs are a meaningful limitation: a blind cosine threshold alone can associate different people, even on this constrained corpus. They are **not** hidden by reporting only the end-to-end query result. The production runner-up margin and explicit human confirmation remain required; these results do not justify automatic clustering, merging or naming. No thresholds/models/inputs were changed after the result.

This was404 actual Provider.Analyze operations:400 corpus images plus4 no-face controls. Each follows the existing face and object task execution path. No portraits, embeddings or per-photo predictions were retained in CI artifacts. Local opt-in compilation/skip and vet passed before execution; the expensive native corpus run was not repeated without cause.

## Resource and privacy bounds

Test: `internal/library/native_face_quality_test.go`. Preparation uses only Python's standard library (`experiments/face-quality/prepare-native.py`) and the existing `scripts/Prepare-AIProvider.ps1`; no new runtime dependencies. The corpus and a pinned public coffee control are downloaded to disposable CI storage. `scripts/Test-NativeFaceQuality.ps1` removes all inputs/provider copies after completion and restores its environment. Individual analysis scratch must be empty at the end. Output contains aggregate metrics only.

Existing production resource limits are unchanged. The whole evaluation has a12-minute context,14-minute Go-test timeout and18-minute Windows-job timeout. The context is an evaluation resource bound, not a new per-image product limit. No GPU, cloud inference, paid infrastructure or user-PC workload is required.

## Limitations

ORL is a small historic grayscale mostly upright/frontal adult-face corpus with limited expression, illumination and glasses variation. It does not represent children, age progression, difficult pose, family resemblance, group pictures, diverse natural phone libraries or broad no-face scenes. Three flat controls and one coffee scene do not establish a real-world false-positive rate. Pair metrics and held-out end-to-end outcomes are both reported because neither can replace the other.

This run uses the actual Windows binary pipeline under the CI account. It is not Windows10 or installed LocalService evidence for every corpus image. Existing installed provider tests retain their separate scope. This completes the declared production-pipeline screening task without claiming universally correct recognition or silently enabling automatic identity merging/naming. Security Group2/3 work, Full Scan and deferred runtime isolation remain unchanged.
