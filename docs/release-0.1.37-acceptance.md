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
