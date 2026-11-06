# Automatic naming: prototype acceptance evidence

The frozen policy is cosine0.65, competitor lead0.15, and two manually confirmed
references from different original asset hashes. The weaker reference must meet
the threshold. Another identity's strongest reference can veto the assignment.
Automatic labels never become references without explicit user confirmation.
Scores are similarity values, not confidence probabilities. No threshold,
reference choice, corpus split or pass criterion was changed after inference.

## Executed policy screens

| Screen | Known correct / total | Known wrong | Known unresolved | Unknown false assignments | Unknown rejected | Unknown unresolved |
|---|---:|---:|---:|---:|---:|---:|
| ORL separate-policy screen |147/160|0|13|0|100|0|
| Independent Nottingham |129/181|0|52|0|239|6|

ORL: source `cc34b4d9eef39aa0a092893e1f14019edab5ebe5`,
[run42158326967](https://github.com/aataat12/immich-clone/actions/runs/42158326967),
native job127697698876. ORL was previously studied for review suggestions; this
is supporting separate-policy evidence, not the independent source.

Nottingham: source `d6cfd0968d5c5b8d9671520fcabeb71dd1f7254a`,
[run42164599554](https://github.com/aataat12/immich-clone/actions/runs/42164599554),
job127718001981. All72 fixed references were available. Of495 present originals,
485 produced exactly one face and10 none; no multiple detections. Known frontal
expression queries:69/72 correct,0wrong,3unresolved, with71 detections. Angled
queries44/72 correct; cap queries16/36 correct. Unknowns had239 detections, all
rejected;6unresolved include3missing originals and3without a detected face.
All original denominators remain visible. All predeclared gates passed.

The [frozen protocol](../experiments/automatic-names-nottingham/PROTOCOL.md)
records original archive pin, deterministic identity split and all pass gates.
Credit: [PICS, University of Stirling](https://pics.stir.ac.uk/) and its
[Nottingham original dataset](https://pics.stir.ac.uk/2D_face_sets.htm).
Both native runs passed original integrity, three flat plus natural no-face
controls and scratch cleanup. Only aggregate results were retained. No portraits,
embeddings, per-image predictions or research corpus enter product packages.

FEI has no recognition result. Its official host refused cloud access before
inference, including the normal public-page/session flow. Those attempts are not
failed recognition outcomes and contribute no accuracy evidence. No access
bypass, alternate identity, mirror or corpus transfer was used.

## Decision and limits

This evidence supports enabling the unchanged conservative policy for the
reversible0.1.43 prototype, subject to actual enabled-publication and combined
release acceptance. It supplies independent identities, expressions, pose and
cap observations from a corpus unused in this project's policy selection.
Upstream model-training overlap is unknown. Controlled monochrome portraits do
not establish general color/family-library accuracy, children, age progression,
family resemblance or unconstrained lighting/background performance. The release
must retain uncertainty, visible automatic provenance, correction/undo and the
manual-reference-only rule; do not advertise universal accuracy.

[Catalog compatibility](release-0.1.43-catalog-compatibility.md) prevents older
writers from losing automatic provenance. Enabled photo/video/reanalysis hooks,
real native assignment lifecycle, UI confirmation, restart and independent recovery
remain required functional evidence before packaging. Model/provider/dependencies
are unchanged; no new charge, product training or cloud exposure is introduced.
