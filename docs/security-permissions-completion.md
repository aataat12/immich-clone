# Child-folder and file permissions — stationary-tree implementation


Current final acceptance: [41535508518](https://github.com/aataat12/immich-clone/actions/runs/41535508518),
source 77d1f1b87411b0ed51834bd054ef2603c616ea0f, job 125586209991, passed **all** steps on Windows
Server 2022. The exact stationary permissions implementation passed its full matrix
again, with 21 unsafe production configurations rejected and all ordinary-user,
normal service, Setup, original-preservation and browser controls passing. Original
SEC-01 is complete; seven other original items and delivery gate SEC-08 stay open.
Earlier failure qualifications below describe earlier runs, not the final result.

Scope: original security item 1, application-owned state, diagnostics and rollback
roots and **all existing descendants**, not only a selected canary. This is a
fail-closed permission audit plus the existing root-inheritance normalization,
not a recursive ACL/ownership migration. **Original item 1 is complete for the
implemented stationary-tree policy** at source
`81b0f96a51548926c46e4098cbc5ec518c0bdb65`, with the actual Windows Server 2022
acceptance below. This is an implementation/test checkpoint, not a new installer
or a claim that the user's installed 0.1.29 already contains these changes.

## Production entry point and ordering

`Manage-Service.ps1` invokes `Assert-PrivateDirectoryTree` from the exact bundled
`Storage-Security.ps1` for the full DataDir, diagnostics directory and AppDir's
rollback directory. It validates state placement and refuses unrelated nonempty
state before these audits. Audits run before root ACL changes, diagnostics,
service lookup/stop, snapshots, initialization or error-path writes. The failure
handler keeps `$pathsValidated` false and `$diagnosticsReady` false through all
preflight and final verification passes. Preflight rejection emits one fixed generic
failure without ACL/ownership changes, diagnostics, rollback or service actions.

After all read-only preflights pass, existing roots receive their intended DACLs;
root inheritance then removes old inherited broad grants from inheriting children.
A second complete read-only audit with `-VerifyInherited` checks the actual final
root and descendant ACLs, including every inherited ACE and required self/directory
propagation right. Roots must also have inheritance protection enabled. All three
final audits must pass before diagnostics, service actions or data-content changes.
This verifies propagation rather than assuming it worked. A failure in final
verification can leave safe root ACL normalization already performed; the helper
does not claim transactional ACL rollback. It still changes no service state or
data content and emits no diagnostic file on that rejection.
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
- Final `-VerifyInherited` applies these ceilings and completeness requirements to
  every root and child regardless of inheritance status; protected roots are
  required. It rejects an unnormalized broad inherited grant or a missing inherited
  service/admin/reader right even if preflight correctly admitted that repairable
  tree. The preflight and final verification modes use the same ACL evaluator.

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
   accepted by preflight but rejected by final verification before normalization;
   after normalization final verification passes and the actual ordinary-user token
   is denied. Verify correct protected root status and missing inheritance refusal.
5. Child junction and junction-ancestor fixtures reject without touching outside
   bytes/ACLs; unreadable descendant ACL/enumeration executes under a real restricted
   token and cannot become an accepted partial audit.
6. Normal LocalService initialization, authenticated upload/catalog commit, original
   hash preservation, restart, bundled preparation and browser regression remain
   usable under the audited permission policy.

## Actual Windows acceptance and item closure

[Run 41534740456](https://github.com/aataat12/immich-clone/actions/runs/41534740456),
source `81b0f96a51548926c46e4098cbc5ec518c0bdb65`, job `125583595153` executed
the complete SEC-01 matrix on Windows Server 2022 using the production policy and
management entry point. The primary agent inspected the actual job logs. Passed:

- Correct stationary state, catalog, access-code, original, backup, diagnostics and
  rollback permissions denied the real ordinary-user token read/write access,
  including complete protected descendants. Intended installer-reader,
  LocalService and administrator controls remained usable.
- Actual unsafe read/write positive controls preceded exact production rejection
  for arbitrary explicit/protected SID grants, excess intended-reader rights,
  deny rules, incomplete protected access, null DACLs and untrusted owners. Rejection
  preserved the running service PID and state, bytes, owner/ACL values, diagnostics
  and rollback snapshots; unsafe preflight configurations were not silently repaired.
- Inherited root grants were admitted by preflight but rejected by final verification
  before normalization. After root normalization, complete inherited verification
  passed and the actual ordinary-user token was denied. Protected-root and correct
  protected-child controls passed; denied descendant audit did not become a partial
  success. Junction fixtures left outside target ACLs and bytes unchanged.
- Production initialization, authenticated upload/catalog commit, original hash
  preservation and service restart passed. The actual staged Inno Setup probe
  passed its junction/retained-preparer controls, normal existing LocalService
  transition, rollback snapshots, exact helper bytes, preserved state and uninstall
  retention. Go library/web tests, vet and application build also passed.

The overall job at this checkpoint was **not green**: after these security,
service and Setup assertions passed, a later photo-dialog browser assertion failed
while waiting for asynchronous dialog closure. The primary agent is correcting
that test wait separately. This document does not reinterpret that UI failure as
a successful complete browser run, nor does item closure depend on an unexecuted
security fix. The SEC-01 production source and full policy matrix above passed.

Earlier failed checkpoints are retained: run `41532763067` stopped on the initial
PowerShell InvalidCast; early smoke in `41533454482` isolated the byte-backed
AceFlags bitwise operation, corrected using exact-typed `Enum.HasFlag`. Run
`41533819218` passed production smoke/normal service controls but exposed an
expected-rejection test harness treating native stderr as a terminating exception.
The harness now captures the actual nonzero process exit status privately. Neither
failure was waived or counted as permission acceptance; final verification above
uses the corrected production flag handling and complete rejection matrix, without
weakening policy.

## Boundary of completion

Original item 1 is closed for stationary application-owned tree configuration and
the implemented fail-closed policy. This is not a promise that an unsafe existing
library was automatically repaired, nor authorization to modify external source
permissions. A refused unsafe configuration still requires deliberate administrator
correction before installation can continue.

Concurrent filesystem replacement/reparse swaps remain original item 3: string-
path inspection plus later ACL/file operations is not an atomic handle-relative
boundary. Writable custom AppDir code trust, native worker same-token authority,
full accepted-old-installer/provider upgrade/rollback/restore and clean Windows 10
acceptance remain their separate release gates. Those do not require keeping the
completed stationary permission item indefinitely labelled only a probe.
