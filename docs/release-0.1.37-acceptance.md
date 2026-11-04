# 0.1.37 combined acceptance

## First combined attempt: harness setup failure, no publication

[Run 41746893765](https://github.com/aataat12/immich-clone/actions/runs/41746893765), source `5dbac5653e379ffc81f449d4d0f887d53a849639`, job `126302059718`: failed at the newly added installed tests/backup.cjs call. No installer artifact or release was published.

Module verification, Go tests/vet, existing dependency check, full-disk recovery, application service/browser/backup restore, actual checksum-verified 0.1.36 installation and upgrade, initial failed-upgrade rollback, native HEIC/video AI/reanalysis, Hidden Media batch/navigation and source-loss import checks passed before the failure.

The added installed browser call requires IMMICH_BROWSER_OUTPUT. The preceding application step set that variable only in its own PowerShell process, unlike IMMICH_BROWSER_MODULE persisted through GITHUB_ENV. Therefore the installer step's process had no output directory and Node refused path.join with undefined. No new recovery-product failure was established. The installer step now explicitly sets the same synthetic browser-evidence output directory before running Test-Installer.ps1. No application code, limit, timeout or acceptance assertion was changed to bypass the failure. A new combined run is required before publication.

## Corrected combined attempt: actual independent restore refusal, release held

[Run 41748196056](https://github.com/aataat12/immich-clone/actions/runs/41748196056), source `3b747066de13ed5bf070adacf37dd433216df254`, job `126306404063`: FAILED. Installer retention and publication were skipped. Published baseline remains 0.1.36; 0.1.37 must not be presented as accepted or available.

The output-directory correction worked. Actual installed LocalService/Edge recovery instructions downloaded successfully, matched the Ready archive's SHA256 and byte size, omitted the access code and opened restore help. Full archive/range downloads, desktop/narrow layout and explicit discard preserving downloaded copies and originals passed.

Actual checksum-verified 0.1.36 upgrade, the initial rollback, native HEIC/video AI/reanalysis, Hidden Media batch/navigation, source-loss recovery and the later injected 0.1.38 rollback passed. The actual previous writer retained pending deletion state and refused reimport; current explicit deletion cleanup Retry passed.

The next operation, restoring the actual pre-deletion archive independently using the installed executable, returned:

> Restore worker failed or exceeded its memory/process/output budget; inspect the archive and selected -restore-memory-bytes budget before retrying. Existing data was not replaced.

The existing assertion `installed executable restores the actual pre-deletion archive independently` failed. Remaining independent-restoration and restored-reinstall acceptance did not complete. The generic CLI category does not identify which worker/resource/status/progress path failed. No claim is made that this is progress-read:32, the same cause as an earlier intermittent failure, archive corruption, a new resource regression, or a recovery-guidance defect.

No new automatic retry, timeout/budget increase, relaxed assertion or production mitigation was added. No failed installer is being offered. The failure is recorded without copying user media or private diagnostics; this was disposable synthetic/public CI state. Root-cause work may overlap the explicitly deferred restore issue, so clarification is needed before reopening that investigation. Group 2/3 security work remains paused. User can remain on 0.1.36 without repeating local tests.

## Authorized narrow restore investigation: progress-file sharing defect

Scope was reopened only to resolve this release blocker, not to resume the broader security backlog. Review found that progress reads used Go os.Open on Windows, which omits FILE_SHARE_DELETE. A progress-file name can be visible with a live DELETE-access handle during publication; opening a reader without delete sharing conflicts with that handle even when the publisher permits reads.

Deterministic Windows baseline [41751340469](https://github.com/aataat12/immich-clone/actions/runs/41751340469), source `10e171898899963804b650ef3fd922de2cb60669`, job `126316879241`: EXPECTED REGRESSION FAILURE. With an actual DELETE-access/shared-read/write/delete handle held on the published progress file, both baseline os.Open and the unmodified production reader returned sharing violation 32. The production error was `progress-read:32`. The handle remained live; no resource-budget overload was injected.

The narrow fix changes only the Windows progress-file open to GENERIC_READ with FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE. Sharing delete permits the existing handle; it grants the reader no write/delete access, does not change ACLs, and adds no retry, limit increase or validation bypass. Non-Windows reading remains os.Open. CLI errors now include a bounded allowlisted restore-stage/numeric code or unavailable, preserving the worker-failure category without raw paths/content.

Focused corrected Windows run [41751727364](https://github.com/aataat12/immich-clone/actions/runs/41751727364), source `5925a1e25d4a51352e50a95ae17a4d0ca3f2de5c`, job `126318150201`: PASS first corrected execution, 5.337 seconds. The same delete-access handle remained open: os.Open still returned 32, corrected production read and catalog-growth check succeeded, wrong nonce and insufficient catalog budget still failed. Existing independent default/explicit restore, aggregate limits, corrupt/cyclic catalog cleanup, cancellation, hard memory/timeout/output and confirmed-reaping tests passed. Local corresponding restore tests/vet passed; no redundant large-byte or entry-count run was performed.

Evidence limit: the earlier combined CLI error did not capture its detailed stage. The deterministic sharing defect is established and matches the historically observed progress-read:32 category; it does not prove every prior generic restore refusal had this cause. Combined installed acceptance remains required before publication. No claim is made of physical-storage, controller-crash or Windows 10 acceptance.
