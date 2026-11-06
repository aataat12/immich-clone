# Release 0.1.43 development checkpoint

Latest checkpoint: [catalog compatibility accepted](release-0.1.43-catalog-compatibility.md).
Version2 now protects automatic provenance; genuine published0.1.42 refusal and
full Windows feature/recovery tests passed. The [independent quality decision](automatic-naming-quality-0.1.43.md) now supports
enabling the unchanged prototype policy for actual pipeline/release validation.
The original checkpoint below is historical.

Accepted scope: R43-01 clearer date provenance, R43-02 gallery search, and R43-03
automatic known-person identification. The wider research recommendations remain
0.1.44 planning. Baseline is published 0.1.42 source
`f37f61f2fe79deb92eeb3ff614ea6b245180a391`. Two bounded workers contributed date
fixtures/API review and automatic-name mechanisms; the coordinator integrated
search and acceptance. Unrelated development was not changed.

## Implemented candidate

- Gallery cards use Date taken for a usable capture date, otherwise Date imported.
  Camera wall-clock timestamps are retained. Information says capture time zones
  are not read, rather than claiming the original supplied none; import time is UTC.
- Main-gallery search matches assigned face names (including every person in a
  multi-person photo), filenames, custom tags and active object tags. Suggestions
  and dismissed names/tags are excluded. Eligible catalog metadata is cached by
  committed transaction; only the requested gallery page is decoded/rendered.
  Hidden media is searched only in its separate view. Queries, counts and clear
  controls are visible; sort/visibility are preserved in the URL. Query-bound
  cursors refuse stale metadata. Viewer close retains results, page and scroll.
  Search failures clear stale cards and show an error distinct from no matches.
- Automatic naming has a separate frozen candidate policy, trusted manually
  confirmed references from two source originals, competitor-margin rejection,
  per-face provenance and correction/undo. Photo, sampled-video and additive
  reanalysis publication hooks are transactional. Automatically named faces never
  become trusted references. Existing explicit/cleared/undone labels are protected.

## Current evidence

Local tooling: bundled Node executes actual gallery date helpers; gallery/People
inline scripts and browser fixtures pass syntax checks. PowerShell feature fixture
parses. Go is not installed locally; no dependency setup was requested. Windows
Server 2022 feature/regression and pinned-native policy screens passed below.
The development workflow builds only a disposable test executable, not an installer.
Only synthetic/public fixtures and aggregate native metrics may be retained.

## Automatic naming is incomplete and disabled

`automaticNamesValidated` remains false; there is no runtime bypass switch.
The proposed .65 similarity / .15 competitor lead / two-source policy is a candidate,
not a validated confidence probability. A separate frozen native evaluator checks
160 known queries and100 unknown queries after two references per known person.
ORL has already been studied for suggestion evaluation; even a narrow pass is not
independent general-library accuracy evidence. Do not tune and certify on this
same evaluation set. See the frozen protocol in experiments/face-quality.

Before enablement: independent representative quality evidence, explicit acceptance
criteria, older-writer compatibility for assignment provenance, and full combined
upgrade/rollback/provider/browser/independent recovery acceptance. An older writer
can discard additive provenance when editing newer labels; automatic assignment
must stay disabled until that path cannot promote an automatic error to a manual
reference. Existing installer catalog rollback alone does not prove arbitrary
old-executable/new-catalog reuse safe. Tests of synthetic embeddings prove logic
and durability, not real recognition accuracy.

## Release gates

No 0.1.43 installer, tag, public download or final Windows 10 certification exists
at this checkpoint. Published0.1.42 remains the current prototype. Original
preservation, no new charges, unsigned-installer limits and open SEC-05/SEC-08
remain unchanged. Finish all three accepted features and their evidence before
packaging; then validate genuine0.1.42 upgrade, injected rollback, pinned providers,
full browser journeys and independent restore/reinstall before publication.

## Executed checkpoint evidence

Feature acceptance: source `112c0ec159f877f90786fb417424ec10c6c0a0dc`,
[Windows Server 2022 run42158716082](https://github.com/aataat12/immich-clone/actions/runs/42158716082),
job127698964272. Pinned Go1.27.3 module verification, all internal/command tests,
vet and actual date helpers passed. Real LocalService/Edge acceptance passed:
assigned Alice/Bob and multi-person matches, filename/custom-tag queries,65-item
filtered paging, viewer return/page/scroll, clear/empty/error handling, hidden
scope and reload, desktop/narrow layout and all65 original SHA256 values.
The unit/API suite also covers suggestions/dismissals, stale/query-bound cursors,
and distant matches across10,003 catalog records. Synthetic screenshot artifact
13476190940, ZIP SHA256
`6366ec32f6c3010cf375d344c8842448efb69038f5b7959d56ed2f7d86bc4535`.
An initial run failed only a new test's incorrect reanalysis method call; corrected
source passed the full gate. Runner gofmt normalization was copied back locally;
it changes formatting only, with no product behavior change.

Separate native screen: source `cc34b4d9eef39aa0a092893e1f14019edab5ebe5`,
[run42158326967](https://github.com/aataat12/immich-clone/actions/runs/42158326967),
job127697698876 passed. Native policy/mechanism code is unchanged in the corrected
feature source. The overall initial workflow is red because of the separate,
subsequently corrected feature fixture; this native job itself is successful.
All400 corpus images produced exactly one face;40 fixed reference enrollments
were available. Of160 known queries,147 correct,0 wrong,13 unresolved; of100
unknown queries,0 false accepts and100 rejected. Existing review/native original
integrity, no-face and scratch cleanup checks also passed. This establishes only
the declared narrow screen; it does not justify general-library accuracy or
turning on automatic naming. Production remains disabled.

This checkpoint completes the first development/validation slice requested by
the user, not the0.1.43 release. Next: address old-writer provenance retention,
obtain independent representative quality evidence without tuning to this test,
and validate enabled hooks/recovery before the combined packaging gate.
