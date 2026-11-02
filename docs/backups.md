# Staged backup capacity/progress checkpoint

Candidate 0.1.23 retains the accepted schema-1 ZIP and independent restore path.
No original-layout, dependency or required-secret change. User reports the 0.1.21
local gallery checkpoint passed; no repeat of completed local checks or reboot.

## Behavior

- Backups page preflights application-storage available bytes against catalog
  snapshot + conservative ZIP bound + 64MiB reserve. Manifest/entry limits match
  Restore. Recheck under the same consistency lock; cap archive writes to the
  checked bound. Changing free space can still fail safely. Browser download
  destination needs separate free space and cannot be inspected by the server.
- Persist one service-owned job outside catalog/ZIP snapshots. Copy/read-back/hash
  progress is visible without buffering ZIP bytes in the browser. Closing the
  page keeps a service job running. Cancellation covers lock wait, snapshot,
  copy, verification and hashing. Restart exposes interruption and cleans partials.
- Archive creation holds the existing write gate for consistency. Read-only
  browsing/status/cancel remain usable; uploads/library edits wait. Read-back and
  hashing use the completed private stage after releasing the write gate.
- Verify each archived checksum by reading it back, then compute archive SHA256.
  Publish an exclusive linked download only on successful completion/persistence.
  This does not replace independent restoration tests or certify future bit rot.
- Retain the one ready archive across restart for repeated/range downloads. Block
  a new job until explicit discard. Bind cancel/discard/download to the job ID;
  discard requires confirmation, cannot race an active download, and affects only
  this generated temporary ZIP. Originals and downloaded copies stay untouched.
  Unknown/corrupt job state makes backups unavailable without deleting retained
  completed files or preventing normal library operation.
- Existing GET /api/backup remains for older callers, with the same staged-space
  guard and no partial response. The main UI routes to the progress page.

## Acceptance

Linux race/vet and actual backup/progress/cancel/budget/read-back/truncated-ZIP,
retained-job/restart/discard/authentication and independent restore checks pass.
Windows actual LocalService capacity/progress/cancel/download/restart/range,
explicit discard, independently restored downloaded archive, actual post-preflight
ENOSPC cleanup/retry and full installer regression are pending.

No external destination picker, incremental/deduplicated backup, backup-byte
resume, automatic retention or unattended scheduling. Full-library backup/ZIP64
size/endurance and disconnected-storage acceptance remain separate gates. No
personal paths, capacity reports, IDs, archive hashes or media enter public logs
from a user machine; CI uses disposable synthetic data only.
