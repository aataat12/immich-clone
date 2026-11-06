# Automatic known-person naming: frozen candidate screen

Recorded before executing the new policy. This screen does not inherit approval
from the review-suggestion evaluation. It tests the production candidate matcher
directly with actual production native embeddings and does not enable naming.

Use the same pinned ORL archive, provider, native preprocessing, privacy/cleanup
bounds and attribution as `PROTOCOL.md` and the existing native quality screen.
No new dependency, model, provider, paid resource, user photo, raw embedding or
individual prediction is required or retained. ORL has already been studied for
suggestions: this is a separate-policy screen on a previously studied corpus,
not a fresh generalization claim or representative family-library evaluation.

Candidate policy is fixed at cosine **0.65**, identity lead **0.15**, and **two
manually confirmed references from distinct original assets**. The weaker of the
two references must meet the threshold. Compare that weaker score against every
competing identity's strongest reference, even a competing identity with only one
reference. Exclude the query original, dismissed/hidden/manual-geometry-only
references, automatic labels, invalid vectors and incompatible providers. No
automatic output becomes a reference. No runtime bypass enables the policy.

For subjects 11–30, enroll exactly photos 1 and 2 and evaluate photos 3–10:
**160 known queries**. Subjects 31–40 remain unenrolled: **100 unknown queries**.
No alternate enrollment is chosen if a photo fails. Missing detections remain
in denominators. Subjects 1–10 are not enrolled or scored in these outcomes.
No threshold, margin, reference choice or pass gate is retuned after the result.

Predeclared narrow gates: at least 360/400 exactly-one-face detections, all 40
fixed enrollments available, at least 80/160 correct automatic known assignments,
zero wrong identities and zero unknown false accepts. Uncertain queries remain
unnamed. Report all outcomes, including misses and unknown processing failures.
The existing native inference, original integrity, no-face and scratch cleanup
checks must also pass. Automatic-naming synthetic tests cover user corrections,
undo, repeated analysis, reanalysis, multiple people, provenance, restart, export
and independent restore. A narrow native pass alone does not approve deployment.

Run `scripts/Test-NativeFaceQuality.ps1 -AutomaticNames` on the established cloud
Windows Server runner. It performs native inference once, then runs both the
existing review screen and this separate automatic policy screen. The JSON
contains aggregate metrics only. Corpus/provider copies are removed afterward.
If the screen fails, keep the policy frozen and disabled and record the result.
The production constant remains disabled until broader held-out diverse-library
evidence and release acceptance explicitly justify changing it. That evaluation
must address pose/lighting/age variation and similar-looking different people,
which this constrained grayscale corpus cannot establish.
