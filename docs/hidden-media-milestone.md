# Milestone 3: Hidden Media management

Current candidate: hidden-item discovery, managed location, reversible restore and explicitly confirmed permanent managed-copy deletion. Not in a released installer.

The gallery has a Hidden Media entry point using the existing bounded hidden gallery. Each hidden card offers Managed file location with a read-only absolute path and Copy path (manual Ctrl+C fallback). This identifies the content-hash-named managed original, not an external source file. Full source paths are not known; relative source aliases remain catalog metadata. The path is returned only to an authenticated local session for a currently hidden item, is not included in diagnostics, and is served with no-store caching. The page does not launch Explorer or a shell, rename a file or alter filesystem permissions. Existing Download original saves with its original filename.

Restore to library uses the existing durable visibility update. No original, face, tag, analysis result, proposal or import history is deleted. Hidden originals remain included in backups.

## Acceptance for this slice

- HTTP authentication/Host guard coverage includes the new read-only location route; visible assets refuse location inspection through this feature.
- Browser acceptance uses an existing generated public video with completed sampled observations, a named face, tags and an applied reanalysis record. Hide, open/copy location, inspect a narrow dialog, restart while hidden, restore, compare full review metadata and original SHA256.
- Existing independent backup/restore, media recovery and reanalysis tests remain in the focused Windows workflow.

## Remaining milestone work

The deletion implementation below is a candidate. Remaining acceptance gates:

1. Focused Windows LocalService/browser deletion, permission-failure, crash/retry and real pre-delete archive restoration: PASS in run 41697638495 (details below).
2. Deletion-specific full-volume refusal/retry and transaction rollback checks: PASS in run 41702306375, within the scope documented below.
3. Combined actual old-installer upgrade/failed-upgrade rollback, including pending intents, followed by installed deletion/restore/uninstall checks.
4. Final target Windows acceptance before marking milestone 3 delivered.

No security Group 2/3 investigation or mitigation, every-frame Full Scan, new dependency or user-PC action is included. Windows Server testing does not certify Windows 10 compatibility.

## Verified candidate evidence

