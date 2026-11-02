# Current status

Authoritative requirements: `../Immich_Clone_Project_Requirements.md` including the
Server-first amendment. No new charges, parallel agents or publication of personal
machine reports, private paths/media/inventories or raw support logs. Synthetic/
public licensed fixtures only. CI timestamps are UTC.

## Current prototype: 0.1.21 indexed gallery paging

[Source dd59c6b](https://github.com/aataat12/immich-clone/commit/dd59c6b1261d8871b28449ba34f99431ad81fe20),
[application acceptance 41391526713](https://github.com/aataat12/immich-clone/actions/runs/41391526713),
job 125103776422 passed on Windows Server 2022. No dependency, native worker/model,
original layout or authoritative schema change. Prior Home controls remain.

Gallery now queries/renders 60 assets per page with Next/Previous, capture-date,
upload-date and case-insensitive filename ordering, hidden filtering and stable
content-ID ties. Derived ordering/count indexes commit with asset/queue/import
receipts and rebuild from assets on open, including old-writer/restore recovery.
Sorted bulk insertion keeps rebuild cost bounded in the measured fixtures.
Validated cursors bind order/filter/epoch/revision; changed order or restart resets
to page one with a notice. Preview-only status updates retain cursors. Later pages
refresh on user action; Refresh returns to page one. Legacy full-list API remains
for explicit older/test callers; gallery polling no longer uses it.

Actual evidence:
- Linux race/vet plus Windows cursor/filter/order/atomicity/old-writer/rebuild/
  restore guards passed. Index failure cannot commit an asset or false receipt.
- 600 actual JPEG/PNG/BMP/GIF/TIFF originals: interruption/reopen/replay,
  600 native previews, original SHA256 and independent receipt/media restore.
- Edge/LocalService: 121 actual PNG originals with SHA256 receipts across three
  gallery pages; completeness/no duplicates, <=60 cards, forward/back/sort,
  hide/restore, stale-cursor reset and no legacy full-list reads. Desktop/narrow
  screenshots downloaded and inspected. Existing media/AI/import regressions pass.
- Actual hash-verified 0.1.19 installer upgraded to 0.1.21, failed 0.1.22 upgrade
  rolls back. Restricted service, backup/independent restore/reinstall/uninstall,
  names/tags/import history and real 96MiB NTFS import ENOSPC/retry passed.
- [Scale run 41391526782](https://github.com/aataat12/immich-clone/actions/runs/41391526782),
  job 125103776560: 50k metadata-only rows, first page 76,120 Go allocation bytes,
  next page 86,616, page+JSON 188,592; rebuild one sample 827.413ms. Not RSS/peak
  memory, a speed guarantee, or 50k actual media. Timer-zero readings are below
  VM resolution. Earlier full-list baseline reused; details in dependency-evidence.

[Installer artifact](https://github.com/aataat12/immich-clone/actions/runs/41391526713/artifacts/13117688399):
`ImmichClone-0.1.21-prototype-x64-setup.exe`, 102590261 bytes, SHA256
`924ced26664cc542fdcc6fbad3bc0c1cfde3733d261fbd14f41af4149ef31ed7`.
Retrieved ZIP SHA256 `5d942d2c1ad70bf60c968b9383f17954ffc38e132cc624a1f18ea158d8515b51`
and executable verified against included checksums. Browser artifact 13117169530
ZIP verified and gallery screenshots inspected. Unsigned prototype; 7-day artifact
retention. Server results do not establish clean Windows 10 compatibility.

## User-reported Windows 10 checkpoints

0.1.15: all twelve People/tag review-flow checks passed (analysis/crops, individual/
batch naming/clear, suggestions/undo, manual missed-face marking, tags/search,
dismiss/restore). Original-photo comparison passed; no newly reported SHA256 proof.
0.1.17: Bulk import reported working well.
0.1.17-to-0.1.19: both Home controls, desktop/narrow/no overflow, actual version,
photos/names/tags/import-history and upgrade preservation passed; no problems.
These are limited functional reports, not broad recognition/endurance/clean-PC
certification. No raw private reports retained. No further 0.1.19 test or reboot
requested. Optional 0.1.21 checkpoint uses a small test library exceeding 60 items
for paging/sort/hide/restore; no full-library import or dependency setup requested.

## Next milestones and remaining acceptance

1. Capacity/progress and recovery for backups, which currently stage a complete
   ZIP inside state. Larger actual mixed-media endurance, storage disconnect and
   long-running recovery remain open before a full real-library import.
2. Optimize resume only with content verification. Current copy limits: 50,000
   nonempty files/selection, 16MiB manifest, 4GiB/file, 128 sessions, one active
   copy lease. Resume retransfers accepted bytes for SHA256 receipt comparison;
   no metadata-only skip/partial-file continuation. See bulk-import.md.
3. People/tag scalability and broader family/age/profile quality remain separate
   gates; current review still scans the pilot catalog. Native still-image AI
   only; HEIC/video AI, face dismissal and ready reanalysis remain deferred.
4. Hidden Media GUI with local locations and deliberate permanent managed-copy
   deletion/recovery remains queued: hidden-media.md. No deletion implemented.
5. Clean Windows 10 prerequisites, final reboot/startup, production security and
   signing remain open. User defers reboot to the end; setup needs no reboot.
   Public/private repository decision remains deferred, visibility unchanged.

## Preserved baseline and records

Managed originals are immutable SHA256 copies; external sources stay untouched.
Persistent jobs, loopback/code-cookie auth, restricted LocalService, metadata/
orientation, previews, original downloads/export and consistent verified restore
remain accepted within their documented bounds. Preview failure never authorizes
original removal. RAW/HDR/high-bit-depth and broad camera coverage remain open.
Do not repeat old diagnostics or completed small local checks without a relevant
change. Exact dependencies/evidence: dependency-evidence.md; architecture:
decisions.md; AI: ai-feasibility.md; partner tasks: partner-handoff.md. Older status
checkpoints remain in Git history.
