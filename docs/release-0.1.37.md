Windows x64 prototype update: offline backup recovery guidance.

- Backups now has expandable **How to restore a backup** help.
- Ready backups offer **Download recovery instructions**. The text records that archive's exact expected ZIP SHA256 and size, plus offline restore/verify command examples and failure guidance. Download the ZIP separately; instructions alone are not a backup.
- The instructions restore into a new folder. They do not execute commands, replace the live library or switch the installed service to recovered data. They omit your access code; the ZIP contains originals and the access code, so retain it privately on separate storage.
- Build Notes now reflects completed bounded import/ZIP64/recovery tests while leaving milestones 4/5 in progress.

Combined installed acceptance passed in [run 41752068473](https://github.com/aataat12/immich-clone/actions/runs/41752068473), source `432481631d1668ad5ced28a105604148db8b0639`, job `126319291211`: actual 0.1.36 upgrade, both rollback paths, installed recovery downloads, media/AI/import/Hidden Media regressions, independent pre-deletion archive restoration and independently restored reinstall all passed. Earlier failed attempts remain in the [acceptance history](https://github.com/aataat12/immich-clone/blob/work/security-baseline/docs/release-0.1.37-acceptance.md).

Windows progress-file reads now share delete access while retaining read-only access. A deterministic Windows regression reproduced `progress-read:32` before the correction and passed afterward with the same conflicting handle still live. Existing restore budget/refusal/cancellation/resource tests passed. CLI restore failures now show a bounded stage/numeric detail; no retry or limit increase was added. The exact cause of every earlier generic refusal was not captured, so no claim is made that all historical restore failures had this cause.

Before upgrading, create and download a backup using your existing app and keep the same application/data locations. No separate dependencies are required. [Target-PC checklist](https://github.com/aataat12/immich-clone/blob/work/security-baseline/docs/validation-0.1.37.md).

Unsigned prototype, tested on Windows Server 2022; this does not certify Windows 10. Clean-machine/reboot, physical-storage outages, controller crashes and full-library endurance remain open. No format, native provider or file-size limit changed. Security Group 2/3 work and every-frame Full Scan remain paused/deferred.

Published installer: **104212004 bytes**; SHA256 `cfa11b989fda6f9dc50b9297320a0f138ea45000940c798d9d74aa318f43c46d`.
