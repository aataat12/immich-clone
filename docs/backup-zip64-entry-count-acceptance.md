# Milestone 5: ZIP64 entry-count backup and independent restore

Completed bounded entry-count slice. This proves an actual archive crossing 65,535 ZIP entries, not an archive or individual payload above 4 GiB. No production behavior, dependency, limit or release version changed.

[Windows run 41740937206](https://github.com/aataat12/immich-clone/actions/runs/41740937206), source `926eedc78d849a35840f0f45dbf9cb0b66a5f9e5`, job `126281976933`: PASS, first attempt. Windows Server 2022, Go 1.27.3 windows/amd64, CGO disabled and GOAMD64=v1. Worker branch was based on main `abe6d20b5fa6e327d02d0a574167c2538cbc2f94`.

## Workload and fixture construction

- Exactly **65,536 actual distinct 32-byte hash-named original files**: 2,097,152 original payload bytes. The catalog also records deterministic filenames, source aliases and 64 hidden items.
- Two completed synthetic receipt sessions of 32,768 entries each preserve the existing 50,000-files-per-selection bound. Each has exact accepted/new counts and byte totals; no duplicates or deleted/error receipts.
- Fixture files and valid catalog/receipt records were seeded directly in 1,024-record transactions through existing catalog helpers. This is deliberately not a 65k browser import, upload or per-file durability benchmark. Fixture writes did not simulate power loss.
- The test computed the exact expected JSON manifest size before filesystem seeding: **9,437,459 bytes**, below the unchanged 16 MiB default. No limit was raised to accommodate the case.

## Executed operations and measured evidence

| Operation | Result |
|---|---|
| Fixture construction | 7.134 seconds |
| Production BackupPlan | Accepted 65,536 originals; existing conservative staging bound **561,168,275 bytes** |
| Actual production StagedBackup | **39,463,589-byte archive**, 3.394 seconds; copy file/byte progress reached its declared totals |
| Production VerifyBackup | Every payload checksum read back successfully, 1.263 seconds |
| ZIP64 structure inspection | Legacy EOCD count sentinels, ZIP64 locator, ZIP64 EOCD counts and actual central-directory reader all agreed on **65,540 entries** |
| Archive metadata | **7,864,558 bytes**, below the unchanged 64 MiB metadata default |
| Independent production restore | **61.346 seconds** with `RestoreLimits{}`; default budgets unchanged |
| Final restored checks | All 65,536 original byte/hash identities, catalog names/hidden/source fields, both sessions and all 65,536 receipt records/counters, required secret and rebuilt gallery counts passed |
| Whole Go test, including cleanup | **82.29 seconds** |

The 65,540 entries are 65,536 originals plus catalog, state marker, synthetic required secret and manifest. The actual ZIP64 binary records were read from their declared archive positions, not inferred solely from a filename, entry list or successful compilation. The archive remained well below 4 GiB.

The archive was moved outside the original state directory, the original state was renamed offline, and restore published to a separate destination. Restoration used only the archive. Every restored original was compared with its independently regenerated deterministic bytes/identity, not repaired from the old library. Receipt pages were checked through both complete sessions, including counts/bytes and absence of false duplicates/errors.

## Reproduction and resource bounds

The normal test suite compiles but skips `TestActualZIP64EntryCountBackupRestore` unless `IMMICH_ZIP64_COUNT=1`. `scripts/Test-ZIP64EntryCount.ps1` opts in on disposable runner storage and restores its environment afterwards. `.github/workflows/backup-zip64-count.yml` runs the actual Windows proof. No user-PC setup is required.

The preflight requires at least **3 GiB free** before seeding. Fixture and archive entry counts are fixed. Production staged-backup byte bounds remain active. Restore uses unchanged defaults: 100,000 entries, 16 MiB manifest, 64 MiB ZIP metadata, 4 GiB catalog cap and 512 MiB worker-memory budget, with destination capacity minus its existing reserve. The enclosing operation context is bounded at 14 minutes, Go test at 17 minutes and Windows job at 18 minutes; these outer bounds do not increase restore limits. Actual peak resident memory and total allocated filesystem footprint were not measured. All temporary sources, archives and restored copies are removed by the fixture cleanup.

Local opt-in-skip compilation and vet passed before Windows execution; an expensive duplicate Linux 65k run was not performed. A local type mismatch in an initial test draft was corrected before publication. There were no failed cloud workload attempts or production fixes.

## Scope still open

This is library backup/restore execution under the CI account, not LocalService/browser or installer acceptance, Windows 10, physical-volume recovery, 65k UI import, natural mixed-media endurance, an original/archive above 4 GiB, full-library throughput, scheduled/incremental backups or complete milestone 5 delivery. Existing 4 GiB HTTP/browser import evidence remains separate. Security Group 2/3 work and known restore progress-read:32 were not investigated; a passing run does not establish a fix. Every-frame Full Scan remains deferred.