[Windows run 41671321055](https://github.com/aataat12/immich-clone/actions/runs/41671321055), source `9ea771a8274f500b344b042440a5abc8f6e324b1`, job `126040140853`: PASS on Windows Server 2022, Go 1.27.3, LocalService, Edge/Playwright 1.62.1. Both hidden-media browser phases passed, including actual clipboard readback, 390px layout, restart while hidden, restore to library, exact original SHA256 and full review-record equality. HTTP/guard and hidden-state independent backup/restore unit tests passed. Existing HEIC/video inference, failure/crash recovery, reanalysis controls and independent restored-frame regeneration passed in the same run. Local full web suite and vet also passed. No installer/version publication or Windows 10 certification; milestone 3 is not complete.


## Permanent deletion candidate

Hidden cards offer Permanently delete managed copy. The dialog names the item, describes the managed-copy/review-data scope and historical-backup/source exclusions, and requires typing DELETE. The API requires the content ID and original import timestamp, preventing an old dialog from deleting a new incarnation with identical bytes. Cancel before submission has no effect; after acceptance there is no undo/cancel-cleanup operation.

The operation nonblockingly acquires analysis, import, catalog-copy and media-decoder coordination gates and rejects an active import lease. It verifies the original before committing intent. A single durable catalog transaction records a bounded version-1 cleanup intent, removes the asset and its face/label/tag/analysis/proposal/queue records, updates gallery/batch indexes, and marks historical import receipts deleted. It then removes only exact hash-derived preview, face-crop and original filenames (no recursive deletion), clearing intent only after cleanup succeeds. External source aliases are metadata, never deletion paths. Existing backups are untouched.

Failed/interrupted cleanup stays visible in Hidden Media with Retry cleanup. Startup makes a bounded three-second recovery attempt; it does not prevent unrelated library access if permissions still refuse cleanup. While any cleanup is pending, new backups are refused; a pending ID cannot be reimported. Historical receipts cannot silently recreate the item; a new explicit import is allowed after completion and starts without old corrections. Manual-face/crop and reanalysis Apply publication recheck identity to reject stale work.

Older executables ignore the additive intent bucket. If an older writer reintroduces an asset while cleanup is pending, the new app refuses further deletion, shows a conflict, and offers Keep current copy to clear intent without touching that copy. Actual old-installer upgrade/rollback validation remains a combined-release gate; tests simulate older-writer catalog publication and check that recovery preserves the reappeared original.

Local tests cover original-integrity refusal before intent, visible/wrong/stale confirmation, cancellation before acceptance, active inference/copy/import/media/actual backup refusal, queued batch removal, deleted import receipts, explicit reimport, partial-cleanup/restart/missing-file recovery, older-writer conflict, retained external source and pre-/post-deletion independent backup restores with corrections. Focused Windows acceptance adds actual LocalService native photo analysis, dialog cancel/type/busy checks, narrow layout, deleted-cache refusal, NTFS delete denial, forced service death with a committed pending intent, restart/retry cleanup and independent restoration of the real pre-delete archive. No power-loss durability or secure erasure claim.


### Permanent deletion Windows run history

- Run [41680203977](https://github.com/aataat12/immich-clone/actions/runs/41680203977), source `a2bffd661350294ff2bb726c5ffde5549d17be34`: Windows units/guards and actual normal deletion, cancel, active-import refusal, narrow confirmation and explicit reimport passed. The test-only advanced icacls delete-child right lacked parentheses, so the permission-failure scenario did not begin.
- Run [41681346823](https://github.com/aataat12/immich-clone/actions/runs/41681346823), source `41e846c326007a626d043a72a27d4f9d5755c192`: the corrected ACL setup succeeded, but deletion returned 409 before the expected accepted-pending state; crash/retry/restore acceptance was not established. The test now captures the response body, verifies that the denial preserves original read/hash access, and allows at most five explicit retries only for the designed transient deletion-busy response. Other 409 causes still fail immediately. Recovery checks run before media regressions to shorten diagnosis.

- Run [41683468062](https://github.com/aataat12/immich-clone/actions/runs/41683468062), source `e7d7b7ff8474875dd73ad672a549af14c3e94f0f`, deletion job `126082909041`: Windows units/guards and actual normal deletion passed. The new precondition exposed HTTP 403 reading the original after the test ACL setup. That setup does not preserve read/hash access, so it does not isolate the intended delete-only cleanup failure. Do not claim permission-denial cleanup, forced-death recovery or pre-delete archive restoration acceptance from this run; execution stopped before those checks. No further fix/retry was started after the user's pause instruction. Resume point: correct the disposable denial fixture while preserving read/hash access, then rerun the held recovery checks. Candidate remains unreleased and milestone 3 incomplete.

Work paused at the user's request after collecting the running checks. All source/test changes are saved; installed 0.1.33 is unchanged. The independent import-boundary job in the same workflow passed; that result does not make the overall deletion workflow pass.


### Resumed fault-fixture correction

The user unpaused development. The test now records the basic `D` deny mask, removes that temporary deny, applies the advanced `(DE)` right and asserts the exact numeric file DELETE mask (65536) and parent DELETE_CHILD mask (64). Existing read/hash preconditions and pending-cleanup/crash/retry/independent-restore assertions remain unchanged. This changes only disposable fault injection, not production ACLs or deletion checks. Windows execution remains the acceptance gate. The unrelated, already-passed 4 GiB job is now opt-in through workflow dispatch or `[import-boundary]` in the trigger commit message.


- Run [41693853717](https://github.com/aataat12/immich-clone/actions/runs/41693853717), source `84389ff59f587098cc0dbbf900f741bc932d4dd0`, job `126119082605`: exact-mask probe recorded basic `D` as 1114112 (DELETE plus SYNCHRONIZE), while advanced `(DE)` and `(DC)` were exactly 65536 and 64. Actual original read/hash access passed; accepted-pending deletion and pending-state recovery after forced service death passed. The workflow then reached its 20-minute limit waiting on Playwright `response.text` in the cleanup-retry test. Do not claim retry completion or restored-archive acceptance from this cancelled run. The test now reads diagnostic bodies only on error for Retry, checks the actual pending list/file disappearance after the UI action, bounds diagnostic body reads at five seconds, and bounds each deletion browser phase at 180 seconds. Production deletion code and acceptance assertions are unchanged.


## Current outcome after unpause

[Run 41697638495](https://github.com/aataat12/immich-clone/actions/runs/41697638495), source `d0aaf0b4233fcdb18156b8a110a3b6ba1a367d23`, job `126132262444`: PASS on Windows Server 2022, Go 1.27.3, actual LocalService and Edge/Playwright 1.62.1. The exact `(DE)`/`(DC)` masks preserved original read/hash access; deletion returned accepted-pending with managed bytes retained, and the pending intent survived forced service termination/restart. After restoring the disposable permissions, UI Retry returned HTTP 200, removed the pending record and original, and re-enabled backup planning. Independent restoration of the actual pre-deletion archive recovered the hidden original, named face, custom tag and paused proposal; crop regeneration/integrity checks passed.

The same run passed normal confirmation/cancel/busy/narrow/reimport checks, focused Windows units/guards, HEIC/video inference and failure/crash recovery, reanalysis controls, hidden location/clipboard/restart restoration and final independent restored-frame regeneration. No production ACL, deletion implementation, provider, limit or acceptance assertion was weakened to get this result. The prior timeout was addressed by bounding the browser observation and avoiding an unnecessary body read on a successful Retry; actual state/file/backup assertions established completion. The earlier failed/cancelled run history remains above.

The unrelated 4 GiB job was intentionally skipped and reuses its successful Windows proof in run 41683468062. Remaining gates 2–4 above remain open: deletion-specific disk-full/transaction-failure execution, actual old-installer upgrade/rollback and installed deletion/recovery, then target Windows acceptance. No installer/version was published; use installed 0.1.33. Work is unpaused, with this focused recovery checkpoint complete; milestone 3 is not delivered.

## Full-volume and transaction-failure checkpoint

[Run 41702306375](https://github.com/aataat12/immich-clone/actions/runs/41702306375), source `43ad2bb00e7198e8b9123c7dc02af3678344c3e1`, job `126148456024`: PASS on Windows Server 2022, Go 1.27.3. The bounded fixture uses an isolated 96 MiB NTFS volume, 4,096 synthetic manual face records with names, and catalog padding to leave only 69,632 bytes of reusable catalog reserve before exhausting the volume. The application now refuses a new deletion below a 16 MiB free-space floor before beginning its durable transaction; HTTP returns 507 with an instruction to free storage and retry. The test checks that no journal appears, the hidden asset and review records remain unchanged, original bytes remain identical, the catalog stays readable, and deletion succeeds after removing the disposable filler. Existing actual upload/import and backup full-disk recovery also passed on separate bounded volumes.

The same run passed `TestDeletionTransactionRollback`: damaged disposable batch bookkeeping causes failure after the deletion transaction has staged its journal and review-record removals. Rollback retains the hidden asset, review data and original, publishes no intent, and permits successful deletion after repairing the fixture. This is a logical transaction failure, not a claim of power-loss or failed-sector simulation. Local library/web suites and vet also passed.

Earlier attempts are retained as evidence:

- [41700458714](https://github.com/aataat12/immich-clone/actions/runs/41700458714), source `052a63a4d6dea0eadcd3a5517497a34904b83790`: fixture failed because Windows' existing catalog mapping reserve still allowed deletion to succeed on the full volume. This did not prove the intended rejected-write path.
- [41700965342](https://github.com/aataat12/immich-clone/actions/runs/41700965342), source `f067a85449997a6bebc230cbc7ebe88140212047`: fixture retained a read transaction while requesting a mapping expansion, blocking the write. That fixture-only read lock was removed; individual disk tests now have a 120-second timeout.
- [41701392733](https://github.com/aataat12/immich-clone/actions/runs/41701392733), source `e6af7da1f6969119754a1d5358dde6ad375b2fee`: actual disk exhaustion rejected mapping expansion and left the open database handle unmapped. The preservation/readability assertion failed. This is the reason for the new pre-transaction low-space refusal.
- [41701950792](https://github.com/aataat12/immich-clone/actions/runs/41701950792), source `a39b2cac61d1be92035b73833aa7b3948ad9862a`: full-volume refusal, unchanged catalog/original/review, retry, and existing upload/backup recovery passed before adding the separate transaction-rollback case.

The free-space floor is a preflight check, not an exclusive reservation against unrelated processes consuming space, and does not establish recovery from every possible catalog growth or physical I/O failure. No database dependency was changed. These tests do not establish Windows 10 compatibility or installed-package acceptance. Gates 3 and 4 remain open: actual previous-installer upgrade/failed-upgrade rollback with pending intents and installed recovery, then target Windows validation. No new installer was published; installed 0.1.33 remains the user baseline. Group 2/3 security work and Full Scan remain paused/deferred.
