# Frozen independent Nottingham automatic-name evaluation

Recorded before Nottingham native inference. The existing candidate stays fixed:
cosine0.65, competitor lead0.15, two manually confirmed distinct originals.
FEI has no recognition results: two cloud attempts were refused with403 before
inference. No further FEI retry, mirror, proxy, transfer or result is used here.

## Official source, research permission and integrity

The University of Stirling's [PICS source](https://pics.stir.ac.uk/) explicitly
permits research use and requests attribution to that URL. Its
[dataset page](https://pics.stir.ac.uk/2D_face_sets.htm) publishes the original
Nottingham archive and describes71 identities, four frontal expressions, two
three-quarter expressions and a frontal bathing-cap view, monochrome288x384.
Credit: Psychological Image Collection at Stirling / University of Stirling,
Nottingham original face set. No registration or contact is submitted.

Use only `https://pics.stir.ac.uk/zips/nottingham_originals.zip`:
**38347802 bytes**, SHA256
`b97fecf7c3e8250518a190aa265e617fbd916e14a2bbbcd79a58c65bf1da190e`.
Pin established through archive metadata/header inspection only: no rendering or
native inference.495 GIFs,38832533 uncompressed bytes, maximum116279 per entry,
all288x384. Retain inputs only in disposable runner storage; never commit or
upload portraits, archive, embeddings, individual predictions or subject metrics.
Only anonymous aggregate metrics may enter logs. No redistribution is needed.

The source presents examples `m17-1.gif` through `m17-7.gif` in its described
sequence. We infer configuration1-4 as frontal expressions,5/6 as three-quarter
and7 as cap from that published order; no measured-angle claim is made.
The exact numerical configurations remain reported even if that inference is
imperfect. Reference1/2 is fixed before results; no better image is selected.

## Frozen split and incomplete records

Take every one of the71 IDs in the pinned archive. Sort by ascending SHA256 of
ASCII `nottingham-id-v1:` plus the decimal ID (without leading zeroes). The first
36 IDs are known; the remaining35 are unknown. This deterministic hash order
avoids incidental separation by acquisition/filename order and was chosen before
inference, without inspecting portraits or predictions. There is no calibration
cohort or selection by demographics, successful detection or similarity score.

Enroll exactly configurations1 and2 of each known ID:72 fixed references.
Known queries are all remaining configurations3-7, plus the one extra original
`m39-8.gif`. The archive lacks `m5-4.gif`, `m5-5.gif` and `m13-4.gif`; these three
scheduled observations remain unresolved in the denominators. All181 present
known query files and all242 present unknown files are analyzed. Consequently:

- Native corpus operations:495 present originals, including72 references.
- Scheduled known queries:181, all present.
- Known configurations3/4:72 scheduled, all present.
- Known configurations5/6:72 scheduled, all present.
- Known configuration7:36 present; extra configuration8:1 present.
- Unknowns:35 identities x7 configurations =245 scheduled queries:242 present
  and3 missing originals; never enrolled.

No reference is replaced after detection failure. Zero/multiple detections remain
unresolved. All queries, including incomplete records, are accounted for. Compare
each exact-one-face observation using the unchanged production candidate matcher;
no automatic result enters the reference set. Use actual image SHA256 as source
identity so duplicate bytes cannot supply independent references.

## Execution and predeclared gates

Use the same pinned Go/native provider/model hashes, production GIF first-frame
decode, EXIF-orientation path and native inference. No manual landmark/crop,
resizing, training, new model/provider/package, GPU or paid service. The archive
is read without extraction, verified before/after, and provider scratch must empty.
Run the existing three flat plus pinned natural coffee no-face controls as well.

All gates must pass unchanged:

1.72/72 fixed references; at least420/495 present originals produce exactly one
   face. Unknowns need at least210/245 exact-one-face detections.
2. Known configurations3/4: at least66/72 detections and54/72 correct assignments.
   All known queries: at least90/181 correct. Missing files/detections count
   against those denominators, establishing non-vacuous useful coverage.
3. **Zero wrong known identities** across all181 scheduled queries, and **zero
   unknown automatic identities** across all245 unknown queries. Rejections stay
   unnamed. Report configuration groups and every unresolved/missing outcome.
4. All four no-face controls, original integrity and scratch cleanup pass.

No adjustment to policy, references, corpus, split or gates follows an outcome.
Failures are evidence; do not retry inference without an identified operational
failure. The cloud workflow first executes the same bounded acquisition and hash
checks; if its official source refuses access, stop without fallback/inference.
Download limit64MiB, per GIF512KiB; native context18min, Go test20min, job25min.
No artifact-upload step exists. Disposable corpus/provider directories are removed
in `finally`, including failures. Synthetic logic/provenance, actual publication
hooks and prior-writer compatibility remain separate release requirements.

## Interpretation

This source was not used to choose the candidate policy. It adds independent
identity/expression/pose/cap evidence beyond ORL. Monochrome controlled portraits
do not establish color-library, age-progression, children, family-resemblance or
unconstrained lighting/background accuracy. Numerical splits are not a demographic
balance guarantee. Model-pretraining overlap is unknown; independence refers to
this project's policy-selection process. Scores are not confidence probabilities.
Passing supports only the bounded reversible prototype behavior, not universal
recognition or Windows10 certification, and does not change the production gate.
