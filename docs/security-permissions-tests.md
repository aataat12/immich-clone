# Stationary managed-tree permissions acceptance


Current final acceptance: [41535508518](https://github.com/aataat12/immich-clone/actions/runs/41535508518),
source 77d1f1b87411b0ed51834bd054ef2603c616ea0f, job 125586209991, passed **all** steps on Windows
Server 2022. The exact stationary permissions implementation passed its full matrix
again, with 21 unsafe production configurations rejected and all ordinary-user,
normal service, Setup, original-preservation and browser controls passing. Original
SEC-01 is complete; seven other original items and delivery gate SEC-08 stay open.
Earlier failure qualifications below describe earlier runs, not the final result.

Scope: original security list item 1, stationary file/directory ACLs for application
state, originals, catalog, access code, backup, diagnostic and rollback fixtures.
Tests use the exact `Storage-Security.ps1` tree auditor and production
`Manage-Service.ps1`; there is no duplicate acceptance policy in the test script.
Concurrent path replacement and native child/state isolation remain separate
items. No user media, machine inventory, actual access code or fixture contents
are emitted by these tests.

## Cases implemented in `scripts/Test-StorageSecurity.ps1`

- Create a disposable ordinary Windows account and use an actual logon token for
  file reads/writes and the audit-access-denied control. Positive controls prove
  this account can exercise intentionally exposed fixtures.
- Verify legacy root grants/inherited descendants pass the read-only preflight,
  but fail the final `-VerifyInherited` audit before normalization. After exact
  root normalization, require that final audit to pass and unrelated ordinary-user
  read/write access to be denied. The final audit examines inherited as well as
  explicit rights, including required self/propagated rights.
- Grant the arbitrary account Modify directly on state/catalog/code/original/
  backup files, with both protected and unprotected DACLs. Prove actual read/write
  access first, then require production Prepare to refuse the unsafe tree.
- Add an inheritable foreign allow on an originals directory; show child access
  and require rejection at its explicit ancestor. Repeat unsafe direct grants on
  actual disposable log/rollback descendants, protected and unprotected.
- Require rejection of an explicit deny and excessive known-principal rights.
- Create a protected NULL DACL through Windows security APIs; prove ordinary-user
  read/write access and require rejection instead of treating no ACEs as safe.
- Let the ordinary account create a file, then remove the temporary foreign grant.
  Verify its untrusted ownership persists and require refusal without silently
  taking ownership or changing the descriptor.
- Accept intended explicit rules and a fully configured protected descendant;
  require actual ordinary-user read/write denial on safe state/catalog/code/
  original/backup/protected/log/rollback files.
- Reject a protected child missing required service/maintenance rights.
- Reject a child junction without following it or modifying its target bytes/ACL.
- Audit a complete normal tree under the actual ordinary token, then make a child
  unreadable. The same production audit must throw rather than skip the error.

Positive-control writes deliberately change disposable bytes before the audit;
no real original is used. For each production refusal, snapshot watched descriptors
and file hashes, installer diagnostic state and rollback directory entries. Require a nonzero
exit code, unchanged watched bytes/ACLs/diagnostics/rollback and the **same running
service process ID**, demonstrating refusal before stop/restart/snapshot/mutation.
Unsafe grants on an existing rejected tree are **not** claimed to be revoked.
The fixture is refused for installation/upgrade; no recursive repair is hidden
behind the tests or policy.

The existing integration also executes the production helper, initializes fresh
state, confirms LocalService identity, performs authenticated upload/catalog
commit and verifies original hashes across service restart. It checks health and
original bytes again after the rejected fixtures. Account/service/fixtures have
finally cleanup. Junctions are removed explicitly before recursive fixture cleanup.

## Execution status

**Complete original item 1 (SEC-01), stationary managed-tree scope: passed on
Windows Server 2022/Windows PowerShell 5.1.** Evidence:
[41534740456](https://github.com/aataat12/immich-clone/actions/runs/41534740456),
source `81b0f96a51548926c46e4098cbc5ec518c0bdb65`, job `125583595153`.
Root inspected the entire expanded matrix, not only the smoke test. Actual
ordinary-user safe-file denials, exposed-fixture positive controls, all production
unsafe-configuration refusals and unchanged PID/ACL/hash/diagnostic/rollback
comparisons passed. Final inherited-rights verification, protected descendants,
NULL DACL, untrusted owner, junction preservation and actual unreadable-audit
controls passed. Normal production helper/LocalService upload and restart retained
original hashes. The later staged Setup/LocalService/original-preservation/uninstall
acceptance also passed in that job.

**Overall-run qualification:** this job subsequently failed an unrelated new-photo
dialog asynchronous-close browser assertion. It must not be called an all-green
CI run or a completed release. The observed failure occurred after the complete
SEC-01 and staged Setup checks; its browser wait correction and final combined
rerun remain a separate coordinator gate. No permission policy/test assertion was
relaxed to obtain the stationary-tree result.

Early validation issues in Windows PowerShell enum normalization and expected
native-stderr capture were corrected before this accepted matrix. Prior smoke-only
or interrupted runs are not substitutes for the full result above. Expected
rejection subprocesses use `Start-Process` exit status, with private temporary
stdout/stderr files deleted in `finally`; failures emit exception type and decimal
script line coordinates only, not raw messages/paths.

This completes stationary permission implementation/validation for next-release
source, not delivery of a new installer/version. The test requires a disposable
elevated Windows runner, the already-built app via `-Binary`, and the exact shared
packaging helpers. Root normalization followed by final verification is not an
atomic ACL transaction: if post-normalization verification fails, safe root ACL
changes may already exist, but the service has not been stopped. The unsafe
preflight fixtures above reject before any such normalization or mutation.

A successful stationary permissions result does not certify concurrent reparse
race resistance, a worker filesystem sandbox, hostile backup/catalog resource
bounds, clean Windows 10 or the complete installer/upgrade/restore release gate.
