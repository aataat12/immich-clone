# SEC-03 handle-bound filesystem candidate

The existing stationary checks do not prevent replacement between validation and
privileged filesystem access. `internal/winfs` began as an isolated primitive; the fixed installer snapshot/
rollback integration below has now passed focused Windows execution. No new external dependency, model,
schema, runtime prerequisite or product version is selected.

The candidate opens a local drive root, then opens each directory component using
NtCreateFile relative to the previous directory handle. Every component is opened
without following reparse points and checked through the resulting handle.
Directory handles exclude write/delete sharing and remain live until Close.
Single-component files are opened relative to the held leaf directory, without
truncation, then checked for reparse attributes, expected object type and exactly
one hard link before any write. Writable handles also exclude write/delete sharing.
Copy reads/writes the validated handles and syncs the destination. This is not yet
atomic publication or recovery orchestration.

Required actual Windows checks: normal Unicode/space-path copy; a hard-link
positive control that really changes an outside sentinel, followed by candidate
source/target refusal; symbolic-link, traversal, device-name and ADS refusal;
real rename controls before/after pins and leaf/ancestor replacement refusal while
pinned; 2,000 concurrent leaf-replacement/write attempts with successful link swaps,
both accepted/refused outcomes and unchanged outside bytes. The synthetic racer
uses the disposable administrator test process; it is not a separate-account proof
or the user PC. No test silently skips unavailable symbolic-link privileges.

**SEC-03 remains open.** Before closure, integrate actual installer ACL handling,
logging, snapshot/rollback, initialization and Setup publication paths; execute
real account/race and existing permission/upgrade/recovery acceptance. Root creation,
ACL inheritance propagation, late-created descendants, recovery after partial
writes and service/native-worker interaction are separate integration obligations.
Passing this primitive's tests cannot replace those obligations or certify Windows10.
SEC-01/02/04/07 remain closed; current original status stays 4 closed / 4 open.
0.1.39 is still published; 0.1.41 remains the previous unpublished candidate.

