# Status — 2026-11-01

Authoritative requirements: `../Immich_Clone_Project_Requirements.md`, including
the later user-authorized Server-first amendment. No paid cloud client OS needed.
No personal hardware, exact OS build, drive inventory, source paths or enrollment
state belongs in this public repository, logs or artifacts.

## Current product

A working native Go prototype provides loopback access-code login, streaming
upload, confirmed browser-folder copying, immutable SHA256 originals, duplicate
detection, persisted JPEG/PNG preview jobs, browsing, download, JSON export and
consistent backup/verified independent restoration. No original deletion API.
External folders are never server-scanned or changed. App state is outside sync
folders. Browser-folder copying does not retain relative directory structure yet.

Full-image viewing and actual H264/AAC MP4 playback have passed in Edge on Server.
No AI features are implemented.
Format limits are explicit in `prototype-notes.txt` and the UI.

## Milestones

| Milestone | State / acceptance remaining |
|---|---|
| M0 target/testing path | Server-first path authorized; final local Windows 10 validation pending |
| M1 basic dependencies | Go 1.27.3, bbolt 1.4.3, x/sys 0.49.0 executed under LocalService; core accepted for Server prototype. Optional media/AI remain separate gates |
| M2 originals slice | Server service and Edge browser tests passed, including confirmed folder copy and independent restore |
| M3 recovery | Abrupt exit, duplicate concurrency, corrupt backup, missing originals path, denied permission and actual NTFS full-disk recovery passed. Physical disconnect, extended endurance and OS reboot pending |
| M4 installer | Server fresh install, actual version upgrade, injected failure rollback to prior healthy version and data-preserving uninstall passed. Clean Windows 10 and reboot validation pending |

## Executed evidence

- [Infrastructure diagnostic 41320566461](https://github.com/aataat12/immich-clone/actions/runs/41320566461): reused without rerun; diagnostic-only limits retained.
- [Synthetic file primitives 41322200662](https://github.com/aataat12/immich-clone/actions/runs/41322200662): six assertions passed on Server NTFS.
- [Pinned FFmpeg experiment 41322933829](https://github.com/aataat12/immich-clone/actions/runs/41322933829): actual synthetic media operations passed; no app redistribution.
- [Go/database/LocalService 41325465670](https://github.com/aataat12/immich-clone/actions/runs/41325465670): rollback/reopen, JPEG/PNG encode/decode, service HTTP and database persistence across restart.
- [Originals application 41326182923](https://github.com/aataat12/immich-clone/actions/runs/41326182923), commit `1ee7fa2`: full native service, upload/download/hash, jobs, independent restoration and denied permissions passed.
- [Installer/browser/full disk 41327909119](https://github.com/aataat12/immich-clone/actions/runs/41327909119), commit `b0fb334`, job `124911644966`: all assertions passed. Actual Edge sign-in/upload/confirmed folder-copy/preview/backup download and narrow layout passed. Installer versions 0.1.0→0.1.1 and rejected 0.1.2 injection restored healthy 0.1.1; uninstall preserved catalog/code/originals. Actual 96MiB disposable NTFS volume filled, failed upload not catalogued, earlier original intact, and subsequent upload succeeded after space freed. Logs inspected.

Passing artifact 13092226402: installer ZIP retrieved and verified SHA256
`ffbfce3134316024f27bd53195e3139eb7ca089833ca4ae3543c7b363b4de81f`.
Installer 0.1.1 SHA256 at that commit:
`5310e7883040d89076893c2d5d0e874b6f23418380da5cfa26b5e5f9268c3e0e`.
Desktop/narrow screenshots inspected; no overflow or private data. Artifact expires
2026-11-08; this is historical evidence, not the latest media-enabled package.

Earlier installer failures exposed PowerShell service command discovery, strict
CIM parameter widths and false-success exit propagation; fixed and accepted only
after end-to-end lifecycle passed. No failed build is a validated release.

## Next concrete work

Verify the local validation helper under Windows PowerShell, then hand off the
packaged prototype for local Windows 10 installation and reboot validation. No bulk
real-library ingestion, production release, Windows 10 reliability, AI inference
or clean-machine compatibility claim yet.

## Media-enabled acceptance

[Run 41328317537](https://github.com/aataat12/immich-clone/actions/runs/41328317537),
commit `19be51c`, job `124912848751`: all checks passed, logs inspected. Pinned
FFmpeg actually generated an H264/AAC MP4; application upload recognized it and
Edge decoded/played it (video dimensions and playback time checked). Full-image
view and HTTP byte-range retrieval also passed. Installer lifecycle and actual
full-disk recovery passed again for the changed build. Generated video thumbnails
remain unsupported; no FFmpeg binary is bundled.

The local Windows PowerShell validation helper is being exercised under the
installed package before the Windows 10 handoff. It uses an empty test library,
two synthetic originals, service restart, PNG decode and independent restoration;
no hardware inventory, personal-source reads or remote report upload.

## Local helper acceptance

[Run 41328990649](https://github.com/aataat12/immich-clone/actions/runs/41328990649),
commit `683fd8d`, job `124914893938`: every step passed, including the local
acceptance helper executed in built-in Windows PowerShell 5.1 against the actual
installed service. Native PNG decode, restart, original retrieval, independent
restore and checksum verification passed; no machine inventory is emitted.
Final service-level recovery test now adds reinstalling against restored state
while the original directory is unavailable, before the local Windows 10 gate.
