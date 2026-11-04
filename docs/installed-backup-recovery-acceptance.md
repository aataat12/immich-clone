# Installed backup/recovery scale and interruption acceptance

This is a bounded Milestone 5 installed-service acceptance record. The actual accepted installer passed the focused Windows execution below. It
uses only generated public test bytes on a disposable runner.

## Scope and existing evidence

Reuse the prior [ZIP64 entry-count proof](backup-zip64-entry-count-acceptance.md)
(65,536 originals), [large-volume proof](backup-large-volume-acceptance.md)
(actual >4 GiB archive and independent restore), and
[restore interruption proof](restore-interruption-acceptance.md). Those exercise
the library directly; this run adds the accepted installer, actual LocalService,
Edge session, controller crash, withdrawn state volume, repeated full archive
creation/download, and independent installed-command recovery. It does not
repeat the 65,536-entry workload.

The accepted installer is `ImmichClone-0.1.38-prototype-x64-setup.exe`,
104,194,388 bytes, SHA256
`14e1f0dd6b5bc709d45235963e137e47ee5740fee13ec3c8c03d4756ddd24038`,
from accepted run `41771361544` / source
`86807bff81930a5394c60d695185bbdb7008d81c`.

## Workload and guards

`scripts/Test-InstalledBackupRecovery.ps1` owns an expandable, size-capped,
test-created NTFS VHD and installs the application with its complete state there.
It refuses an occupied service/installation and insufficient host capacity. The
application executable and downloaded archives remain outside the removable
state volume. No real user files or existing disks are selected.

`tests/installed-backup-recovery.cjs` streams one actual 4 GiB original and one
1 MiB original using deterministic AES-CTR test entropy in bounded chunks. It
also imports sixteen distinct, valid JPEGs carrying EXIF orientation, capture
time and camera-make metadata. All eighteen use the real HTTP import protocol
and completed receipts. One photo receives a tag and named manual face; another
is hidden. Valid JPEG previews must finish before fault injection. Entropy
prevents compression from turning the archive-size gate into a small ZIP.

The harness observes at least 64 MiB of actual backup copying before terminating
the owned service process. Cold recovery must expose interrupted status, deny a
partial download, clean partial staging, preserve the complete source catalog,
and support a successful new backup. It then verifies a complete >4 GiB archive
through streaming download, exact SHA256/byte count, range bytes, recovery
instructions, refresh/restart retention, and rejection of discard while a
download is active.

After discarding only the temporary server archive, a second nonzero copy is
interrupted by detaching the test state volume. The withdrawn volume must not
produce a downloadable Ready archive. Cold startup without the volume must
refuse; after reconnect, the interrupted/failed job must remain non-downloadable
and a fresh complete backup must work. Both independent downloads are retained
and checked again after another restart and temporary-archive cleanup.

Finally, the original state volume is detached. The actual installed executable
restores the second downloaded ZIP into a new independent folder and verifies
all original hashes. `TestInstalledBackupRecoveryMetadata` checks all identities,
sizes, source aliases, EXIF/geometry, hidden state, named face, tag, complete
receipt entries/counters and access-code preservation against the pre-fault
baseline. It prints aggregate results only.

## Execution: PASS

- [Focused run 41783177421](https://github.com/aataat12/immich-clone/actions/runs/41783177421)
- Executed source: `bb71cc8fcf4dfb22707e2ad9497bfd6ca7a7bc92`
- Successful job: `126419210212`, Windows Server 2022 x64, Go 1.27.3,
  installed LocalService and actual Edge, Playwright 1.62.1.
- Job: 2026-11-04 06:05:00–06:11:23 UTC. The acceptance step ran
  06:05:44–06:11:20 UTC, including setup and cleanup.

| Gate | Actual result |
| --- | --- |
| Real installed import | 18 originals, 4,296,027,382 physically stored bytes; completed receipts; preparation 42,107 ms |
| Abrupt service termination | Actual copying at least 64 MiB before force-kill; cold interrupted status, partial download refusal, partial staging cleanup, original metadata retained |
| First complete cycle | Archive strictly larger than 4 GiB; create, full read-back, streamed download SHA256/size, protected active download, exact 64 KiB range, recovery guide and refresh retained; 52,481 ms |
| Ready restart and explicit discard | Same retained job/checksum after restart; only temporary server archive removed; independent first download hash/size unchanged |
| Live state-volume withdrawal | Actual DiskPart detach during nonzero copy at 06:08:18; no partial Ready/download; absent-volume startup refused, no listener/library exposed |
| Reconnection | Same catalog reappeared; actual installed CLI verified every original SHA256 before resumed service |
| Second complete cycle | New >4 GiB archive, complete read-back/download/checksum/range/recovery-guide/refresh guards; 42,385 ms |
| Restart, downloaded copies and cleanup | Both independent downloads rehashed with exact byte counts unchanged; second Ready retained after restart; temporary `.zip`/`.part` cleanup preserved originals and downloads |
| Independent restoration | Original state VHD detached at 06:10:43; actual installed CLI restored second downloaded ZIP to a new folder using unchanged default budgets; published 06:11:06 |
| Independent integrity and metadata | Installed CLI verified every restored original SHA256 at 06:11:09; Go oracle passed all 18 identities/sizes/aliases, JPEG EXIF/geometry, hidden state, named manual face, tag, complete receipts and access-code hash at 06:11:15 |

The per-cycle duration includes archive creation, full read-back, streamed full
download, range and instruction checks. The logs assert each archive exceeds
4,294,967,296 bytes and that complete downloaded byte counts equal the service's
archive size; they intentionally do not print archive contents or saved codes.
The restored CLI queues derived previews for regeneration; this probe checks
preserved source metadata/geometry, not newly regenerated preview pixels.

No production change was needed for these M5 gates. Combined candidate-installer
regression remains a release gate when integrating unrelated application changes;
this result establishes the accepted 0.1.38 installed backup implementation.

## Limits

Two complete large-byte backup/download cycles are a bounded endurance probe,
not a multi-day soak or a large heterogeneous personal-library claim. A test
VHD withdrawal exercises mounted NTFS volume unavailability; it does not emulate
power loss, physical controller failure or every storage driver. Reboot and
physical-storage behavior remain distinct. Cloud Windows Server execution does
not establish Windows 10 compatibility. Group 2/3 security work and changes to
restore limits, archive format, dependencies and providers remain outside scope.
