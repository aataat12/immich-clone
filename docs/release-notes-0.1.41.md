# Immich Clone 0.1.41 prototype

Corrects 0.1.40 Setup preparation refusing normal Windows ProgramData parent
permissions. Creating a sibling folder is distinguished from deleting/replacing
an existing protected folder. Unsafe replacement/permission takeover rights,
reparse paths, writable code and metadata races remain refused. Fresh state is
anchored privately before later Setup actions; its contents are initialized by
LocalService. Preparation failures identify the pending-recovery stage safely.

Use this release instead of 0.1.40. The corrected installer supports upgrading
retained 0.1.39 files when the previous failed attempt left the owned service
stopped and disabled. Select the same application and data folders. Setup manages
service recovery; no manual permission edits, uninstall, SDK or routine reboot
is required. Existing files and library data must be preserved.

The release gate tests actual Program Files/ProgramData parent permissions,
genuine checksum-verified 0.1.39, stopped/disabled service recovery, initial/late
failed-upgrade rollback, pinned media/AI providers, original/organization
preservation, browser controls and independent restore/reinstall. The focused
gate retains separate-account concurrency, unsafe-parent refusal and interrupted
Setup/retry controls. Exact source, successful run and installer SHA256 are added
to the release body after those checks pass.

Download the installer, SHA256SUMS.txt and validation-0.1.41.md from this release.
Verify the checksum and retain any existing independent Ready backup. If the
service is unavailable, retain that backup without attempting a new live export.
Do not bypass a refused location or Windows protection block. The package remains
unsigned. Target-PC validation is pending; Server 2022 evidence does not certify
Windows 10 clean-machine, reboot, endurance or physical power-loss acceptance.

Existing-function UI changes introduced in 0.1.40 are retained. Curation/Cinema,
global review counts/filters, Full Scan and RAW/HDR expansion remain deferred.
No providers, schema, import limits or restore budgets change. Milestone 7 and
SEC-05/06/08 remain open. Keep media, access codes, backups and support logs private.
