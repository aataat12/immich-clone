Windows x64 prototype update: offline backup recovery guidance.

- Backups now has expandable **How to restore a backup** help.
- Ready backups offer **Download recovery instructions**. The text records that archive's exact expected ZIP SHA256 and size, plus offline restore/verify command examples and failure guidance. Download the ZIP separately; instructions alone are not a backup.
- The instructions restore into a new folder. They do not execute commands, replace the live library or switch the installed service to recovered data. They omit your access code; the ZIP contains originals and the access code, so retain it privately on separate storage.
- Build Notes now reflects completed bounded import/ZIP64/recovery tests while leaving milestones 4/5 in progress.

Focused Windows Edge and independent CLI restoration passed in [run 41745932733](https://github.com/aataat12/immich-clone/actions/runs/41745932733). Combined publication remains gated on actual 0.1.36 upgrade, rollback, installed recovery downloads, media/AI/import/batch regressions and independently restored reinstall; the pipeline appends its successful run at publication.

Before upgrading, create and download a backup using your existing app and keep the same application/data locations. No separate dependencies are required. [Target-PC checklist](https://github.com/aataat12/immich-clone/blob/work/security-baseline/docs/validation-0.1.37.md).

Unsigned prototype, tested on Windows Server 2022; this does not certify Windows 10. Clean-machine/reboot, physical-storage outages, controller crashes and full-library endurance remain open. No format, native provider or file-size limit changed. Known intermittent restore progress-read:32 remains unexplained. Security Group 2/3 work and every-frame Full Scan remain paused/deferred.
