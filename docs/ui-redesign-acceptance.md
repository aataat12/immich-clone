# Existing-function UI redesign: Server/browser checkpoint

The five embedded pages now use the reviewed Library, Hidden Media, People & tags,
Bulk import, Backups and Build Notes navigation. Existing controls, IDs, requests,
authentication and hash-based CSP remain in place. New navigation passes search
queries to the existing People interface and opens/focuses the existing task
panels. Curation remains unavailable; Cinema, durable review decisions/counts and
library-wide server filters are deferred. This is an unpublished candidate.

## Exact validation

Source `d1a47efe66cdec69aa340bbf3adda897280611f5` on
`work/ui-existing-behavior` passed Windows Server 2022
[run 41958781493](https://github.com/aataat12/immich-clone/actions/runs/41958781493),
job `127019911895`. The dedicated `ui-redesign.yml` gate used pinned Go 1.27.3,
Playwright 1.62.1 and the runner's actual Edge with the existing pinned media/AI
artifacts and hash-checked models. No new product dependency/provider was added.

The gate passed web/cmd tests and vet, actual LocalService hidden-navigation and
People-index fixtures, then the full application/browser suite. Coverage included:

- Five page shells at 1360 and 390 pixels, search/query handoff, focus and task
  panels, Hidden Media navigation, no browser errors or horizontal page overflow.
- Authentication/sign-out/revoked-cookie/wrong-code/relogin; original-file hashes;
  BMP/TIFF/GIF/HEIC/HEVC/video metadata, previews and actual H264/AAC playback.
- Actual native model bulk inference, face name/confirm/undo, tag dismissal and
  restore, manual marking and private export; separate face/photo paging/search.
- Import disconnect, close/reselect/resume and receipt hashes; cancellation
  retention; provider loss/recovery; paged gallery bounded to 60 DOM assets,
  hidden restore/stale cursors and no legacy full-list gallery reads.
- Backup creation/readback/hash, cancellation/capacity, range download after
  restart and discard preservation; independent legacy/managed restore and verify
  including names, tags and crops with the original state unavailable.
- Denied-storage service startup reporting and offline diagnostic privacy.

Service fixtures attach the tests to fresh isolated real LocalService instances.
The accepted elevated/direct-service entrypoint guard is preserved. The default-off
`Test-Application.ps1 -UIReview` avoids repeating its old elevated standalone
hidden-navigation probe only after `Test-UIRedesign.ps1` has exercised the same
test attached to a real service; the People fixture likewise uses a fresh service.

The first run, 41957993727, passed the new navigation and existing browser flows
but failed late because Windows PowerShell 5 rejects the existing Range header
assignment. The accepted rerun changed the workflow shell to `pwsh`, as expected
by the existing application script; product/assertion behavior was unchanged.

## Visual evidence and final wording

The successful run retained only 23 synthetic browser PNGs in artifact
`13384194348`; its ZIP SHA256 is
`c62b677602531daf33fed4a1477340f3fa65f923f783dba8286cb0dabf288e15`.
Desktop and narrow screenshots were visually inspected. No private media, backup
archive, support log or real-machine inventory is included.

After the passing run, visible text alone was shortened: unavailable Curation now
says "Not available yet", and Library task descriptions omit implementation
language. A parser comparison against the passing revision verified identical
markup, attributes/IDs, scripts and styles across all five pages; `git diff
--check` passed. Those final words were not in the full browser-run screenshots.
The final combined candidate must retain this checkpoint and its test fixtures.

## Limits and next integration

This accepts the existing-function UI on the disposable Server/browser fixture.
It does not close SEC-03, validate installer concurrency, certify Windows 10,
clean-machine prerequisites/reboot/endurance, or deliver the deferred UI features.
The combined-validation discussion owns merging this branch with accepted full
installer/metadata concurrency evidence and running the genuine prior-release,
provider and recovery gate. Published 0.1.39 is unchanged; no PC action is needed.
