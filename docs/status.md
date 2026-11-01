# Current status — 2026-11-01

Authoritative requirements: `../Immich_Clone_Project_Requirements.md` and its
Server-first execution amendment. No new charges or parallel agents. No personal
machine specifications, paths, media or returned logs belong in this repository.
Historical checkpoints remain in Git history; this file supersedes their gates.

## Current milestone: additional image formats 0.1.7

Candidate proof passed on Windows Server 2022 under LocalService:
[41342166588](https://github.com/aataat12/immich-clone/actions/runs/41342166588),
commit `46a8113a3d516e4782f6fa79d95a8516dea07ac3`. Independent System.Drawing
BMP/TIFF/GIF fixtures decoded to verified pixels, PNG roundtripped, truncated
inputs rejected, original hashes preserved and service restart passed. Linux
race tests and Windows CGO-disabled cross-build also passed.

Integration at `16d82bd7d0ac85c69e7b0220e8a7933950596e64` adds BMP, first GIF
frame and first TIFF page previews. 64 MiB encoded-file and 40MP limits apply
to these added codecs. TIFF has a preview-only viewer; original download remains
independent. Existing unsupported files reconsidered once. End-to-end Windows
application/browser/installer acceptance passed in
[41342541106](https://github.com/aataat12/immich-clone/actions/runs/41342541106).

## Implemented scope

Native Go loopback-only, access-code authenticated Windows LocalService; immutable
SHA256 originals; duplicate detection; confirmed browser-folder copying with
relative-source aliases; durable preview jobs; JPEG orientation/capture-date/camera
metadata; timeline grouping/sorting and details; image viewing; browser H264/AAC
MP4 playback; exact original downloads; JSON export; consistent backup and verified
independent restore. Installer upgrades/rollback and uninstall preserve data.
Private bounded diagnostics have browser/offline export and no automatic upload.
No original deletion API or server-side source scan. State stays outside sync.

## Accepted evidence and limits

| Area | Actual evidence |
|---|---|
| Infrastructure diagnostic | [41320566461](https://github.com/aataat12/immich-clone/actions/runs/41320566461), reused without rerun; not product validation |
| Native runtime/catalog | [41325465670](https://github.com/aataat12/immich-clone/actions/runs/41325465670), actual LocalService/database/image operations |
| Recovery/reinstall | [41329698834](https://github.com/aataat12/immich-clone/actions/runs/41329698834), restored to different location with old state unavailable; auth/original hashes verified |
| Diagnostic 0.1.3 | [41334317634](https://github.com/aataat12/immich-clone/actions/runs/41334317634), online/offline logs, permission failures, privacy and installer regressions |
| Metadata 0.1.5 | [41340063567](https://github.com/aataat12/immich-clone/actions/runs/41340063567), all eight transforms, metadata, Edge playback, install/upgrade/rollback/uninstall, independent restore and actual NTFS disk-full recovery |
| Local Windows 10 | User reports 0.1.3 restricted service, upload/hash, native PNG preview/full view, service restart and independent restore passed. User reports two JPEGs in 0.1.5 display expected date/time and orientation; BMP/GIF/TIFF checks in 0.1.7 pass and previous JPEGs remain correct. Limited user-reported evidence; no raw transcript or personal inventory retained |

No setup reboot is required. User explicitly deferred full OS reboot acceptance
to the end; it does not block development. No local action currently needed.
Server execution does not establish clean Windows 10 compatibility.

## Remaining gates and next work

0.1.7 acceptance, screenshot inspection and artifact checksum verification are
complete. Installer SHA256 and run/artifact evidence are in dependency-evidence.md.
User reports 0.1.7 BMP/GIF/TIFF acceptance passed; previous JPEGs remain correct.
Next: investigate video-thumbnail/HEIC providers and their redistribution/native
dependency closure through bounded experiments. HEIC/HEIF, RAW,
HDR/color management and video thumbnails are not supported yet. TIFF variants
and TIFF orientation/metadata remain limited; no blanket TIFF claim. AI stays
optional/unimplemented until actual intended-model inference and licensing pass.

Final acceptance still needs clean desktop prerequisites, OS reboot startup,
physical storage disconnect, endurance/large-library behavior, and security review.
Unsigned prototypes are not production releases or approval for bulk ingestion.
Exact dependency versions/licenses/checksums and prior evidence are in
`dependency-evidence.md`; significant choices are in `decisions.md`.
