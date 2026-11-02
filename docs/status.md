# Current status

Authoritative requirements: `../Immich_Clone_Project_Requirements.md` including the
Server-first amendment. No new charges, parallel agents or publication of personal
machine reports, private paths/media/inventories or raw support logs. Synthetic/
public licensed fixtures only. CI timestamps are UTC.

## Current prototype: 0.1.23 backup capacity/progress

[Source 66a698d](https://github.com/aataat12/immich-clone/commit/66a698d066c437219c6e1b23fe8dbd5ed01ad577),
[acceptance run 41398447659](https://github.com/aataat12/immich-clone/actions/runs/41398447659),
job 125124797979 passed on Windows Server 2022. No new dependency, native worker/
model, original layout or backup schema. ZIP stays compatible with existing Restore.

Backups page now provides conservative same-volume free-space preflight, persisted
service-job status, progress/cancel, complete checksum read-back, archive SHA256,
retained repeated/range download and explicit temporary-copy discard. A new job
cannot overwrite the retained ready archive. Originals and downloaded copies are
unaffected by discard. Jobs survive page closure; service shutdown/restart marks
active work interrupted and removes incomplete candidates. Unknown/corrupt job
state disables backups without deleting completed files or disabling the library.
Archive creation serializes library writes; browsing/status/cancel stay available.
Read-back/hash happen after releasing the write gate. See backups.md.

Actual acceptance:
- Linux race/vet and Windows actual snapshot/ZIP/read-back/restore, cancellation,
  budget/truncation, authentication/origin, unknown-state and retained-file guards.
- Edge/LocalService: cancellation while a partial upload holds the library write
  gate; status remains responsive. Create/read-back/hash/download, retained range
  requests before and after service restart, Home navigation, explicit discard,
  original/downloaded-copy preservation and diagnostic privacy passed.
- Both the legacy complete download and the browser-downloaded managed archive
  were restored independently to new locations. With old state unavailable,
  original SHA256, names/tags/import receipts and regenerated face crops passed.
- Real NTFS ENOSPC: original/import test retains its 96MiB volume; backup gets a
  separate 192MiB fixture to fit its conservative reserve. Space disappears after
  preflight, copy fails without publishing a partial, preflight rejects the full
  volume, freeing space allows retry and full archive checksum read-back.
- Actual accepted 0.1.21 installer upgrade to 0.1.23, injected failed 0.1.24 upgrade
  rollback, restricted service, reinstall/uninstall/restore and existing 600-image,
  121-image browser paging, AI/media/import regressions passed.

[Installer artifact](https://github.com/aataat12/immich-clone/actions/runs/41398447659/artifacts/13120556502):
`ImmichClone-0.1.23-prototype-x64-setup.exe`, 102593297 bytes, SHA256
`ca713b421b963c0ef645fd8479ce5999c32b5a70e04abd0110f9266779722828`.
Retrieved ZIP SHA256 `d6982bcc3459c64123638aaad43a544c31409945fbbd566345aedc6e174b6c3b`
and executable verified against included checksums. Browser artifact 13120586550
ZIP verified and backup desktop/narrow screenshots inspected. Unsigned prototype,
7-day artifact retention. Server evidence does not certify clean Windows 10.

Two acceptance-harness failures were resolved before publication: PowerShell's
reused Range header contaminated a later full download (now isolated and HTTP 200
asserted), and the backup fixture was initially too small to pass its own reserve.
No failed-run installer was published. Details in backups.md/dependency-evidence.md.

## User-reported Windows 10 checkpoints

0.1.15: all twelve People/tag review-flow checks passed; original-photo comparison,
not newly reported SHA256 proof. 0.1.17: Bulk import worked well. 0.1.17-to-0.1.19:
both Home controls, desktop/narrow layout, actual version and preservation of
photos/names/tags/import history passed. 0.1.21: all requested gallery checks passed,
no observed problems. These are limited functional reports, not broad recognition,
endurance or clean-PC certification. No raw private reports retained. No repeat of
completed checks or reboot requested. Small 0.1.23 backup-page validation is next;
keep the ZIP private because it contains originals and the access code.

## Preserved gallery/import/AI baseline

0.1.21 indexed gallery renders/requests <=60 assets with capture/upload/case-folded
name sorting, visibility filters, Next/Previous and validated stale-cursor recovery.
Indexes/counts commit with assets/import receipts and rebuild after open/restore/
older writers. Legacy full-list API remains for explicit older/test callers only.
[41391526713](https://github.com/aataat12/immich-clone/actions/runs/41391526713)
passed 600 actual mixed native-image originals/previews/recovery/independent restore
and 121 actual browser PNGs across three pages with no full-list reads.
[Scale 41391526782](https://github.com/aataat12/immich-clone/actions/runs/41391526782):
50k metadata-only rows, page+JSON 188592 Go allocation bytes, one rebuild 827.413ms.
Not RSS/peak memory, a speed guarantee or a 50k-media ingestion result. Prior
native worker/model/licensing checks are reused; no source rebuild of providers.

## Next milestones and remaining limits

Current candidate (0.1.25): bulk local AI analysis, including the requested Build Notes
page beside People & tags, top-right Home navigation and maintained Status /
Released In columns. Full remaining milestone acceptance: [roadmap.md](roadmap.md).
The controls and page are implemented; Windows Server acceptance is pending. Do not
publish the candidate as an accepted installer until the workflow passes.

1. Large-container/ZIP64 size and throughput, longer actual mixed-media endurance,
   and storage-disconnect recovery remain open before full real-library import.
   Web backups still stage a complete archive plus catalog/reserve in application
   storage; browser destination needs additional space. No external destination
   picker, incremental backup, automatic retention/scheduling or backup-byte resume.
2. Import still limits selection to 50000 nonempty files, 16MiB manifest, 4GiB/file,
   128 sessions and one active copy lease. Resume retransfers accepted bytes for
   SHA256 receipts; no metadata-only skip or partial-file continuation. See bulk-import.md.
3. People/tag scalability and wider family/age/profile quality remain separate
   gates. Native still-image AI only; HEIC/video AI, face dismissal and ready
   reanalysis remain deferred. See ai-feasibility.md.
4. Hidden Media local-location GUI and deliberate permanent managed-copy deletion
   with recovery remain queued: hidden-media.md. No deletion implemented.
5. Clean Windows 10 prerequisites, final reboot/startup, security/signing remain
   open. User defers reboot to the end; setup needs none. Repository visibility
   decision remains deferred and unchanged.

Managed originals remain immutable SHA256 copies; external sources stay untouched.
Persistent jobs, loopback/code-cookie auth, restricted LocalService, metadata/
orientation, previews, downloads/export and consistent restore retain documented
bounds. RAW/HDR/high-bit-depth and broad camera coverage remain open. Do not repeat
old diagnostics or completed small local checks without a relevant change. Exact
dependencies/evidence: dependency-evidence.md; choices: decisions.md; local checks:
local-validation.md; partner tasks: partner-handoff.md. Older status is in Git history.
