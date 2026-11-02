# Stationary managed-tree permissions acceptance

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
- Verify legacy root grants/inherited descendants pass the read-only audit and
  root normalization then denies unrelated ordinary-user read/write access.
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
no real original is used. For each production refusal, snapshot watched descriptors and file hashes,
installer diagnostic state and rollback directory entries. Require a nonzero
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

Windows execution of this expanded stationary-tree matrix is pending. The earlier
S-010 Windows run established root replacement and simple junction preflight only;
it must not be cited as acceptance of this broader descendant matrix. Record the
combined run/source/job and actual outcomes here after the runner executes it.
The test script remains Windows PowerShell 5.1 compatible and requires a disposable
elevated runner, the already-built app supplied via `-Binary`, and the current
shared packaging helpers.

A successful stationary permissions result does not certify concurrent reparse
race resistance, a worker filesystem sandbox, hostile backup/catalog resource
bounds, clean Windows 10 or the complete installer/upgrade/restore release gate.
