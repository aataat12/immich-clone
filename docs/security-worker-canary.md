# Worker authority canary — pending Windows execution

Companion review: [worker isolation](security-worker-isolation-review.md).
This experiment does not modify the production launcher, provider, token policy,
installer or application data. It uses synthetic local canaries and a unique
LocalService service on a disposable Windows fixture only.

## Implementation and expected interpretation

`experiments/worker-access-canary/main_windows.go` imports the production
`immichclone/internal/media.Run` directly from the root module. No copy of its
Job Object or process-creation code is used. The service first runs an ordinary
same-token child as a positive control, then the job-bounded child. It checks
LocalService identity as a boolean and emits no SID/token inventory, path, secret,
file content or hash. `scripts/Test-WorkerAccessCanary.ps1` builds CGO-disabled,
configures synthetic ACLs, starts the service, validates the exact boolean report
and actual canary bytes, then stops/deletes the service and fixture in `finally`.

A successful probe must establish all of the following:

| Question | Ordinary LocalService child | Same child via production Job Object |
|---|---|---|
| Read synthetic secret/catalog with LocalService Modify permission? | Allowed | Allowed |
| Overwrite synthetic original under that permission? | Allowed | Allowed |
| Read/create files under SYSTEM/Administrators-only negative-control directory? | Denied | Denied |
| Commit 600 MiB allocation? | Allowed positive control | Refused by current job limits |
| Spawn a child? | Allowed positive control, then killed/waited | Refused by one-active-process job limit |

The independent denial controls show that access is governed by the token/ACL,
not that the helper blindly returns success. The output-control interpretation
is explicit: `job_secret_read_allowed` and `job_original_write_allowed` being
true are expected evidence of the missing boundary, never a sandbox pass.
The synthetic catalog is not bbolt and bypasses no database lock; it tests file
read authority, not modification of a live locked catalog.

## Evidence and limits

Go 1.27.3 Linux-to-Windows cross-compilation with `GOOS=windows CGO_ENABLED=0`
passed. That proves compilation only. Actual service/token/access/resource
execution on Windows is pending; root will attach the combined focused workflow
run and exact source checkpoint after inspecting results. No Windows success or
isolation claim is made by this document before that execution.

The existing accepted memory/spawn/death/cancellation and native-inference runs
remain reusable evidence as listed in the companion review. This experiment adds
actual current file-authority controls. It does not replace parser quality tests,
original-preservation/restore tests, full installer acceptance, Windows 10 checks
or a future restricted-token/AppContainer/worker-identity experiment. No worker
mitigation is selected or integrated on the basis of compilation alone.
