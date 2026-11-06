# Release 0.1.43 development checkpoint

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
Server 2022 feature/regression and pinned-native policy screens are the next gate.
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
