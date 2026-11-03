Windows x64 prototype update: batch Hidden Media actions and the Show/Hide toggle.

- Select hidden items with checkboxes, Select this page or Clear selection. Up to 500 selections stay selected across pages and sorting; leaving Hidden Media or reloading clears them.
- **Restore selected to library** restores the selected items in one operation.
- **Permanently delete selected managed copies** lists the filenames and requires one typed DELETE confirmation. It processes items individually and reports completed deletions, pending cleanup, failures and unattempted items. A failure stops further requests and retains valid remaining selections. Stop after current item is available during deletion. Completed deletions cannot be undone in the gallery; external sources and earlier backups remain.
- The toggle now reads **Show Hidden Media** / **Hide Hidden Media**. The extra Photos & videos Home button beside it has been removed.

Focused Windows Edge acceptance passed in run 41735057371: cross-page selection/restore, confirmation/cancel, replaced-original protection, stop-on-error with untouched later selections, pending cleanup, selection reset and narrow layout. The earlier fixture failure is retained in docs/hidden-media-milestone.md. Publication is gated on combined actual previous-installer upgrade, rollback, installed batch/navigation, media/AI/import regression and independent restore acceptance; the pipeline appends its successful run when publishing.

Create and download a backup before upgrading using the existing application and data locations. No separate dependencies are required. [Target-PC checklist](https://github.com/aataat12/immich-clone/blob/work/security-baseline/docs/validation-0.1.36.md).

Unsigned early prototype. Windows Server 2022 evidence does not certify Windows 10. Clean-machine/reboot and extended acceptance remain separate. Milestone 4 remains in progress; no native provider or file limit changed. Known intermittent restore progress-read:32 remains unexplained. Security Group 2/3 work and every-frame Full Scan remain paused/deferred.

Combined installer acceptance: [run 41737634745](https://github.com/aataat12/immich-clone/actions/runs/41737634745), source `8d9107766d8d5967e63ad5da5bada5d6dfac42bd`, job `126270821113`: PASS, first attempt. Actual 0.1.35 upgrade, both failed-upgrade paths, installed multi-page batch actions and Show/Hide labels, replaced-copy preservation, partial/pending results, source-loss recovery, media/AI/reanalysis and independent restored reinstall passed.

Published installer: 104213538 bytes; SHA256 `33d044091b0c03861a4357ccc7ccbf75ae42340d87be72659455255383a5c79a`.
