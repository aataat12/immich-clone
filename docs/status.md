# Status — 2026-11-01

Requirements: `../Immich_Clone_Project_Requirements.md`, copied verbatim from the project source.
No production application or installer exists yet.

## Milestones and acceptance

| Milestone | Acceptance | State |
|---|---|---|
| M0 compatibility | Exact PC edition/build/architecture/hardware; legitimate cloud Windows 10 access, licensing, cost and automation established | Blocked on PC details and cloud entitlement/access |
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

Read-only PC inventory script supplied; no dependency installation requested.
Storage experiment tests synthetic bytes only, independently of media/AI choices.
Its result must be recorded after execution; it does not certify a production store.

## Next

Resolve eligible cloud Windows 10 access. Vet and pin a redistributable media tool
and actual model/runtime, then run bounded representative experiments; do not build
substantial features around unproven dependencies. No costs authorized.
