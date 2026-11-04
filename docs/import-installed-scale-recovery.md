# Milestone 4: installed mounted-volume recovery and sustained mixed import

## Scope and acceptance

Continue from the accepted `0.1.38-prototype` executable, SHA256
`14e1f0dd6b5bc709d45235963e137e47ee5740fee13ec3c8c03d4756ddd24038`.
No new application build, provider, schema, file-size limit or dependency is introduced
by these checks. The existing exact 4 GiB HTTP and installed browser boundary,
selected-source disappearance, mixed-media service-crash/independent restore and
real NTFS ENOSPC proofs are reused, not relabelled as new executions.

The new focused workflows execute the actual installed LocalService and Edge on
Windows Server 2022. They do not establish Windows 10 compatibility, hardware
power-loss resilience, network-share behavior, a diverse natural-media corpus,
50,000-file ingestion or days of endurance. The unchanged public limits remain
50,000 entries per selection, 4 GiB per file and full accepted-byte rehashing on
resume. A larger-file support path is not introduced.

### Mounted-volume outage gate

`scripts/Test-ImportVolume.ps1` creates only its own random disposable 8 GiB
expandable NTFS VHD, asserts its label/size and unused drive letter, and installs
the accepted package with state on that volume and program/source files elsewhere.
The browser selects three physical files: an earlier small original, a valid MP4
with a 16 MiB free box to make partial transfer observable, and a duplicate.

After observing an accepted first receipt and at least 128 KiB of actual partial
upload staging, diskpart detaches the entire mounted state volume while the
service and transfer are live. The browser must report interruption, not success.
Cold startup while the volume is absent must fail without a replacement library
or listener. After reattaching that same volume, offline SHA256 verification,
startup staging cleanup, saved paused receipts, browser reselect/resume, exact
byte/new/duplicate counters and full source/original hashes must pass. The resumed
MP4 preview must actually finish before asserting quiescent staging.

This proves a real mounted-volume availability loss including catalog and staging;
it is stronger than the earlier directory rename but does not emulate physical
hardware removal or sudden power loss. Recovery is explicitly cold restart after
reconnection, not automatic hot recovery while the catalog volume disappears.

First run [41782181291](https://github.com/aataat12/immich-clone/actions/runs/41782181291),
source `efcfe4d59c7230f2f31b3c848e43d66b39b32eb0`, job `126416026301`,
failed the final harness assertion: a legitimate `preview-*.part` was in progress
after successful resume. Actual detach, absent-volume startup refusal, reconnect,
offline original verification, saved receipt/resume/counters and all original
readbacks had passed. The correction waits for the actual resumed video preview
to finish before demanding empty staging; it does not suppress failures, extend
native limits or change production behavior. Corrected run [41782544189](https://github.com/aataat12/immich-clone/actions/runs/41782544189), source `f0c1bc2963395a3ca1ab9a68a8e63b7e5b32ff7b`, job `126417198309`, passed. Both offline SHA256 verifications and the resumed MP4 preview passed. No production change was required.

### Sustained mixed-copy gate

`tests/import-endurance.cjs` creates 2,048 actual files, 1,536 distinct hashes and
512 deliberate duplicates, then executes two complete browser copy sessions with
a service restart between sessions and another restart before final verification.
The first session must record 1,536 new originals and 512 duplicates; the second
must record zero new originals and 2,048 duplicates. All 4,096 receipts, exact
accepted bytes, bounded 250-entry history pages, source aliases and released
copy-first processing must survive.

The distinct originals include 256 each of PNG, JPEG, TIFF, HEIC, MP4 and MOV.
PNG pixel content varies. JPEGs use three already hash-pinned public reduced camera
samples and a synthetic 12-megapixel fixture, with unique valid COM markers.
Other formats reuse existing accepted synthetic content with unique trailing/free
boxes. This deliberately measures real ingest/processing workload, not natural
visual diversity. The workflow generates the existing independent image fixtures
with already pinned CI-only Pillow tooling; no interpreter is installed for users.

Acceptance requires all 1,536 previews to finish after restart, all managed
originals to download with the expected SHA256, all source hashes/aliases to
remain intact, stable bounded 60-item gallery traversal with no missing/duplicate
assets, narrow-page no-overflow and installed offline Verify. Copy and verification
times are runner observations, not throughput promises. Workflow and each phase
are time bounded; no unbounded retry or success-on-timeout is allowed.

Execution [41782439700](https://github.com/aataat12/immich-clone/actions/runs/41782439700),
source `a74d9236059410e837d0fda12828157e1e0db25c`, job `126416862620`,
failed its verification harness because the gallery request omitted the required `visibility=library` parameter and correctly received HTTP 400. Both full 2,048-entry copy sessions had already passed all receipts/counts/bytes checks (35,986 ms first, 33,770 ms second). The harness request now matches the existing production API; no API defaults or input validation were changed. Corrected execution is pending. No milestone completion or new release is claimed yet.

## Remaining gates

Record actual results here before updating delivery status. Final target OS,
clean-machine/reboot and extended operation remain milestone 8 gates. Physical
hardware/network-share fault behavior is not established by virtual-volume tests.
Milestone 5 installed large-byte backup/restore is assigned independently; its
ZIP64/count and restore proofs remain separate evidence. Security Group 2/3 work
and every-frame Full Scan remain paused/deferred. User validation of 0.1.38 is
pending and no additional local test, dependency installation or reboot is requested.
