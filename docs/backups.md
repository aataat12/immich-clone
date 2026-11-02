# Staged backup capacity/progress checkpoint

Accepted bounded 0.1.23 retains the accepted schema-1 ZIP and independent restore path.
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
ENOSPC cleanup/retry and full installer regression passed in
[41398447659](https://github.com/aataat12/immich-clone/actions/runs/41398447659),
source `66a698d066c437219c6e1b23fe8dbd5ed01ad577`, job 125124797979.
Original/import fixture: 96MiB NTFS; backup fixture: separate 192MiB NTFS with a
256MiB hard fill cap. Both archives restore with old state unavailable, retaining
original SHA256, names/tags/import history and crop regeneration. Exact delivered
installer/checksum is authoritative in status.md. Screenshots inspected.

No external destination picker, incremental/deduplicated backup, backup-byte
resume, automatic retention or unattended scheduling. Full-library backup/ZIP64
size/endurance and disconnected-storage acceptance remain separate gates. No
personal paths, capacity reports, IDs, archive hashes or media enter public logs
from a user machine; CI uses disposable synthetic data only.

## Harness corrections before acceptance

Run 41396771601 passed new backup UI/cancel/read-back/download and restart/range
checks, then restoration failed. The range probe reused a PowerShell web session;
PowerShell copies supplied headers into WebSession.Headers and reuses them. The
probe is now isolated; the subsequent full archive must return HTTP 200 before
restore. Run 41397485363 cleared both restores/installer but its 96MiB backup fault
fixture was correctly rejected by the conservative staging reserve. The original
96MiB test remains; a separate 192MiB backup fixture now reaches post-preflight
ENOSPC. Disk fault checks run early. Final acceptance above passed both fixes;
no failed-run installer was published.
Primary header-behavior source inspected:
https://github.com/PowerShell/PowerShell/blob/master/src/Microsoft.PowerShell.Commands.Utility/commands/utility/WebCmdlet/Common/WebRequestPSCmdlet.Common.cs
