# SEC-03 interrupted upgrade recovery

This is the existing durable full-payload intent/interrupted mixed-copy workstream.
It does not cover the separate metadata-concurrency or genuine prior-release/
full-provider combined closure rows. Published prototype remains 0.1.39.

## Implemented ordering

Setup uses a persistent private sibling `<app>-upgrade-recovery` directory and
its private `payload` child instead of a temporary full-payload backup. Creation
uses the native private-root primitive; existing unsafe roots refuse. Transaction
metadata binds the actual application identity, existing state identity when
present, and a normalized state-location digest. Another state selection or
replaced application/state directory refuses recovery.

Preparation audits before stop, then stops/disables the owned service before
pending recovery and the next snapshot. Pending recovery precedes replacement of
`rollback` state, so a mixed installation cannot become the new rollback source.
A new pending record is exclusive, private, synced and closed after the prior
snapshot but before native application publication. An uncommitted transaction
is never overwritten. Fresh-install recovery preserves any newly initialized
state and removes its service registration rather than claiming prior-state
rollback.

Publication retains complete, synced and verified copies of every selected old
script/runtime/provider file. A separate synced/closed prewrite intent records
all destination identities before any old byte is truncated. The completion
receipt still requires all candidate copies and data closes; intent is never a
successful-publication marker. Interrupted recovery admits old/new/partial bytes
only on the recorded actual destination objects, with existing nofollow,
single-link, private-backup, trusted-code and sharing controls. Missing old leaves,
replacement objects and corrupt backups refuse; already removed new leaves allow
repeat recovery. All state snapshot sources and live state leaves are held and
verified before payload rollback, followed by the prior three-file restoration.

The service remains disabled during mixed copying. Candidate startup is explicit
and manual, after verified complete publication. Health and the existing failure
injection precede transaction commit; automatic startup/failure restart returns
only after commit. A failed pending recovery does not restart mixed code.

Commit/recovery writes a private synced terminal record before retirement.
Retirement holds/audits every surviving selected file before deletion, admits
only fixed metadata and numbered flat backup leaves, and deletes the terminal
record last. Unknown files, unsafe objects or incomplete removal retain recovery
data and report failure. A valid committed terminal marker permits retirement
retry without rolling back a healthy candidate. Empty private root/payload
directories remain for the next transaction.

## Evidence

Initial integrated source `8604f41379b64864175d5bc917fc8cef69bb598f` passed focused
Windows Server 2022 [run41946697550](https://github.com/aataat12/immich-clone/actions/runs/41946697550),
job126980574062. Actual production Inno Setup was killed as a process tree during
native publication of a synthetic 1 GiB payload. The helper was confirmed killed,
the service stopped/disabled, and the prewrite intent retained without completion.
New bundled Setup recovered before taking its next snapshot, completed retry,
preserved saved catalog/executable and independent access-code/original hashes,
verified the complete large payload, retired transaction data and restored auto
startup. A real separate standard user could not modify the persistent intent.
Normal custom/fresh Setup, explicit failed-upgrade rollback, uninstall, native
filesystem/installer/restore tests, build/vet and SEC-01 compatibility also passed.

Final source `649c1450b9e2d3e0e14afbb855ad8918d79f2001` passed
[run41947440346](https://github.com/aataat12/immich-clone/actions/runs/41947440346),
job126983031492, including the full actual Setup kill/retry gate, normal/fresh
installation, explicit rollback/uninstall, native boundaries/resources and closed
permission-policy compatibility. The refinement verifies/holds prior state sources
before code rollback and adds corrupt-state-before-code, empty-root/prepared-only
and fresh-state preservation controls. All passed. The gate canonicalized the Go
sources before validation; its exact formatting diff is retained in the checkpoint
without further behavioral edits. Logs and terminal run conclusion were inspected.
This accepts the same interrupted-upgrade row, not SEC-03 as a whole. Best
remaining estimate changes from three passes to two: metadata concurrency and
genuine prior-release/full-provider/recovery combined validation. No recovery CI
remains active.

## Limits

Process termination is executed evidence; physical power-loss/controller-cache
behavior and every possible interrupted metadata write are not certified. Invalid
or incomplete private recovery metadata refuses and requires administrator review,
not guessed restoration or a successful-completion claim. Real provider payloads,
genuine prior-release upgrades, concurrent metadata replacement and final combined
recovery validation remain in their original separate rows. Disposable Windows
Server 2022 execution does not certify Windows 10. No new provider, dependency,
application catalog schema, release, paid resource or Group 3 work.
