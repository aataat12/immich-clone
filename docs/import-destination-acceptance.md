# Milestone 4: managed-originals directory outage and recovery

Completed bounded acceptance slice. No production behavior, dependency, limit or installer was changed.

[Windows run 41737710757](https://github.com/aataat12/immich-clone/actions/runs/41737710757), source `9d365f2a35269c0cbf8054ad659de6b309f725a5`, job `126271092301`: PASS, first attempt, Windows Server 2022, Go 1.27.3 windows/amd64, CGO disabled, GOAMD64=v1, actual LocalService and Edge/Playwright 1.62.1. The dedicated worker branch was based on main source `006f6d6cc184154744dcaba0556c7c5571b15235`; this was a built application/service proof, not an installed-package acceptance run.

## Actual operations and results

- The authenticated service accepted the first of three real source files and saved its receipt. Copy-first remained active while the harness renamed the actual `originals` directory to its disposable offline name; catalog and source directories stayed available.
- The browser opened the saved import and resumed with the same physical files. Re-hashing/replaying the earlier accepted file failed while its destination was unavailable. The original AssetID and accepted byte/file counters remained intact, the saved failure was shown, and copying paused with its lease released.
- The test also sent the genuinely pending 8 MiB file through authenticated HTTP while the originals directory remained absent. Publication returned HTTP 500, saved a per-file error and left no accepted AssetID, extra original, replacement originals directory or temporary upload. The earlier catalog item and offline original hash stayed intact.
- The harness reconnected the original directory before restarting the service. The saved paused import and its one earlier receipt survived restart. The browser reselected all three files and resumed the same import. Final state was completed, three accepted receipts, two new originals, one duplicate and the exact manifest byte total. Every source and downloaded managed original matched SHA256, receipt errors cleared, staging was empty and copy-first was released. Offline CLI Verify passed all catalogued originals.

The fast `TestImportOriginalsDirectoryUnavailableRecovery` independently exercises actual directory rename, rejection of both new publication and accepted replay, retained offline bytes/catalog/receipt counters, staging cleanup, reconnect, catalog close/reopen, resume and duplicate accounting. It passed locally in Go 1.27.3, including a race run, and also in the Windows job before the service/browser phases. Browser JavaScript errors were absent.

## Reproduction in disposable cloud CI

The dedicated workflow `.github/workflows/import-destination.yml` provisions pinned Go and Playwright, then runs `scripts/Test-ImportDestinationLoss.ps1`. That script builds and initializes randomly named disposable state, registers its own LocalService, executes the three browser phases in `tests/import-destination-loss.cjs`, verifies originals, and removes only its own service/state. The normal Go regression does not require a special environment flag or large source fixture.

## Scope and remaining limits

This is a real directory-availability outage on NTFS, not physical disk/volume detachment, a network-share outage, loss of the catalog volume or proof of restart while destination storage is still absent. Reconnection explicitly precedes restart. It is not Windows 10, a packaged installer, a 4 GiB browser file-picker test, mixed-media endurance, native AI or large-backup acceptance. Existing separate proofs retain their original scopes.

There were no failing cloud attempts or production fixes in this bounded slice. Security Group 2/3 work and the known restore progress-read:32 issue were not investigated. Every-frame Full Scan remains deferred. These tests do not complete milestone 4 or expand its accepted file-size limit.
