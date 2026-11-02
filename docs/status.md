# Current status

Authoritative requirements: `../Immich_Clone_Project_Requirements.md` including the
Server-first execution amendment. No new charges, parallel agents or publication
of personal machine reports, source inventories, private paths/media or raw logs.
Use synthetic/public licensed fixtures only. CI timestamps are UTC.

## Current prototype: 0.1.17 bulk managed-copy checkpoint

User resumed postponed import work while privately validating 0.1.15 People/tags.
[Source 386cbb2](https://github.com/aataat12/immich-clone/commit/386cbb244bdf243cf908e84fc28f67e788f1551b),
[acceptance run 41378462423](https://github.com/aataat12/immich-clone/actions/runs/41378462423),
job 125063798577 passed on Windows Server 2022.

Implemented dedicated Bulk import navigation/page; persisted bounded manifest and
per-file SHA256 receipts; durable accepted byte/file/new-original/duplicate counts;
separate provisional transfer-byte progress; pause/cancel retaining originals;
browser-close/reselect/resume; import-first scheduling of previews/AI. A renewable
60-second runtime lease expires safely and clears on restart; persisted sessions
resume only on explicit request. Asset/queue/receipt/counters commit together.

Actual acceptance:
- Linux race/vet and Windows storage/HTTP authorization/manifest guards, receipt
  rollback, changed-content rejection, interrupted reads, lease expiry/cancellation.
- 120 actual JPEG/PNG/BMP/GIF/TIFF originals: interruption/reopen/replay, 120 decoded
  previews drained, original checksums and receipt history independently restored.
- Edge desktop/narrow: actual partial-body socket disconnect, no false receipt,
  page close/reselect/resume, duplicate accounting, cancellation retention,
  original/source hashes, no gallery polling, private diagnostic behavior.
- Service restarts preserve completed/cancelled receipts; independent restore with
  old state unavailable preserves receipts, People names/tags and regenerated crops.
- Actual hash-verified 0.1.15 installer upgraded to 0.1.17; injected failed 0.1.18
  upgrade rolls back; fresh restored install/uninstall and bundled native AI/media
  regressions passed. No worker/model rebuild or dependency change.
- Real disposable 96MiB NTFS full disk: import fails without an accepted receipt,
  old originals remain valid, retry after freeing space commits exactly once.

[Installer artifact](https://github.com/aataat12/immich-clone/actions/runs/41378462423/artifacts/13112034249):
`ImmichClone-0.1.17-prototype-x64-setup.exe`, 102581661 bytes, SHA256
`1db93bc321919c51abe0205e10075d3c6ab09fbbd93d51b0d0caa460cb95c6c7`.
Retrieved ZIP SHA256 `a71c3977d0fb74b846c0f2f0aef3bfaec22ef11260c1dedf4c2a6575d4d94aff`;
verified against included checksums. Browser artifact 13112303230 downloaded,
ZIP verified and import screenshots visually inspected. Artifact retention: 7 days.
Installer unsigned; Server testing does not establish Windows 10 compatibility.
First run 41377830420 passed; screenshot review found a stale copying line after
cancel. Final source/run above fixes that and adds partial-socket/import-ENOSPC checks.

Limits: 50,000 nonempty files/selection, 16MiB manifest, 4GiB/file, one active copy
session, 128 saved sessions, receipt results paged at 250. Resume retransfers and
hashes accepted files against receipts; no metadata-only skip or partial-file
continuation. Large-library endurance, gallery paging and large-backup capacity
remain open. Do not invite a full real-library import. See `bulk-import.md`.

## Private 0.1.15 AI checkpoint remains pending

Keep the user's requested small People/tag validation independent of import work;
no new local action or dependency installation requested. 0.1.17 retains the same
AI provider and review behavior. 0.1.15 remains available from
[41374756786](https://github.com/aataat12/immich-clone/actions/runs/41374756786),
artifact 13111240610, executable SHA256
`f99e4d68ae45aaf067886a59fa49e38c9fdea17d92b08276510c995e36e51623`.
That Server run passed actual local CPU inference, naming/confirmed suggestion/
undo/manual marking, tags/dismissal, missing-model recovery and independent restore.
Broader cross-photo family/age/profile quality remains unaccepted. Never auto-name
or merge people using a pair threshold. See `ai-feasibility.md` and `local-validation.md`.

## Next milestones and remaining acceptance

1. Import: bounded gallery/API paging and larger actual mixed-media endurance;
   continue interruption/recovery/backup checks without using private inventories.
   Optimize accepted-file resume only with content verification. Review capacity
   and progress for backups that currently stage a complete archive in state.
2. People/tags: incorporate private target-PC feedback; wider recognition quality,
   review improvements and model replacement stay separate gates. Native still
   image AI only; HEIC/video AI and ready-result reanalysis remain deferred.
3. Hidden Media GUI, actual managed locations, restore and explicit permanent
   deletion with durable recovery remain queued: `hidden-media.md`. No new delete
   implementation or user-media deletion performed.
4. Clean Windows 10 installer prerequisites, final reboot/startup, physical storage
   disconnect, extended endurance and production security/signing remain open.
   User explicitly deferred reboot acceptance to the end; no setup reboot needed.
   Repository public/private decision remains deferred and visibility unchanged.

## Preserved baseline evidence

Existing managed copies remain immutable SHA256 originals with persistent jobs,
loopback/code-cookie authentication, restricted LocalService, metadata/orientation,
previews, browsing/download, consistent ZIP backup and verified independent restore.
BMP/GIF/TIFF first frame/page and optional HEIC/video previews have bounded support;
RAW/HDR/high-bit-depth and broad camera coverage remain unaccepted. Preview failure
never authorizes removal of an original. User previously reported original/PNG/
JPEG orientation/date, BMP/GIF/TIFF, MP4 and corrected MOV behavior passed locally;
retain only these generic findings, never raw reports or personal inventory.

The original diagnostic is verified in `dependency-evidence.md`; do not rerun it.
Older detailed status checkpoints remain in Git history. Exact provider versions,
licenses, hashes, support limits and replacement policy: `dependency-evidence.md`.
Significant choices: `decisions.md`. Partner tasks: `partner-handoff.md`.
