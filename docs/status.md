# Status — 2026-11-01

Requirements: `../Immich_Clone_Project_Requirements.md`, copied verbatim from the project source.
A prototype originals slice is implemented; Windows service/application validation is
in progress. No production release or installer exists yet. Latest user amendment authorizes
Server-first development and final local Windows 10 validation; see requirements.

## Milestones and acceptance

| Milestone | Acceptance | State |
|---|---|---|
| M0 compatibility | Exact PC edition/build/architecture/hardware; legitimate cloud Windows 10 access, licensing, cost and automation established | Server-first path authorized; local Windows 10 validation deferred |
| M1 dependencies | Pinned versions, redistribution/support ledger; real media operations and intended model CPU inference under service identity; Server execution now; final local Windows 10 execution | In progress |
| M2 originals slice | Safe upload/import, hash preservation, duplicates, metadata, thumbnails, browser listing and exact download; persistent jobs and restart recovery | Gated on applicable M1 evidence |
| M3 recovery | Fault injection, concurrent uploads, interrupted jobs, consistent backup and actual restore to independent location | Not started |
| M4 installer | Clean Windows 10 install without SDK, prerequisites, least privilege service, reboot, upgrade/rollback/uninstall preserving media | Not started |

Included: browser photo/video management, upload/import, metadata, derived previews,
download/export, durable jobs, backup/restore. Deferred: optional replaceable AI,
advanced search, albums/sharing until basic reliability. Excluded: mobile apps,
mobile backup and Immich mobile-client compatibility. No full parity claim.

## Evidence reused

Server diagnostic run 41320566461 and job 124889659817 inspected through GitHub
run metadata, step results and logs on 2026-11-01. No rerun. See dependency ledger.
Its historical 'next milestone' paragraph is superseded by M0/M1 above.

## Current work

User-specific system information is excluded from public records. Application state
must remain outside sync roots; external media sources are read-only by policy.
No user media has been accessed.

[Pinned FFmpeg candidate run 41322933829](https://github.com/aataat12/immich-clone/actions/runs/41322933829)
passed at commit `de7903fe25f2c4b96228671ac1cb49160badc7dd` on Server 2022.
Job 124896798777 logs inspected: all nine assertions passed, including actual
JPEG/PNG decode/thumbnail operations, MPEG4-Part2/AAC decoding/frame extraction,
container title metadata, preserved source hashes and malformed input rejection.
Initial run 41322703178 exposed an expected-nonzero-exit harness issue, corrected
before this passing run. No Windows 10/service-account, H264/HEVC, HEIC/RAW/HDR,
real-camera corpus or AI inference validation. Candidate remains unaccepted.

Read-only PC inventory script supplied; no dependency installation requested.
Storage experiment tests synthetic bytes only, independently of media/AI choices.
[Run 41322200662](https://github.com/aataat12/immich-clone/actions/runs/41322200662),
commit `6970003766aa0f74702a645e145a83b60dba200a`, passed. Job 124894603992
logs inspected. Server 2022 Datacenter 10.0.20348, Windows PowerShell
5.1.20348.5756, NTFS. Inventory parsed/executed with required fields present and
private output suppressed. Six storage assertions passed: same-volume publish,
SHA256 preservation, partial-upload isolation, existing-target rejection,
independent-process read, and synthetic original backup/restore after removal.
This does not certify a production store, application-consistent backup, power-loss
durability, Windows 10, media codecs or AI. No dependency experiment accepted yet.

## Next

Execute the pinned Go/database/service probe on Server, then build the originals
vertical slice around passing dependencies. AI remains separately gated; do not build
substantial features around unproven dependencies. No costs authorized.

## Runtime milestone

[Run 41325465670](https://github.com/aataat12/immich-clone/actions/runs/41325465670),
commit `45d44e5`, passed on Server 2022. Actual bbolt rollback/reopen and JPEG/PNG
encoding/decoding tests passed. Native service startup, HTTP, database writes and
restart persistence passed as LocalService with Go 1.27.3, CGO disabled, GOAMD64=v1.
Job 124904357122 logs inspected. These dependencies are accepted for the basic
Server prototype; local Windows 10, endurance and release acceptance remain open.

Implemented pending application CI: loopback authentication, streaming upload,
content-addressed originals, duplicate detection, explicit preview job status,
JPEG/PNG thumbnails, browser listing/download, JSON export, consistent ZIP backup
and checksum-verified independent restore. No external folder import or AI yet.
Linux cloud unit/integration tests with race detection and vet passed.
