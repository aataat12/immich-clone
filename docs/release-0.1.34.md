Windows x64 prototype candidate: Hidden Media management.

- Open **Hidden Media** to find hidden items, copy the managed-file location, or restore an item to the library.
- Permanent managed-copy deletion requires typing **DELETE**. It removes that managed original and associated review data/caches. External source files, historical import receipts and earlier backups remain unchanged. Recovery after completed deletion requires an earlier backup or an explicit reimport.
- Incomplete cleanup stays visible for retry and blocks new backups. If an older executable reimports a pending item, cleanup stops and offers **Keep current copy**.
- New deletion refuses application storage below a 16 MiB free-space floor. Free space and retry; this is a preflight check, not a reservation against concurrent external disk use.
- Oversized interrupted import requests report HTTP 413. The existing 4 GiB per-file limit remains.

Create and download a backup before upgrading with the existing application/data locations. No separate dependencies are required. Safe reanalysis and sampled HEIC/video AI remain available; every-frame Full Scan is deferred.

Focused cloud acceptance: deletion/crash/retry/independent restore run 41697638495; actual NTFS full-volume refusal/retry and transaction rollback run 41702306375; actual 4 GiB HTTP boundary run 41683468062 (independent import job passed; the overall workflow failed an unrelated fixture). Combined installer acceptance remains pending until the candidate pipeline passes. See docs/hidden-media-milestone.md for evidence and limits.

Unsigned early prototype. Windows Server 2022 results do not certify Windows 10. Target-PC, clean-machine/reboot and extended reliability validation remain separate. The known intermittent restore progress-read:32 remains unexplained; a passing run does not establish a fix. Group 2/3 security work remains paused.