Primary API references: Microsoft [NtCreateFile](https://learn.microsoft.com/en-us/windows/win32/api/winternl/nf-winternl-ntcreatefile),
[CreateFile sharing](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew),
and [reparse operations](https://learn.microsoft.com/en-us/windows/win32/fileio/reparse-points-and-file-operations).
The existing Go/x/sys implementations and signatures were inspected locally.
SetKernelObjectSecurity is explicitly not selected for filesystem ACL updates:
Microsoft cautions against that use. Inheritance propagation must be designed and
executed before replacing existing ACL behavior; no silent recursive ACL repair.


## First actual Windows result

Run [41835754542](https://github.com/aataat12/immich-clone/actions/runs/41835754542),
source `50f2d8493ee00f343f7b64cd5e0a14dcf088b073`, job `126594088975`: FAILED.
Real copy, hard-link positive/refusal, symbolic-link/name/device/ADS refusal and
outside-byte controls passed. A held directory still renamed: metadata-only
READ_ATTRIBUTES handles did not establish the claimed sharing exclusion. Directory
opens now request FILE_LIST_DIRECTORY as well. The concurrent actor also achieved
no successful link swaps while the writer repeatedly acquired the name; the test
now yields a short acquisition window and still requires successful actual swaps,
both write/refusal outcomes and unchanged outside bytes. No passing race evidence
or installer integration/SEC-03 closure is claimed by this failed run.


## Corrected actual Windows acceptance

[Run 41836459908](https://github.com/aataat12/immich-clone/actions/runs/41836459908),
source `0a08a79df693e1110505aa5d8e755e8d9bfc56f5`, job `126596509835`: **PASS**
on Windows Server2022 x64, 1.409 seconds for the actual three-test package; vet and
pinned-module verification also passed. No test skipped its link/privilege controls.

- Normal Unicode/space-path copy, hard-link positive control and source/target
  refusal, symbolic-link/name/device/ADS refusal and unchanged outside bytes passed.
- Actual rename controls worked before/after guards. All 200 leaf/ancestor rename
  attempts failed while held; redirected-directory refusal passed.
- The concurrent actor completed 530 actual link swaps during 2,000 writer attempts:
  1,238 writes accepted and 762 refused, with the outside sentinel unchanged. These
  are one disposable-runner interleaving's counts, not a universal race proof.

The directory-sharing correction is production candidate code; the actor yield
only ensures the test actually exercises contested link publication. Error/refusal
and outside-byte assertions were retained. This supplies representative execution
before any installer dependency on the new APIs. It does **not** close SEC-03,
prove full installer ACL/snapshot/rollback safety or certify Windows10. The installed
application and prior unpublished 0.1.41 candidate are unchanged. No new release,
model/provider change, worker, charge, local dependency or user-PC test is requested.

Next checkpoint: use the verified handles in actual privileged installer operations,
including ACL inheritance semantics and snapshot/rollback; then execute adversarial
account/race tests and applicable combined acceptance. Do not substitute another
primitive-only run for that integration or repeat this passing run without a change.


## Installer snapshot/rollback integration candidate

The user authorized one worker for this checkpoint. That worker independently
reviewed staging, file-set failure handling and metadata policies and added
protected-child/receipt/inheritance/drive-alias tests. Cross-compilation/vet are
preparation evidence only; the focused Windows integration run is required.

A standalone CGO-disabled Go `installer-files.exe`, built with the already pinned
Go/x/sys components, now performs the management helper's fixed three-file
snapshot and rollback byte copies. It introduces no service, listener, account,
SDK, external runtime, model or catalog schema. Inno bundles it, hashes it at
compile time and pins the exact extracted executable against write/delete sharing
from before checksum verification through Setup exit. Both Prepare and failure
rollback use the new staged copy; manual installed management defaults to the
installed helper. No per-file CLI or old-helper fallback exists.

Each operation pins the application/data/rollback roots and opens all fixed files
before the first existing-file truncate. A held-handle, normalized volume-GUID
path comparison refuses physical role overlaps hidden by DOS drive aliases.
Rollback directory and every opened snapshot/receipt file are checked using their
actual security descriptors; protected children cannot hide broader explicit
rights behind a private parent. The rollback root must grant admin/system full
self and inheritable rights, with no unrelated reader/writer. No ACL or owner is
rewritten by this helper; destination data files keep their existing descriptors,
while newly created files inherit their role root's existing policy.

Before overwriting snapshots, readiness is invalidated and synced. All three files
must copy, sync, hash-verify and close successfully before a bounded protected
completion receipt is written, synced and closed. The receipt binds the actual
app/data directory identities and all three byte hashes/sizes. Restore checks the
receipt and every source before opening all live destinations; it verifies all
restored bytes and propagates Sync/Close errors. A failed restore exits nonzero,
so management cannot restart or report rollback_complete as though it succeeded.
This is not an atomic three-file transaction: partial restore failure still needs
recovery and is not hidden by a healthy HTTP response. Existing filenames/layout
are retained; the receipt is new installer-only metadata, not an app schema change.

Required focused execution now includes complete round-trip, interruption after
first snapshot copy, unsafe third source/target before earlier truncation,
corruption and changed-root refusal, actual-object privacy and drive-alias controls,
production Inno/custom/Program Files service execution, staged helper mutation
denial and actual injected failed-Setup rollback. Existing SEC-01/02 tests remain.
The worker's review identified the actual-child privacy and drive-alias gaps before
CI; neither root-only ACL checks nor lexical overlap checks are accepted substitutes.

**SEC-03 remains open.** This slice does not change path-based root creation/ACL
inheritance, logging/rotation/fallback, elevated initialization, Inno publication or
uninstall deletion. No unguarded creation fallback is added. Full original-item
account/race and genuine prior-release/full-provider/independent recovery gates
still remain after this focused integration. No new installer release or PC action
is requested, and the original count remains 4 closed / 4 open.

Primary API checks: Microsoft [held-object security](https://learn.microsoft.com/en-us/windows/win32/api/aclapi/nf-aclapi-getsecurityinfo)
and [normalized handle paths](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getfinalpathnamebyhandlew).
There is no string-path fallback if a normalized volume-GUID path query fails.


## Accepted focused installer copy integration

[Run 41842717775](https://github.com/aataat12/immich-clone/actions/runs/41842717775),
source `d74c6ee47d2750d90457a32e3f88119e7a8eb0af`, job `126618212032`: **PASS**,
first focused execution of this integration on Windows Server2022 x64. The worker
contributed the independent privacy/inheritance/physical-alias tests and reviewed
staging, readiness and recovery ordering; the primary agent inspected actual logs.

The new file-set tests passed actual three-file snapshot/restore; interruption
after the first copy with readiness invalidation and refused recovery; unsafe third
snapshot/live targets before earlier truncation; corrupt snapshot and replaced-root
refusal; protected Everyone-readable child/receipt/source refusal; required root
inheritance; and real DOS aliases hiding either direction of physical role overlap.
Both packages passed vet. The underlying NTFS test repeated 200 held leaf/ancestor
refusals and 2,000 concurrent write attempts, with 219 actual link swaps and unchanged
outside bytes (one runner interleaving; not a universal race proof).

Actual production Inno execution observed and attempted modification of all three
staged helper payloads, including installer-files.exe. Elevated and separate real
standard-user writes were denied while Setup held its sharing locks. Existing
custom-path LocalService preparation, exact snapshots and originals/code preservation
passed. An actual injected post-initialization Setup failure invoked verified native
restore, returned a healthy service only afterward and emitted one new rollback_complete
record; all three readiness entries, executable/access-code/original hashes passed.
Unsafe code-root/script/executable/provider refusal, protected custom upgrade,
Program Files install/uninstall and the full existing SEC-01 suite passed.

This fixture used the production Inno script/current runtime with minimal disposable
provider placeholders. It was not the genuine prior-version/full-provider combined
release gate; no fixture installer was published. Root/ACL creation and propagation,
logging/fallback, elevated initialization, Inno publication and uninstall filesystem
operations remain outside this conversion. Partial restore is still a recovery-required
failure, not an atomic transaction. Source ACLs, ownership, attributes and timestamps
are not copied; data destination permissions are retained/inherited as documented.
SEC-03 remains open and the original status stays **4 closed / 4 open**. The published
0.1.39 and earlier accepted unpublished 0.1.41 installer artifacts are unchanged.
No PC action, dependency install, new release, paid service or additional worker.

Next checkpoint: convert the remaining privileged root/ACL and diagnostic writes
without weakening inherited-rights policy; then test those real operations and the
applicable combined prior-release/provider/recovery gate. Reuse this passing evidence
unless relevant source, version or a reported failure changes.


## Native diagnostic integration candidate

The next bounded SEC-03 slice replaces management installer-log append/rotation
and failure-note writes with fixed operations in the existing staged native helper.
Existing directory/file handles and actual descriptors are checked before content
mutation. The log uses the existing private access policy and exact JSON fields;
only allowlisted labels, numeric results and native UTC timestamps are accepted.
Both current/previous leaves are held before rotation. Rotation copies at most
4MiB into the checked previous file and syncs before truncating current; an already
oversized current log is refused. Interrupted log writes may lose diagnostic
records; this is not transactional logging. There is no pathname rename fallback.
The failure note contains only stage/result, audits actual code mutation rights,
and is best effort so refusal cannot suppress installer recovery.

One user-authorized worker independently reviews and tests this slice. Native
Windows execution, including an actual failed Setup with a hard-linked failure
note, is pending. Cross-compilation/vet do not establish acceptance. Root/ACL
creation/propagation, initialization, Inno publication/deletion and final combined
release gates remain outside this conversion. SEC-03 remains open; 4 closed /
4 open, published0.1.39 unchanged. No new dependency, release or PC action.
