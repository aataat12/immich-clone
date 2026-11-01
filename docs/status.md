# Status — 2026-11-01

Requirements: `../Immich_Clone_Project_Requirements.md`, copied verbatim from the project source.
No production application or installer exists yet.

## Milestones and acceptance

| Milestone | Acceptance | State |
|---|---|---|
| M0 compatibility | Exact PC edition/build/architecture/hardware; legitimate cloud Windows 10 access, licensing, cost and automation established | PC inventory received privately; blocked on cloud entitlement/access and support policy |
| M1 dependencies | Pinned versions, redistribution/support ledger; real media operations and intended model CPU inference under service identity; Windows 10 execution | Not passed |
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

PC inventory and intended external media folder received in conversation; private
hardware/drive details are not committed. ESU status confirmed privately; OS security support remains unresolved. Local Visual
Studio installation is not evidence of a qualifying subscription. No known licensed
cloud Windows 10 access. Recommend read-only external-library semantics for synced
media and application state outside sync roots. User data has not been accessed.

Pinned FFmpeg candidate probe added; execution result pending. This is independent
of application runtime selection and uses synthetic fixtures only.

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

Resolve eligible cloud Windows 10 access. Vet and pin a redistributable media tool
and actual model/runtime, then run bounded representative experiments; do not build
substantial features around unproven dependencies. No costs authorized.
