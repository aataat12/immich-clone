Windows x64 prototype update: Hidden Media management.

- Open **Hidden Media** to find hidden items, copy the managed-file location, or restore an item to the library.
- Permanent managed-copy deletion requires typing **DELETE**. It removes that managed original and associated review data/caches. External source files, historical import receipts and earlier backups remain unchanged. Recovery after completed deletion requires an earlier backup or an explicit reimport.
- Incomplete cleanup stays visible for retry and blocks new backups. If an older executable reimports a pending item, cleanup stops and offers **Keep current copy**.
- New deletion refuses application storage below a 16 MiB free-space floor. Free space and retry; this is a preflight check, not a reservation against concurrent external disk use.
- Oversized interrupted import requests report HTTP 413. The existing 4 GiB per-file limit remains.

Create and download a backup before upgrading with the existing application/data locations. No separate dependencies are required. Safe reanalysis and sampled HEIC/video AI remain available; every-frame Full Scan is deferred.

Focused cloud acceptance: deletion/crash/retry/independent restore run 41697638495; actual NTFS full-volume refusal/retry and transaction rollback run 41702306375; actual 4 GiB HTTP boundary run 41683468062 (independent import job passed; the overall workflow failed an unrelated fixture). Combined installer acceptance passed in [run 41704978730](https://github.com/aataat12/immich-clone/actions/runs/41704978730), source `4e66d707cfc1847d33da87309b6aee53abb13f60`, job `126157762864`, first attempt. Actual 0.1.33 upgrade, both injected rollback paths, installed Hidden Media clipboard/restart/restore, pending-intent rollback, actual older-writer reimport/conflict preservation, independent pre-deletion restoration, retained AI/reanalysis proposals, uninstall and restored reinstall passed. See docs/hidden-media-milestone.md for evidence and limits.

Unsigned early prototype. Windows Server 2022 results do not certify Windows 10. Target-PC, clean-machine/reboot and extended reliability validation remain separate. The known intermittent restore progress-read:32 remains unexplained; a passing run does not establish a fix. Group 2/3 security work remains paused.

Published installer: 104197020 bytes; SHA256 `9ff8331cc1e908a816bb03a89a33e5ef68f7224cd094a7e6eb635b76d23557db`. Target-PC steps: [validation-0.1.34.md](validation-0.1.34.md).
