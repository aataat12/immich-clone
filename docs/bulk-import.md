# Managed-copy bulk import acceptance

User reported the small 0.1.15 People/tag and 0.1.17 bulk-import functional
checkpoints passed. Reuse verified work; broad accuracy/endurance remain open.

Primary storage remains application-owned immutable copies. Sources are never
moved, rewritten or deleted. Identical SHA256 content reuses a managed original;
relative source aliases are retained. Previews are disposable derived data.

## Small milestones

1. Queue foundation: transactional pending-job index rebuilt from authoritative
   asset states on open; interrupted work/retry/restore/downgrade recovery retain
   jobs. Compare bounded metadata-only Windows measurements and execute a real
   PNG job, then full service/installer/independent restore regressions.
2. Import-first scheduling: authenticated, bounded transfer session suspends new
   previews while copying. Expiry/service restart cannot strand paused processing.
   Cancellation retains committed originals and queued jobs; byte acceptance is
   distinct from preview completion. Test interruption/concurrency/recovery.
3. Dedicated bulk page: deliberate copy confirmation, files/folder selection,
   byte/file progress, duplicates/failures and cancel/retry. Avoid gallery polling
   and rendering during transfer. Closing the browser requires source reselect;
   reconcile durable accepted files without falsely claiming an unreceived file.
4. Scale acceptance: actual synthetic media ingestion (not metadata-row counts),
   interrupted transfers/restart/disk-full, representative mixed media, bounded
   gallery paging, endurance, independent backup/restore and original SHA256.
   Only then invite a large real-library import. Keep synthetic inputs public.

Browser upload currently limits each file to 4 GiB. That is an explicit limitation
until a tested larger-file path replaces it. Backup currently stages a complete
archive inside application state; large-library backup needs capacity/progress
review. Neither limit is cleared by queue benchmarks or available source space.

No new dependencies, paid resources or local setup are needed for this plan.
Final OS reboot validation remains deferred by the user.

## Accepted bounded 0.1.17 checkpoint

Dedicated copy page with a persisted bounded manifest and per-file SHA256 receipt.
Original asset/preview-queue/receipt/counters commit together. Accepted bytes are
separate from in-flight XHR byte progress. Duplicate aliases reuse originals. One
active import holds a renewable 60-second in-memory processing lease; startup
marks previously running sessions paused. Expiry releases processing, and only
explicit resume may reacquire it. Existing work may finish; no new preview or AI
job starts under the lease. Cancel keeps accepted originals and receipt history.

Reselect/resume validates the manifest and retransfers all selected bytes, checking
accepted entries against their saved SHA256. This deliberately avoids claiming
that matching path/size/timestamp proves unchanged content. Lost acknowledgements
are reconciled without double counting. Partial-file continuation and avoiding
accepted-byte retransfers remain later work. Limits: 50,000 entries/selection,
16MiB HTTP manifest, 4GiB/file, no empty files, 128 saved sessions. Page receipt
results are bounded to 250; transfer does not poll/render the gallery.

Acceptance: Linux race/vet plus actual 120 mixed native-media copies, interruption
and reopen, replay/hash conflict, cancellation, duplicate accounting, preview drain
and independent original/receipt restore passed. Windows full service/Edge actual
partial-body socket disconnect, browser close/reselect/resume, real hash-verified
0.1.15 installer upgrade, rollback, receipt restore and import-specific actual NTFS
full-disk failure/retry passed in
[41378462423](https://github.com/aataat12/immich-clone/actions/runs/41378462423),
source `386cbb244bdf243cf908e84fc28f67e788f1551b`. Installer checksum and artifact
are authoritative in status.md. Desktop/narrow screenshots inspected after fixing
a stale copying line on cancel. Milestones 1–3 are accepted within the stated
limits; milestone 4 remains open for larger endurance and capacity. Gallery paging
is cleared by the 0.1.21 checkpoint below.
No large real-library invitation or new local test request.


## Bounded gallery acceptance criteria

Reuse the measured full-list bottleneck in dependency-evidence.md rather than
rerunning the baseline. The gallery must request/render bounded pages with capture
date, upload date and filename ordering plus hidden/visible filtering. A derived
transactional ordering index must rebuild from authoritative assets on open;
import, preview metadata changes, hide/restore and rollback/restore must keep it
consistent. Avoid a full List+sort behind a superficially paginated response.

Acceptance: bounded page/JSON allocations at metadata scale, deterministic ties
and validated cursors, no silently skipped/duplicated records across stable pages,
stale-page recovery during catalog changes, browser navigation and preserved media
operations. Metadata-only measurements remain separate from larger actual mixed-
media ingestion/endurance and source/original SHA256 checks. No new dependency or
large private-library test needed to develop this next checkpoint.


Accepted 0.1.21 implements the indexed gallery checkpoint: 60 items/page, forward/
back controls, capture/upload/case-folded filename ordering, hidden filtering,
stale-order reset and rebuild after open/restore/older writers. Sorted bulk index
construction avoids costly random inserts in unspilled nodes. Linux race/vet,
600 actual mixed-format originals/previews with independent restore, and bounded
page measurements at 1k/10k/50k metadata-only rows passed on Linux and Windows
Server 2022. Application run 41391526713 at source
`dd59c6b1261d8871b28449ba34f99431ad81fe20` also passed 121 actual browser PNG
originals across three pages, no full-list reads, all controls/stale-cursor recovery,
actual 0.1.19 upgrade/rollback/restore and real import ENOSPC. Scale run 41391526782
records metadata measurements separately. Exact installer/hash is in status.md.
Small local paging checks may use an existing test library exceeding 60 files.
Full-library acceptance remains open for backup capacity and longer endurance.
