# Managed-copy bulk import acceptance

User resumed import development while the private 0.1.15 People/tag validation is
pending. Reuse the verified queue and native providers; do not infer AI acceptance.

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
limits; milestone 4 remains open for larger endurance, gallery paging and capacity.
No large real-library invitation or new local test request.
