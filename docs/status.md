# Current status

People candidate: reversible false-face dismissal/restoration preserving names,
undo history, tags, analysis and originals. Linux focused race/full tests/vet
passed; Windows Server 2022/Edge acceptance passed (41519696666), including
actual dismissal/reload/restore. See face-dismissal.md. Security worker source reviews are in
security-installer-review.md and security-worker-isolation-review.md; neither
claims new runtime evidence or clears the outstanding security release gates.

Next-release candidate: 64 accepted TCP connection cap and rolling sixty-second
response write-idle bound. Linux real-socket race tests/vet passed; focused Windows
validation passed on Windows Server 2022/Edge (41501827158). See
security-baseline.md S-008. No installer/version changed.

Next-release restore candidate: validate the complete manifest/ZIP integrity and
reject noncanonical or special-file archive entries before extraction. Linux
negative/valid-restore tests and vet passed; Windows Server 2022/Edge validation
passed (41510214737, S-009). Next security review: parent-folder/reparse and ACL
boundaries, aggregate archive/catalog work and native isolation.

Next-release installer candidate replaces root DACLs instead of retaining unknown
explicit grants and preflights reparse components. Disposable Windows acceptance
passed on Windows Server 2022 (41513726020, S-010): ordinary-user access checks,
junction preservation, actual production helper/LocalService upload and restart.
Child explicit ACLs, concurrent path races and staging the new upgrade preparer
remain gates before packaging. No installer/version changed.

## Current cloud-accepted prototype: 0.1.29

[Acceptance 41487369408](https://github.com/aataat12/immich-clone/actions/runs/41487369408),
source f2242c2942caf1cc923a1ea7ad708900a8a3ad5a, job 125421140599 passed Windows
Server 2022. Bundles explicit login/sign-out/current-session revocation, rolling
copy-idle and admission safeguards, legacy export overlap limits, responsive AI
status and the verified static zlib 1.3.2 worker. Model weights/schema unchanged.
Provider directory: opencv-4.12.0-zlib-1.3.2-cb35f0ccb3e3; old immutable provider
remains available for rollback, not selected by the new executable.

Actual restricted-service browser/native/backup/import/paging tests passed, plus
old 0.1.27 inference/names/tags retained without rerun, failed cross-provider
rollback back to 0.1.27/old worker, successful 0.1.29 upgrade, actual new-worker
inference on a distinct original, installed-payload code/private-marker scan,
0.1.30 failed-upgrade rollback, independent restore/reinstall, original hashes
and real NTFS full-disk recovery. Linux tests/vet passed. No dependency install.

Artifact 13161554725 retrieved; installer 104117297 bytes SHA256
646a47acc8ac77751c8343202d12c09eff0a754fd0731f976b0b7e8fc9dcea66
matches manifest. ZIP SHA256
458c02d6b316f8054d068565a5815159e20a197472d3e6161812f1e535416493
verified. Browser artifact 13162742432 ZIP verified; login-narrow, People-narrow
and Build Notes desktop screenshots inspected. Unsigned prototype, seven-day
artifacts. Source remains on work/security-baseline/PR #2; no main merge claimed.

User-reported 0.1.29 local version/data, sign-out/confirmation, wrong-code
rejection, manual unlock, refresh persistence and five-page desktop/CSP checks
passed, with no observed failures. Updated native inference and narrow layout
were not tested; see local-validation.md. No repeat of passed checks requested.
Do not enlarge the library merely to force AI inference: use an already
unprocessed photo if available, otherwise leave new-provider Windows 10 inference
pending for normal use. No full-library import or reboot requested. Broader
security/People milestone, clean Windows 10, long endurance and wider resource/
storage/isolation review remain open. Server success is not certification.

## Working boundaries

Authoritative requirements: ../Immich_Clone_Project_Requirements.md including the
Server-first amendment. No new charges. User authorized two bounded security workers alongside People
feature development on 2026-11-02; release security gates remain. Never publish private
machine details, media/inventories/paths or raw support logs. Public/synthetic
fixtures only. Ask the user before a security mitigation conflicts with the agreed
plan. Repository visibility remains unchanged; decision deferred. CI time is UTC.
Older acceptance checkpoints remain in Git history and dependency-evidence.md.

## User-reported Windows 10 checkpoints

0.1.15: all twelve People/tag review-flow checks passed; original-photo comparison,
not newly reported SHA256 proof. 0.1.17: Bulk import worked well. 0.1.19: Home
navigation, desktop/narrow layout/version and data preservation passed. 0.1.21:
requested gallery checks passed. 0.1.25: backups, Build Notes and 87/87 eligible
photos completed with names/tags/data retained; desktop passed, narrow not repeated
and new-import follow-up not tested. 0.1.27: version/upgrade, five-page browser/CSP,
desktop and focused existing media/organization/import/backup/Build Notes passed;
authentication-flow checks were unavailable because controls were missing.

0.1.29: user-reported version/upgrade and existing photos/names/tags/ready AI,
pagination/import history passed; sign-out confirmation, incorrect-code rejection,
manual re-login and authenticated refresh/navigation passed. All five pages and
AI batch status worked at desktop width without blocking JavaScript/CSP errors.
No failures observed. Updated runtime inference was not tested (no eligible
unprocessed photo); unsupported video remains outside still-photo AI. Narrow
layout was not tested because that viewport was unavailable in the test surface.
Do not repeat passed checks or enlarge the library solely for testing. Native Windows 10 inference can wait for an already
unprocessed photo from normal use. These reports do not certify broad recognition,
clean-PC prerequisites, endurance or reboot startup. Reboot remains at final gate.
No raw private reports retained. See local-validation.md for the recorded result and remaining scope.

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
