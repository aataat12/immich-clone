# Backups: offline recovery instructions

Shipped in **0.1.37-prototype** after combined installed acceptance; not available in 0.1.36. See [combined acceptance](release-0.1.37-acceptance.md). No library format, dependency, restore budget or service configuration change.

## User behavior

- Backups has an expandable, keyboard-accessible **How to restore a backup** section, usable without creating a job. It explains restoring to a new local folder, stopping the service during the maintenance window, verification, failure/cancellation and retaining the current library.
- A Ready job exposes **Download recovery instructions** beside its displayed checksum. Other states hide and disable it, including after Discard. Reloading a retained Ready backup restores availability.
- The downloaded plain text records that job's exact expected ZIP SHA256 and byte size. Its filename includes the first 12 checksum characters to distinguish archive instructions. Users are told to keep the ZIP and instructions together and compare the downloaded ZIP against the expected checksum.
- Text includes existing CLI restore/verify templates with explicit example paths, PowerShell exit-code checks and instructions to replace all paths. It explains that restarting the original service continues to use the original library; activating a restored library is a separate operation.
- Generation is client-side from the Ready job metadata. It does not create/download/validate the ZIP automatically, mutate the backup, switch the library, execute commands or add an HTTP endpoint. Instructions contain no access code or local library path. The ZIP remains sensitive because it contains originals and the code.

## Actual evidence

[Windows run 41745932733](https://github.com/aataat12/immich-clone/actions/runs/41745932733), source `57703d882beda971da5b06c3d87b53ca494860ee`, job `126298793482`: PASS first execution on Windows Server 2022, Go 1.27.3 windows/amd64, GOAMD64=v1, CGO disabled, existing pinned Playwright 1.62.1 and headless Edge. No production provider was introduced.

Focused existing managed-backup tests passed locally and on Windows. Node syntax checks passed for the modified inline script and browser harness. The standalone actual Windows binary then served the changed page:

1. Logged in against disposable state and imported a synthetic original.
2. Confirmed instructions were unavailable for Idle, and keyboard Enter opened/closed restore help.
3. Created a real Ready archive; downloaded both ZIP and instructions through Edge. The ZIP hash matched the job SHA256. Text matched its exact checksum/size, Windows path separators, restore/verify flags and exit-code checks; no actual access code or fixture state path appeared.
4. Confirmed text download sent no non-GET request and left the Ready job ID/status unchanged.
5. Expanded help at 1280px and 390px; neither overflowed. No page JavaScript error occurred.
6. Reloaded Ready state and confirmed availability; explicitly discarded the temporary archive and confirmed the instructions button hid/disabled while the already downloaded ZIP remained unchanged.
7. Stopped the disposable app, renamed the old state offline and actually ran the same binary's restore and verify CLI operations on the downloaded ZIP to a new target. Exact restored original bytes and unchanged ZIP SHA256 passed.

Harness: `tests/backup-recovery.cjs`; focused workflow: `.github/workflows/backup-recovery-ui.yml`. It refuses an already occupied app port and cleans only its own generated state/downloads. No user data or downloaded ZIP is uploaded as an artifact.

## Remaining acceptance

This is a standalone Windows binary/browser slice with independent CLI recovery, not installed LocalService/combined installer acceptance or Windows 10 proof. The test invokes the real CLI with generated paths; it does not execute all downloaded prose or automate a user maintenance window. The later combined 0.1.37 run passed the installed LocalService path and updated Build Notes. Published baseline is now 0.1.37.

Physical storage outages, controller crashes and full-library endurance remain separate gates. Milestone 5 is still in progress. Security Group 2/3 work, the unexplained progress-read:32 issue and every-frame Full Scan remain paused/deferred. No local action or installation is requested for this development checkpoint.
