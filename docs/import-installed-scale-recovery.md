# Milestone 4: installed mounted-volume recovery and sustained mixed import

## Scope and acceptance

Continue from the accepted `0.1.38-prototype` executable, SHA256
`14e1f0dd6b5bc709d45235963e137e47ee5740fee13ec3c8c03d4756ddd24038`.
The mounted-volume proof uses that unchanged package. The sustained-load follow-up below introduces a focused application scheduling change; no provider, schema, file-size limit or dependency changes. The existing exact 4 GiB HTTP and installed browser boundary,
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
failed its verification harness because the gallery request omitted the required `visibility=library` parameter and correctly received HTTP 400. Both full 2,048-entry copy sessions had already passed all receipts/counts/bytes checks (35,986 ms first, 33,770 ms second). The harness request now matches the existing production API; no API defaults or input validation were changed. The corrected baseline execution is [41782798531](https://github.com/aataat12/immich-clone/actions/runs/41782798531), source `9a5b108a4c531baafcf5b55aa4a48d1e90400486`, job `126417994811`; FAIL: both full copy sessions passed (40,362ms and37,425ms), then the unchanged ten-minute final preview-drain assertion failed. No failed preview was reported; the aggregate drain deadline failed before final original readback verification. Source review identified the existing service ticker permits at most one preview per second, so a 1,536-preview queue inherently cannot drain within the ten-minute verification bound. The accepted package is not relabelled as a faster build.

### Preview scheduling correction

`cmd/immichclone/main.go` now drains completed preview jobs serially without an artificial inter-item delay. It checks context cancellation and the existing Store copy-first gate before each job. Idle, AI polling and storage-error paths retain a one-second delay; provider time/memory/process limits and single-job concurrency stay unchanged. No automatic AI analysis is introduced. The queue remains authoritative and persisted, with the existing restart recovery.

New tests process ten real distinct PNGs before an intentionally long idle interval, hash-verify originals, exercise a live import lease that keeps every preview queued, release that lease, drain the original jobs and check prompt cancellation. Local race tests, full Go suite and vet passed; targeted worker/import tests also passed on Windows. They test actual Store operations rather than a mock processing callback.

The next focused Windows run builds `0.1.39-prototype-candidate` from the reviewed source, installs the accepted0.1.38 service/providers, stops that service, substitutes only the candidate executable, then executes the same sustained workload. This is explicitly a candidate executable proof, not a new installer or0.1.38 result. It retains the ten-minute preview-completion bound. Candidate execution [41783697663](https://github.com/aataat12/immich-clone/actions/runs/41783697663), source `3191ac612748c61b0561c98aea0bb0f4541bd870`, job `126420857646`: PASS, first candidate attempt on Windows Server 2022 x64, Go1.27.3, actual LocalService and Edge/Playwright1.62.1. All4,096 receipts,1,536 unique managed originals,2,560 duplicate receipts, six MIME counts of256, all completed previews, all original HTTP/source hashes, source aliases, bounded gallery/history pages, staging cleanup, both service restarts and offline installed Verify passed. Each batch copied64,010,893 actual bytes; copy phases took28,191ms and28,977ms. The final preview-drain/page/hash verification took60,053ms. These timings describe this modest-byte synthetic/byte-varied corpus and runner, not user-library throughput or broad natural-media coverage.

The old one-preview-per-second scheduler cannot finish1,536 queued jobs within the same ten-minute verification window even with zero-cost decoding; the candidate removes that artificial scheduling ceiling. The workload and verification bound were retained. Combined installer/upgrade/rollback and independent restoration acceptance remain required before delivery. No new installer or full milestone completion is claimed yet.

## Remaining gates

Record actual results here before updating delivery status. Final target OS,
clean-machine/reboot and extended operation remain milestone 8 gates. Physical
hardware/network-share fault behavior is not established by virtual-volume tests.
The assigned worker completed [milestone5 installed large-byte backup/recovery](installed-backup-recovery-acceptance.md), including two full cycles, actual volume withdrawal and independent hashes/metadata restoration. Its separate ZIP64/count proof is reused. Security Group 2/3 work
and every-frame Full Scan remain paused/deferred. User validation of 0.1.38 is
pending and no additional local test, dependency installation or reboot is requested.
