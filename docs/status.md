# Current status — 2026-11-01

Authoritative requirements: `../Immich_Clone_Project_Requirements.md` and its
Server-first execution amendment. No new charges or parallel agents. No personal
machine specifications, paths, media or returned logs belong in this repository.
Historical checkpoints remain in Git history; this file supersedes their gates.

## Current milestone: optional media provider 0.1.9

Completed integration at `f2ecfd8be6683db923fc5e3115c9a0844c8c9d04`:
[application run 41348548366](https://github.com/aataat12/immich-clone/actions/runs/41348548366)
passed Server 2022 LocalService, Edge and full installer/recovery acceptance.
Adds optional checksum-pinned CPU HEIC/video previews, complete HEIC grids,
retry UI and private diagnostic stages. Windows creation-time Job Objects bound
worker commitment to 512 MiB and one process, with deadline/output bounds and
parent-death cleanup. The exact shared launcher passed service controls too.
Source, notices, compiler versions and build recipe accompany the installed worker.

Latest installer: `ImmichClone-0.1.9-prototype-x64-setup.exe`, artifact `13100442001`.
Retrieved installer SHA256 verified:
`594b29a07a67848f332d35438af7c025ab6c0a2c14c3e068f2dde31f692b11d0`.
Synthetic desktop/narrow screenshots inspected. Missing-provider recovery, installed
provider execution from paths with spaces, independent HEIC restore/regeneration,
failed-upgrade rollback, preserved originals and actual NTFS full-disk recovery all
passed. Focused Linux race/vet and reachable Windows Go vulnerability checks passed.
New provider remains unverified locally on Windows 10; earlier local checks apply
to their respective versions. No dependencies or setup reboot required.

## Prior accepted image formats 0.1.7

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
MP4 playback; optional HEIC/H264/HEVC/MOV previews; retryable bounded worker; exact original downloads; JSON export; consistent backup and verified
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

0.1.9 cloud acceptance and checksum verification are complete. A small copied
HEIC and short MP4/MOV local check is useful if available; no bulk ingestion or
reboot requested. Next main work: measure large-library browsing/job behavior and
address demonstrated bottlenecks. A recommended independent partner task is a
backup/restore failure-path review; see partner-handoff.md. No partner access,
invitation or external task dispatch has been performed.

HDR/high bit depth, RAW, bursts, advanced HEIC/TIFF metadata, broad camera coverage
and optional AI remain unsupported/unverified. AI requires actual intended-model
CPU inference/licensing before features. Final reboot remains deferred.

Final acceptance still needs clean desktop prerequisites, OS reboot startup,
physical storage disconnect, endurance/large-library behavior, and security review.
Unsigned prototypes are not production releases or approval for bulk ingestion.
Exact dependency versions/licenses/checksums and prior evidence are in
`dependency-evidence.md`; significant choices are in `decisions.md`.
