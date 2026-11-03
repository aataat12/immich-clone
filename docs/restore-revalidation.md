# R-RESTORE-01 — unexplained normal restore refusal

Status: open, under original SEC-04; also blocks SEC-08 delivery. No risk acceptance.

[41553917677](https://github.com/aataat12/immich-clone/actions/runs/41553917677),
source 34412d5a181718523d0c730d4c4381deb1d907f1, job 125648002160 observed
`TestCancelledMediaJobSurvivesRestartAndRestore` fail at Restore with the generic
worker-failed/resource-limit category. It was not a deliberately hostile archive
or a requested cancellation of that restore. The test did not publish a partial
destination or report success. No raw private archive or logs are retained.

The underlying cause was not identified. Parent worker-status/progress/catalog
failure paths now return fixed stage labels and numeric OS error codes, preserving
error categories without paths, contents or raw OS messages. CLI retains sanitized
categories. No resource bounds, validation or cleanup checks were bypassed and
no automatic retry hides a refusal.

20 focused Linux repetitions passed after instrumentation. Actual Windows
[41554459739](https://github.com/aataat12/immich-clone/actions/runs/41554459739), source ae1fbc3011de33df9a1c25594b6f9864f535bdb8, job 125649834966 passed
50 focused repetitions, the full internal Go suite/vet and actual app CLI restores
with independent names/tags/faces/original verification and old state unavailable.
These results did not reproduce the failure and do not establish its root cause.

Before SEC-04 closes again: investigate the recorded failure with bounded Windows
reproduction and the stage diagnostics; determine whether it is an implementation,
OS/sharing/resource or fixture condition, without assuming which. Correct any
implementation/fixture defect with a meaningful negative/normal regression, then
rerun the applicable hard resource/cleanup/preservation and app restore checks.
Do not weaken tests, raise limits blindly, or relabel the outstanding investigation
as completed merely because a subsequent run passes. No user PC action is needed.
