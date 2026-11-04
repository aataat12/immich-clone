# 0.1.37 combined acceptance

## First combined attempt: harness setup failure, no publication

[Run 41746893765](https://github.com/aataat12/immich-clone/actions/runs/41746893765), source `5dbac5653e379ffc81f449d4d0f887d53a849639`, job `126302059718`: failed at the newly added installed tests/backup.cjs call. No installer artifact or release was published.

Module verification, Go tests/vet, existing dependency check, full-disk recovery, application service/browser/backup restore, actual checksum-verified 0.1.36 installation and upgrade, initial failed-upgrade rollback, native HEIC/video AI/reanalysis, Hidden Media batch/navigation and source-loss import checks passed before the failure.

The added installed browser call requires IMMICH_BROWSER_OUTPUT. The preceding application step set that variable only in its own PowerShell process, unlike IMMICH_BROWSER_MODULE persisted through GITHUB_ENV. Therefore the installer step's process had no output directory and Node refused path.join with undefined. No new recovery-product failure was established. The installer step now explicitly sets the same synthetic browser-evidence output directory before running Test-Installer.ps1. No application code, limit, timeout or acceptance assertion was changed to bypass the failure. A new combined run is required before publication.
