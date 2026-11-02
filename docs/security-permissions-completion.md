# Child-folder and file permissions — stationary-tree implementation

Scope: original security item 1, application-owned state, diagnostics and rollback
roots and **all existing descendants**, not only a selected canary. This is a
fail-closed permission audit plus the existing root-inheritance normalization,
not a recursive ACL/ownership migration. Source implementation is ready; actual
Windows acceptance is required before marking this item complete.

## Production entry point and ordering

`Manage-Service.ps1` invokes `Assert-PrivateDirectoryTree` from the exact bundled
`Storage-Security.ps1` for the full DataDir, diagnostics directory and AppDir's
rollback directory. It validates state placement and refuses unrelated nonempty
state before these audits. Audits run before root ACL changes, diagnostics,
service lookup/stop, snapshots, initialization or error-path writes. The failure
handler keeps `$pathsValidated` false and `$diagnosticsReady` false until all
three audits pass; rejection emits one fixed generic failure, without diagnostics,
rollback, ownership changes or service actions.

After all read-only audits pass, existing roots receive their intended DACLs;
root inheritance then removes old inherited broad grants from inheriting children.
There is no explicit-child rewrite, original deletion or arbitrary new library-
size limit. Iterative enumeration covers every descendant and stops on errors.
An unrecognized, unreadable or disappearing entry fails closed rather than being
silently skipped. Each child's reparse attribute is checked before ACL access or
pushing a directory for enumeration; a junction/symlink child is not followed.

The intended policies remain:

| Tree | Intended Allow ceilings and required protected-child rights |
|---|---|
| State and diagnostics | SYSTEM and Administrators FullControl; LocalService Modify; current installer-reader SID ReadAndExecute unless it is already one of those exact principals |
| Rollback | SYSTEM and Administrators FullControl; no LocalService/reader grant |

Windows' ordinary Synchronize bit is included in Allow-right normalization. Every
root and descendant owner must be an intended policy principal or the current
installer-owner SID; rollback accepts the current installer owner without granting
it additional direct reader rights. An unexpected owner is refused, never silently
transferred. In particular, an arbitrary owner cannot be hidden by an otherwise
normal inherited DACL. Root DACL replacement does not change ownership, so root
owners are checked too.

## Descendant policy

- Null DACLs are rejected. Missing/unreadable ACLs or enumeration fail closed.
- Every explicit Allow must use an intended SID and stay within its rights ceiling.
  Arbitrary account SID grants are rejected, not only Everyone/Users grants.
- Explicit Deny and unsupported callback/object ACE forms are rejected rather than
  attempting incomplete effective-access inference. Protected DACLs apply these
  rules to **all** ACEs, including entries still flagged inherited.
- Inheriting descendants can retain only approved explicit rules. Existing inherited
  root grants are not falsely rejected, because root normalization replaces their
  source. Unsafe explicit intermediate-directory rules are found while auditing
  that intermediate directory, regardless of their propagation to later children.
- A protected child must already grant every intended principal its required self
  rights. A protected directory must also propagate those required rights through
  both ObjectInherit and ContainerInherit without NoPropagate/InheritOnly blocking
  the chain. This admits a correctly configured protected subtree while refusing
  a private-looking ACL that would break service/admin/reader access below it.

No decision is made by matching account display names; access rules are retrieved
as SecurityIdentifier values. Raw DACL ACE inspection prevents callback/object ACEs
omitted by the higher-level access-rule collection from silently bypassing policy.

## Required acceptance

The test worker owns `scripts/Test-StorageSecurity.ps1` and
`docs/security-permissions-tests.md`. The combined Windows run must establish:

1. Actual ordinary-user read/write denial on correctly configured existing state,
   catalog, access code, original, backup, log and rollback descendants, including
   complete protected child ACLs. Preserve installer-reader, service and admin use.
2. Positive controls showing the same real ordinary-user token can read/write
   intentionally unsafe explicit/protected descendants before the exact production
   audit rejects them. Include arbitrary SID grants, excessive intended-reader
   rights, explicit deny, incomplete protected ACL, null DACL and unexpected owner.
3. Unsafe state/log/rollback fixtures cause the exact management entry point to fail
   with the live service PID/running state unchanged and no bytes, root/child ACLs,
   diagnostic output or rollback snapshot changes. Refusal does not retroactively
   revoke an existing unsafe grant; an administrator must correct that configuration
   before retrying. Do not label refusal as automatic ACL repair.
4. Root-inherited broad grants that the existing root replacement can fix are
   accepted and normalized, then verified denied using the ordinary-user token.
5. Child junction and junction-ancestor fixtures reject without touching outside
   bytes/ACLs; unreadable descendant ACL/enumeration executes under a real restricted
   token and cannot become an accepted partial audit.
6. Normal LocalService initialization, authenticated upload/catalog commit, original
   hash preservation, restart, bundled preparation and browser regression remain
   usable under the audited permission policy.

## Boundary of completion

Once these actual Windows checks pass, original item 1 can be marked complete for
stationary application-owned tree configuration and the implemented fail-closed
policy. It is not a promise that an unsafe existing library was automatically
repaired, nor authorization to modify external source permissions.

Concurrent filesystem replacement/reparse swaps remain original item 3: string-
path inspection plus later ACL/file operations is not an atomic handle-relative
boundary. Writable custom AppDir code trust, native worker same-token authority,
full accepted-old-installer/provider upgrade/rollback/restore and clean Windows 10
acceptance remain their separate release gates. Those do not require keeping the
completed stationary permission item indefinitely labelled only a probe.
