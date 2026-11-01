# Current status — 2026-11-01

Authoritative requirements: `../Immich_Clone_Project_Requirements.md` and its
Server-first execution amendment. No new charges or parallel agents. No personal
machine specifications, paths, media or returned logs belong in this repository.
Historical checkpoints remain in Git history; this file supersedes their gates.

## Confirmed workflow: managed copies and bulk import

The user explicitly chose application-owned copies after discussing the storage
tradeoff. Upload/Choose folder copies bytes into immutable managed storage and
leaves sources untouched. This supersedes the interim external-library priority;
no external-library implementation was introduced and no originals are removed.
Keep personal source paths and inventory out of public records and CI.

Next: dedicated bulk-import page with copy progress, interruption recovery and
import-first scheduling. Commit original bytes/catalog jobs before reporting a
file successful; defer derived processing while an active bulk transfer runs.
Browser selection cannot promise unattended source access after the page closes;
recovery must state when reselecting sources is required. See `bulk-import.md`.
First finish the independently useful queue index and Windows acceptance below.
Current delivered build remains 0.1.11 until acceptance of a replacement.

## Current work: catalog/worker scale

User now reports MOV files are showing. Close the request for private diagnostics
unless the symptom recurs; root cause was not independently established. This is
user-reported visibility evidence only, not a checksum/codec/corpus validation.
MP4 preview/opening previously passed locally; HEIC remains unexercised.

Bounded 1k/10k/50k metadata-only measurement found linear idle polling overhead:
[Windows baseline 41355763079](https://github.com/aataat12/immich-clone/actions/runs/41355763079),
about 88 MB Go allocations and 131 ms per 50k-row idle poll. Linux baseline was
similar. Implemented a derived transactional pending-job index; focused Linux
race/vet and real PNG job checks pass. Windows comparison and full 0.1.13 installer
acceptance pending. This measures catalog/worker behavior, not bulk ingestion.
API/JSON/browser paging remains a separate limitation.
No user input, local setup or reboot is needed now.

## Latest installer: version display and reversible visibility 0.1.11

User requested running-version identification and removal from the main grid.
Implemented actual executable version at top-right, reversible hide/unhide with
Hidden items view, retained originals/jobs/export/backup, and complete-grid
refresh replacement with stale-response rejection. Acceptance passed:
[41352572001](https://github.com/aataat12/immich-clone/actions/runs/41352572001),
commit `944ee3197cd56d27d52d780f0ecd612c1c2dfca2`.
Installer artifact `13102570178` retrieved and checksum verified:
`1884a464c414ac8b788deab6fe9d5dac8997e89aa3ed6c6e10b1363dea5d6b85`.
Server 2022/Edge tested actual header, hide/restore and out-of-order response
protection; Windows storage tests verified hidden-state persistence through jobs,
restart/duplicate upload/backup restoration, and protected visibility endpoints.
Installer lifecycle, provider recovery and actual NTFS disk-full regression passed.
Desktop/narrow screenshots inspected. No reboot or prerequisite installation.
Local hide/restore acceptance has not been explicitly reported. User says MOVs
are now visible; refresh hardening is not an independently confirmed root cause.

## Prior milestone: optional media provider 0.1.9

Completed integration at `f2ecfd8be6683db923fc5e3115c9a0844c8c9d04`:
[application run 41348548366](https://github.com/aataat12/immich-clone/actions/runs/41348548366)
passed Server 2022 LocalService, Edge and full installer/recovery acceptance.
Adds optional checksum-pinned CPU HEIC/video previews, complete HEIC grids,
retry UI and private diagnostic stages. Windows creation-time Job Objects bound
worker commitment to 512 MiB and one process, with deadline/output bounds and
parent-death cleanup. The exact shared launcher passed service controls too.
Source, notices, compiler versions and build recipe accompany the installed worker.

Prior installer: `ImmichClone-0.1.9-prototype-x64-setup.exe`, artifact `13100442001`.
Retrieved installer SHA256 verified:
`594b29a07a67848f332d35438af7c025ab6c0a2c14c3e068f2dde31f692b11d0`.
Synthetic desktop/narrow screenshots inspected. Missing-provider recovery, installed
provider execution from paths with spaces, independent HEIC restore/regeneration,
failed-upgrade rollback, preserved originals and actual NTFS full-disk recovery all
passed. Focused Linux race/vet and reachable Windows Go vulnerability checks passed.
User reports local MP4 thumbnail/opening succeeded in 0.1.9; HEIC was not
exercised and MOV has the unresolved issue above. Earlier checks apply to their
respective versions. No dependencies or setup reboot required.

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
Running-version header and reversible hidden-items view retain originals/jobs
and backup/export records.
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

0.1.11 cloud acceptance and checksum verification are complete. Reload the page
after upgrade, confirm the version and test reversible hide/restore on a test
item. MOV visibility now works according to the user; request diagnostics only if it recurs.
No bulk ingestion or reboot requested. Next main work: measure large-library browsing/job behavior and
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
