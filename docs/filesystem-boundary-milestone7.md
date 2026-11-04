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

One user-authorized worker independently reviews and tests this slice. Actual
Windows acceptance is recorded below; cross-compilation/vet alone do not establish acceptance. Root/ACL
creation/propagation, initialization, Inno publication/deletion and final combined
release gates remain outside this conversion. SEC-03 remains open; 4 closed /
4 open, published0.1.39 unchanged. No new dependency, release or PC action.


## Accepted focused diagnostic integration

[Run41846533710](https://github.com/aataat12/immich-clone/actions/runs/41846533710),
source `7f4b99b3cb7f9749bda5c09e5ce9dcdfb16a24e6`, job `126631387368`: **PASS**
on Windows Server2022 x64, first focused execution. The authorized worker completed
independent review and tests; the primary agent inspected the actual job logs.

Native append/schema/time/signed-code, exact bounded rotation, hard-linked
current/previous/note refusal, actual root/leaf privacy and code mutation-rights
refusal, unknown labels, oversized log and missing-root refusal passed. Snapshot,
readiness, actual descriptor and DOS-alias regressions also passed. Both packages
passed vet. The underlying namespace test completed 200 held rename refusals and
2,000 concurrent writes (1,283 accepted / 717 refused / 371 actual link swaps),
outside sentinel unchanged; counts describe one disposable-runner interleaving.

Production Inno compiled and executed with the current runtime and minimal
disposable provider placeholders. An actual post-initialization Setup failure
encountered a hard-linked optional failure note: outside bytes stayed unchanged,
verified restore returned a healthy service, and one new rollback_complete event
was appended. Code/access-code/original hashes and readiness passed. Staged helper
locks, separate real standard-user trust tests, protected custom/default Program
Files install/uninstall and the existing SEC-01 permission suite passed.

No genuine prior-version/full-provider combined release or Windows10 certification
is claimed. Diagnostic rotation is bounded copy/sync, not atomic logging; failure
may lose log records but must not suppress recovery. Root/ACL creation/propagation,
elevated initialization, Inno publication/deletion/uninstall and final combined
recovery gates remain. SEC-03 stays open, 4 closed / 4 open. No new release,
user-PC action, charges or running workers/CI remain at this checkpoint.

Next: native root creation/ACL application with explicit inheritance behavior,
then remaining init/Setup operations and applicable complete release acceptance.


## Private root creation candidate (bounded scope)

This checkpoint only replaces path-based root creation in the management ACL
function. Each missing component is created by relative NtCreateFile with the
final protected DACL at birth: SYSTEM/Administrators full control, optional
LocalService Modify, optional reader RX, including OICI inheritance. Actual
descriptors are verified while every ancestor/new object stays held. Existing
components are opened without ACL/byte changes. Invalid reader input is rejected
before creation. Failure releases handles and retains safely created components;
there is no pathname cleanup or fallback creation.

**Existing-tree ACL normalization is not converted or claimed safe.** The existing
PowerShell Set-Acl and verification remain, and require a distinct handle-bound
descendant design. Microsoft documents automatic inheritance propagation for
SetSecurityInfo; pinning only the root is insufficient justification for that
conversion. No recursive ACL repair, new policy, dependency, feature or public
release is introduced. SEC-03 remains open and counts remain4 closed/4 open.

Production management calls must provide the staged exact-hash-pinned helper for
root creation; disposable storage acceptance supplies the compiled helper too,
including its ordinary-reader fixtures. Tests cover nested policies/inheritance,
permissive-parent isolation, existing root preservation, links/non-directories,
invalid reader, held ancestors and safe partial progress. Native and actual
Setup/storage acceptance is recorded below; cross-compilation/vet alone is not acceptance.
One user-authorized worker reviews and contributes independent root tests.

API references: Microsoft [new-object security descriptors](https://learn.microsoft.com/en-us/windows/win32/secauthz/security-descriptors-for-new-objects),
[NtCreateFile](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/nf-ntifs-ntcreatefile),
and [SetSecurityInfo inheritance propagation](https://learn.microsoft.com/en-us/windows/win32/api/aclapi/nf-aclapi-setsecurityinfo).


## Accepted focused private-root creation

[Run41849734950](https://github.com/aataat12/immich-clone/actions/runs/41849734950),
source `db37f6ce4d4cac14a829ed5941149260c7d1ede1`, job `126642511204`: **PASS**
on Windows Server2022 x64, first focused execution. The authorized worker completed
review/independent tests; the primary inspected actual job logs.

Native tests passed nested protected creation under a permissive Everyone parent,
exact admin/service/reader rights including duplicate principals, actual child
inheritance, existing unsafe root/child ACL and byte preservation, real symlink
root/ancestor refusal with outside ACL/bytes unchanged, invalid reader and ordinary
file-component refusal, and safe retained progress after later-component failure.
The creation verifier observed held new/existing ancestors refusing renames; an
injected later verification failure retained protected directories and released
handles. Vet and prior snapshot/diagnostic/privacy/DOS-alias regressions passed.
Underlying race controls completed2,000 writes (1,236 accepted/764 refused),398
actual link swaps and unchanged outside bytes: one runner interleaving only.

The real separate-account storage fixture executed native creation alone before
Set-Acl normalization: ordinary-user secret reads and child writes were denied,
and the existing permissive parent descriptor stayed unchanged. Actual production
Setup with current runtime/minimal disposable provider placeholders passed staged
locks, unsafe-code refusals, protected custom/default Program Files installation,
injected post-init failure with verified healthy rollback despite refused linked
failure note, and original/access-code preservation. Existing SEC-01 permission
acceptance passed, including the ordinary-reader policy fixtures.

Scope stops at creation: existing-tree PowerShell Set-Acl propagation, elevated
initialization, remaining Inno publication/deletion/uninstall, real-account races
and complete prior-release/full-provider/recovery gates remain. No full SEC-03
closure or Windows10 certification. Counts remain4 closed/4 open; published0.1.39
unchanged. No new dependencies/features/costs, release, PC action or running
workers/CI. Next checkpoint: handle-bound existing-tree ACL normalization with
explicit descendant/inheritance semantics, without widening this scope.


## Existing-tree ACL pass 1: single-object API proof candidate

The user defines a pass as one session ending in a completion/pause update.
The planning estimate was two passes for existing-tree ACL handling. This first
pass isolates a required native API contract before changing production installer
ACL behavior: an existing directory is opened once with MAXIMUM_ALLOWED through
a held parent, checked no-follow, and receives a protected DACL through its held
handle. Microsoft documents suppression of implicit child propagation for that
access mode. Actual Windows execution must confirm root change, unchanged existing
child ACLs and bytes, and future-child inheritance; documentation is insufficient.
The worker independently reviews and implements adversarial/normal tests.

No installer call site or ACL/refusal policy is changed in this pass. The component
is not reachable through installed helper actions. Do not use this primitive as
proof of whole-tree normalization: a following integration must pin/audit actual
descendants, preserve safe explicit/protected ACEs, replace only inherited rights
as the existing policy intends, handle service sharing honestly, and prove failure
recovery. Unsafe explicit children must still refuse before mutations; do not
protect all children or silently repair unrelated explicit permissions.

The focused filesystem workflow supplies this primitive proof; a filesystem-only
commit skips the larger Setup gate. No dependency, feature, provider, version,
public installer or Group3 work is added. SEC-03 remains open.
Reference: Microsoft [SetSecurityInfo](https://learn.microsoft.com/en-us/windows/win32/api/aclapi/nf-aclapi-setsecurityinfo).


### First ACL proof execution status

Source `8adf2fa91371a3ca502b2f0c7637b27893a4cf84`,
[run41862213427](https://github.com/aataat12/immich-clone/actions/runs/41862213427),
job126686818594: **QUEUED**, no Windows runner assigned at latest observation.
The separate larger installer gate was deliberately skipped. Worker review and
independent controls, Windows cross-compilation and vet passed; actual execution
remains required and must be recorded after logs are inspected. No installer
operation conversion or completed ACL proof is claimed.


## Accepted single-object ACL proof

The earlier queued observation is superseded: actual Windows Server2022
[run41862213427](https://github.com/aataat12/immich-clone/actions/runs/41862213427),
source8adf2fa91371a3ca502b2f0c7637b27893a4cf84, job126686818594 **passed**.
Primary inspected logs this pass. Ordinary named ACL mutation positively propagated
to existing children; candidate held-MAXIMUM_ALLOWED mutation changed only root
with unlocked and locked child variants, preserved protected/unprotected children
and symlink-outside descriptors/bytes, and new children inherited the final policy.
Root/ancestor pin, symlink refusal and actual competing directory-writer refusal
passed. Existing primitive tests/vet passed. No Windows10 certification is inferred.

## Existing-tree ACL integration candidate

Normalize-private-tree is now a fixed helper action for state/log/rollback ACLs.
It opens/pins every object relative to held parents, enumerating actual handles
through Go's GetFileInformationByHandleEx names-only path. Actual owner, nonnull
DACL, explicit/protected rules and required protected inheritance are audited for
all objects before the first permission write. Hard links and reparses refuse.
Root receives protected policy; valid protected descendants are never written;
unprotected descendants preserve supported safe explicit ACE SID/mask/flags and
receive only the intended inherited policy. Owner/protection/effective rights are
verified afterward. Unknown explicit flags/forms refuse; inherited unsupported
entries may be discarded only where existing preflight allowed normalization.

Names/type/link counts are checked again before writing and after parent-first
normalization, detecting late namespace additions. Sharing pins do not prevent
all child creation and do not revoke already-granted owner/WRITE_DAC authority;
no retroactive authority revocation is claimed. Partial failure may retain partial
permission normalization, never reported as success or transactional rollback.

The verified existing service is stopped only after all original read-only static
audits pass, before exclusive native ACL acquisition. If that phase fails, restart
requires final permission audits on all roles; otherwise it remains stopped with
recovery required. No snapshot/content writes occur in that ACL phase. Actual
Setup refusal against an unrelated exclusive writer tests safe prior-service
recovery. Native policy/failure tests run before actual Setup/SEC01 compatibility.
Actual Windows integration acceptance is recorded below; cross-compilation/vet/review
alone are preparation evidence. One authorized worker contributed independent tests.

Scope: state/log/rollback existing ACL normalization only; code-tree icacls grant
remains in the already-known Setup/init path scope. Elevated init, remaining Setup
publication/deletion, uninstall, integrated real-account races and full combined
prior-release/provider/recovery remain open. No new policy, dependency/provider,
feature/version/release, cost, PC action or Group3 work. SEC-03 stays open.


### First normalization integration run

Run41867449513/source20df119b628ec4ebe49ebce4a43ff308d9b05e40,
job126705239534 failed the independent explicit-deny test's post-refusal
ordinary os.ReadFile assertion. Its GW deny also denies SYNCHRONIZE, so a fresh
GENERIC_READ open cannot read despite unchanged bytes. Raw root/child/unsafe
descriptor preservation assertions and all other native normalization/explicit
preservation/protected/inheritance/alias/late-publication/sharing controls passed.
The correction establishes a shared read handle before applying the deny fixture,
then verifies exact bytes through that still-authorized handle after refusal.
No production policy or assertion is weakened. Setup/SEC01 execution was skipped
by the failed native gate; this run is not integration acceptance.


## Accepted existing-tree ACL integration

[Run41868231145](https://github.com/aataat12/immich-clone/actions/runs/41868231145),
source `989fb085d0999da2493463e7323f77feaf78aeca`, job `126707927278`: **PASS**
on Windows Server2022 x64. Primary inspected actual logs; one authorized worker
reviewed integration and contributed independent tests and the fixture correction.

Actual normalization preserved exact explicit directory/file ACE bytes and flags,
protected descriptors, owners and original bytes; admin/service/ordinary-reader
policy and future inheritance passed. Unsafe explicit/protected children, real
directory/file reparses and hard links, deterministic late file/link publication
and exclusive writer refusal all preserved earlier/outside ACLs and bytes. Released
writer retry passed. The corrected deny fixture additionally required a fresh read
permission failure while its preauthorized shared observer verified exact bytes.
Both native packages/vet and earlier copy/readiness/privacy/alias/diagnostic/root
controls passed. Namespace race control:2,000 writes,1,421 accepted/579 refused,
210 actual link swaps; outside unchanged (one synthetic-runner interleaving).

Production Inno/current runtime/minimal disposable providers passed real custom
and Program Files preparation/install/uninstall. An unrelated exclusive writer
caused actual Setup refusal during native ACL acquisition: root descriptor, code,
access code and original bytes stayed unchanged; the safe prior service restarted
only after final-policy verification and became healthy. Injected post-init failure
then used verified snapshot restore, preserved originals/access code and reported
healthy rollback despite refused linked optional note. The full prior SEC-01
separate ordinary-user/storage-policy matrix and LocalService persistence passed.

This completes the bounded state/log/rollback existing ACL conversion checkpoint,
not the entire SEC-03 item or Windows10 acceptance. Code-tree permission grant is
still in the known Setup/init scope. Partial normalization is not transactional;
previously granted owner/WRITE_DAC authority is not retroactively revoked. No
large-library resource/performance or whole-installer race certification. Remaining
init/Setup/uninstall paths, actual integrated account races and genuine prior-
release/full-provider/recovery acceptance remain. Published0.1.39 unchanged;
security4 closed/4 open. No worker/CI left running or new user-PC action.

User-defined pass projection: previous7, now **6 remaining**, consisting of
init1; Setup/publication/code permissions2; uninstall1; integrated races1; combined
closure1. Completed ACL integration0 more passes. Estimates may change only with
explicitly identified evidence/rework, not silently decrementing or replenishing
for conversation/status turns. Scope remains the original SEC-03 checklist.


## Elevated initialization candidate

Eliminate the elevated installer init process: the existing LocalService startup
already opens/initializes the store and access code before listener/health and
SCM Running. Setup creates the private root/policy, registers/configures the same
LocalService and starts it; it never executes the application with -mode init as
the installer. Failure injection moves after actual Health so upgrade rollback
still exercises failure after real service-account initialization. No new token
launcher, initialization implementation, catalog/schema, SDK or service is added.

Windows explicit -mode init now checks TokenElevation before diagnostics.Init or
any state access. Elevated tokens and identity-query failures refuse with fixed
text and exit1; non-elevated/non-Windows behavior is retained. Version mode remains
read-only and bypasses this check. Six disposable service fixtures now create an
empty state root and let their existing LocalService startup initialize it; no
elevated-init bypass is added for tests. Changed scripts are parsed; broad fixture
execution remains a later combined gate, not inferred from parsing.

Independent actual binary controls prove elevated-token refusal before missing
state/log creation, direct mutation, junction traversal or hard-linked leaf writes,
with actual alias positive control and unchanged outside descriptors/bytes. Actual
fresh Program Files Setup uses a brand-new state root and checks LocalService
ownership of marker/catalog/access code and known storage directories, unchanged
commands logging, usable upload and original preservation on uninstall. These
Windows controls passed in the acceptance below; local Linux command tests, Windows cross-build/vet
and one authorized worker's review alone were preparation evidence.

Scope is installer initialization plus explicit elevated init entry point, not a
claim that every elevated CLI mode is secured: verify/show-code/backup/restore
retain their existing library.Open/diagnostic behavior. Their applicable privileged
filesystem operations remain in SEC-03's full boundary inventory and integrated
race/closure review; do not silently drop them or move findings to another ID.
Other remaining Setup/code-permission/uninstall paths remain unchanged. No feature
or Group3 expansion, new dependency/provider/version/release, charge or PC action.


### First initialization gate result

Run41872557865/source603270e486badbf30873d6ad592071dee1ae6724,
job126723288185: all actual elevated-token/direct/junction/hard-link refusal and
outside byte/descriptor controls passed. The workflow then failed its status check
because the PowerShell test used Start-Process ExitCode assertions but did not
explicitly return a successful script exit status. Test-only correction adds exit0
after successful try/finally; exceptions still terminate and candidate exit1
assertions are unchanged. No application policy change or weakened assertion.
Native/actual Setup/SEC01 steps were skipped; this is not full integration acceptance.


## Accepted initialization checkpoint

[Run41873094251](https://github.com/aataat12/immich-clone/actions/runs/41873094251),
source `b31f1aaf2b76ee681aa464714ca8c9f5c96d0cb1`, job `126725226588`: **PASS**
on Windows Server2022 x64. Primary inspected actual logs; the authorized worker
reviewed the bounded change and contributed independent binary controls.

Actual elevated token was independently established. Explicit elevated init
returned fixed refusal/exit1 before missing state/log creation, direct existing
marker/catalog/code/log mutation, junction traversal and actual hard-linked leaf
writes. Positive hard-link control changed/restored the outside file; candidate
refusal preserved all selected/outside descriptors and bytes. Linux command test
preparation, Windows cross-build and vet passed; non-elevated Windows offline init
was not independently executed in this focused gate.

Actual fresh Program Files Setup began with no selected state root or metadata.
After healthy startup the marker, catalog, access code and all four known storage
directories had actual LocalService owner SID, not installer identity. Command log
was unchanged; login/upload and original-byte preservation passed. Custom upgrade,
ACL sharing-refusal safe restart, real failure injected after Health with verified
healthy rollback, retained originals/access code, staged trust controls, fresh/old
original preservation on uninstall and the full SEC-01 matrix passed. Earlier
native ACL/copy/readiness/logging/creation suites passed.

Five broader service fixture scripts additionally get explicit diagnostic-root
mkdir before their existing icacls call because removed init formerly created it.
This small scaffolding correction was independently reviewed/diff-checked, not
executed as those full browser/import/media scenarios here. The initial six fixture
bootstrap edits were Windows-parsed. Production cmd/packaging bytes remain exact
to accepted b31f1aa after these test-only directory additions. No unchanged full
provider/browser gate was rerun or inferred.

Completion is installer plus explicit elevated-init entry points. Other offline
modes retain existing Open/diagnostic paths in SEC-03 boundary inventory/integrated
review; no blanket claim that every elevated CLI write is now secured. Remaining
Setup/code-permission/publication, uninstall, real-account races and combined
prior-release/full-provider/recovery gates remain. No Windows10 certification,
feature/provider/schema/version/release change, cost, local action or Group3 work.
Worker/CI complete. Latest user-defined pass estimate drops6 to **5**: Setup2,
uninstall1, integrated boundaries/races1, combined closure1. SEC-03 stays open.


## Setup pass 1: held code permission grant candidate

Replace only the remaining path-based icacls code-root LocalService RX grant with
fixed native grant-code-read. Payload publication remains Setup pass2. Native
traversal audits actual code owners/DACLs against unchanged SEC-02 trusted mutation
policy before any write, retains every supported safe reader/CREATOR OWNER ACE,
owner and protected bit, and adds only LS RX along uninterrupted unprotected
inheritance chains. Existing protected descendants and their entire subtrees are
left descriptor-identical: especially rollback catalog/access-code snapshots must
never gain LS read rights. No new explicit grant is applied across that boundary.

Directories use the accepted MAXIMUM_ALLOWED suppression of implicit child ACL
propagation. Code files use READ_DATA plus READ_CONTROL/WRITE_DAC/READ_ATTRIBUTES/
SYNCHRONIZE with read sharing only, allowing an executing helper image without
requesting data write/delete access. READ_DATA makes Windows enforce sharing pins;
the ACL action performs no file-content read/write. FILE_NON_DIRECTORY failure with the
specific directory status selects a no-follow MAX directory open; successful
handles are type/reparse/single-link checked. There is no pathname fallback or
broader sharing. Existing state ACL mode still uses its unchanged full-file pins.

Fresh descriptor/namespace checks before apply and exact grant/owner/protection
checks plus fresh namespace validation after apply refuse stale plans/late names.
Partial ACL failure is not transactional and no revocation of previously granted
security-handle authority is implied. The action does not create/delete/publish
payload bytes. Running-own-image, reader/creator-owner preservation, rollback
privacy, unsafe writer/deny, real aliases and deterministic late-name tests are
independently contributed by one authorized worker; actual Windows is pending.
Actual Setup additionally audits rollback privacy after code permission grant.

Scope stays first of the existing two Setup passes: code permissions now; Inno
payload publication next. No new dependency/provider/schema/feature/version,
public release, cost, PC action or Group3 investigation. Uninstall, other offline
command boundaries/integrated races and full combined acceptance remain open.


### First code-grant Windows result

Run41876888789/sourcef387a87fb831e9cbab5dd54b3dcd7a19c6abde5e,
job126738580681 failed only the inherit-only-writer negative fixture on a regular
file. All preserved-reader/creator-owner/rollback, actual running-own-image,
ordinary/service writer/deny, alias and late-name controls passed. Source policy
already inspects non-CREATOR-OWNER inherit-only mutation grants. The corrected
fixture uses a real protected directory and first requires GetSecurityInfo to
show actual OICI+inherit-only BU mutation authority before candidate invocation.
This avoids assuming an inapplicable regular-file inheritance rule survived the
Windows ACL setter. No source-policy change or preservation assertion weakening.
Actual Setup/SEC01 gate was skipped on native failure; no integration acceptance
is claimed for that run.


### Code-grant rerun and fresh-root regression

Run41877617057/source586d183a4776ca0f5bc75ff736dd2eb89627a9ba,
job126741279387: native code suite fully passed, including the required actual
inherit-only directory writer, reader/creator-owner preservation, protected rollback,
running-own-image, links and late names. Actual custom Setup/upgrades, private
rollback audit, ACL acquisition safe restart and post-start verified rollback
passed. Fresh Program Files Setup then returned failure, so this is not full
acceptance. The job listing remained queued while complete logs were available;
actual job logs were inspected instead of treating stale listing as execution proof.

Add an unprotected/inherited code-root positive test matching the missing case.
Fixed-label/numeric verification diagnostics distinguish root/file, owner/protection
and ACE equality/count/order without outputting SIDs or paths; Setup failure
prints only fixed-schema installer diagnostic fields. Protection/preservation
conditions are not relaxed pending actual evidence.


### Reproduced inherited-order verification defect

Run41878394759/sourcea096783b916b54b9ac5e8238cdbd4bafb1e1bb5c,
job126744114489 reproduced the missing unprotected-root case. Final child-directory
verification reported unchanged owner/protection and6 expected/6 actual ACEs,
ordered equality false but exact grant multiset equality true. Actual Windows
reorders inherited allow entries by origin when the root inherits from its parent.
This is a production false rejection, not permission loss or outside mutation.

Correct final comparison to exact multiset equality (SID/mask/flags and duplicate
counts); only allow ACEs are accepted before and after. Owner/protection equality,
exact untouched protected-subtree descriptors, every raw preservation assertion,
pre-apply stale descriptor refusal and fresh namespace checks remain. No deny or
unsupported rule is newly accepted and no extra grant may appear. Actual complete
Setup regression must still pass before checkpoint acceptance.


### Code-file sharing correction before acceptance

Run41878981926/source11372bd9f48fc850181f11c79ab63a06a89a0e2c,
job126746208775 has complete execution logs through runner cleanup with every
native/custom/fresh Program Files/rollback/SEC01 check passed and no error marker.
The API listing still showed queued, so no terminal metadata status is inferred.
The unprotected inherited-root regression passed after exact allow-grant multiset
verification. No whole-checkpoint acceptance yet: final boundary review identified
that security/attribute-only file access does not establish the required sharing
pin. Earlier code tests checked ACL/content preservation but not held-file rename.

Add FILE_READ_DATA to security-metadata file access (still no DATA_WRITE/DELETE),
retaining read sharing only. No actual content I/O is added. This parallels the
already-established directory FILE_LIST_DIRECTORY sharing correction. Independent
positive-control tests now require rename/write/hardlink ability before the hold,
actual held rename/write refusals, changed-link-count refusal after new alias publication, release/retry,
and refusal of a preexisting data writer without ACL/byte changes. The own-image
test remains required to prove executing-code compatibility with the corrected
read authority. This is the existing SEC-03 leaf-pin requirement, not extra scope.
Reference: Microsoft [CreateFile sharing and attribute requests](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew).


### Actual file-pin semantics

Run41880659953/source201b7cd514317ae0f6ca4fad0ef39c1c3607e7d0,
job126751969984 passed actual file rename and data-write sharing refusals, existing
code/rollback/inheritance and running-image controls. The newly added test failed
its overstrong assumption that Windows would also prevent any new hard-link name.
A fresh outside alias to the SAME held managed object can be published; it does
not replace that selected object with an unrelated pre-existing outside object.
No claim is made that sharing freezes link count or revokes metadata authority.

Retain initial incoming-hardlink refusal, actual file identity/namespace pins and
before/after link-count checks. Correct tests require actual outgoing publication,
changed-count Validate refusal with unchanged object ACL/bytes, release/removal/
retry, plus production pre-apply outgoing-link refusal before ANY ACL write with
unchanged old code/root/outside-sentinel ACLs and bytes. Rename/data-writer/incoming
link assertions remain; protected rollback is still skipped exactly. Full incoming
replacement/interleaving matrix remains integrated race/closure work, not silently
claimed by this bounded grant. Product code is unchanged by this test correction.

### SEC-03 code-permission checkpoint accepted

Replaced the installer code-root path-based LocalService RX grant with native
held-tree auditing and handle-based DACL updates. Preserve trusted reader ACEs,
owners and protection; stop inheritance across protected descendants and their
whole subtree so private rollback catalog/access-code snapshots remain unchanged.
Code files request read-data plus security metadata, no data-write/delete, with
read sharing only; real running-image compatibility is verified.

[Run41882276092](https://github.com/aataat12/immich-clone/actions/runs/41882276092)
completed successfully on Windows Server2022, source
802e09733e31caa1a272400dbca4477b8b99c7e5/job126757536563. Complete logs verify
native reader/private-rollback preservation, unsafe grants/aliases, actual rename
and data-writer sharing refusals, late outgoing-alias detection before any DACL
write, unchanged outside objects, running own image, and an unprotected inherited
root. Actual custom and fresh Program Files Setup, LocalService initialization,
upload/original preservation, injected failure/native rollback and SEC01 passed.
Local Windows cross-build/vet passed; worker review/tests completed.

Windows can publish a new hard-link name to the same held object: sharing does
not freeze link count. Initial and before/after link-count checks refuse detected
aliases; the tested late pre-apply alias refuses before every ACL write. These
checks are not atomic with arbitrary later interleavings. No retroactive revocation
of existing WRITE_DAC/owner authority is claimed. Final integrated race/offline-mode
coverage remains open. For admitted ALLOW-only ACEs, post-write verification uses
an exact SID/mask/flags multiset including duplicates because Windows may reorder
inherited ALLOW entries; protected descriptors and stale-plan checks remain exact.
Failed fixture assumptions and the reproduced ordering failure are retained in
filesystem-boundary-milestone7.md; no production trust policy was weakened.

This completes only the code-permission half of Setup. Inno payload publication
is unchanged and remains next. Forecast **5 to4 remaining passes**: Setup payload
publication1, uninstall/cleanup1, integrated boundaries/races/offline modes1,
combined genuine prior-release/full-provider/recovery closure1. SEC-03 remains
open, security4 closed/4 open. No feature, provider, schema, dependency, version,
release, cost, Group3 work or PC action. Keep0.1.39; Server does not certify Win10.

### Setup payload publication candidate (not yet accepted)

Replace only seven root payload files and recursive media/ai publication with
held native relative opens/copy/digest/Sync; Inno extracts bundled source into its
private temporary payload directory. Validate every source/destination and save
all prior payload bytes before first overwrite. Exact held new files are removed
and old files restored on checked mid-copy failure. A private complete publication
receipt permits later Install-failure restoration of scripts/providers as well
as the existing three-file catalog/access-code/runtime recovery. No service starts
on publisher/restore failure; Prepare publication failure leaves prior service
stopped rather than assuming recovery. Data-close failures have no completed
receipt. Temporary recovery is not a crash/power-loss durable transaction; those
interruption/recovery limits remain final integrated/combined acceptance work.
Inno generated uninstall journal/executable, registry and shortcut writes remain
built-in. Fixed seven filename UninstallDelete entries preserve prior removal
behavior; no root wildcard. Uninstall boundary hardening remains next pass.
Worker supplied independent aliases/sharing/partialcopy/fullcode restore/source
shape/overlap/corrupt-backup tests, plus pre-CI empty-required-directory and
missing-target-restore regressions. Cross-build passed. Actual Windows and real
Setup gates pending: do not mark the Setup row complete from compilation alone.
No dependency/provider/schema/version/release/cost or Group3 expansion.

Pre-CI review moved publication into Install before code permissions rather than
Prepare: an ordinary Inno error before PostInstall must not replace application
payload and lose its temporary fullcode backup. Prepare retains only prior
snapshot/stop behavior. Completed payload publication receipt follows data-handle
close; checked copy failures use held automatic restore, close/receipt errors
leave service stopped without claiming automatic recovery. Restore uses existing
only targets, refuses missing live paths before writes and requires all fixed
root filenames. An added deterministic late source-name control must refuse
before first copy. Fresh Install failures attempt exact new-payload rollback
after stopping/removing only their service. These changes await actual focused
Windows acceptance; the original four remaining-pass forecast is unchanged yet.

### Accepted Setup application-payload publication checkpoint

[Run41894239272](https://github.com/aataat12/immich-clone/actions/runs/41894239272)
completed success on Windows Server2022 with exact source
30fa427021139265791f36a14409130adb13b4b9/job126799357326. Complete logs have no
error/failing test and reach runner cleanup. The preceding source33a76c5,
run41893555355/job126796949908, also passed native/real Setup/SEC01; review then
moved publication into Install and added the late-source control, requiring the
accepted rerun. No unchanged green full-provider gate was repeated.

Native payload publication covers exactly seven root files and recursive media/ai
files from the bundled temporary source. Held relative directories/leaves reject
reparses, hard links, unsafe actual mutation ACLs/owners, overlapping roles and
preexisting writers; all targets are audited before old bytes are truncated.
Global source/target namespace and actual-object checks run before copies and
before completion. Original user data is not in the payload set. Sync/digest
checks verify new bytes; data handles close before successful private receipt.

Independent Windows tests use different prior/candidate script/runtime/provider
bytes: full recovery restores every prior file, removes newly-created payload
files via held DELETE handles, preserves unrelated files, and checked mid-copy
injection restores the held prior set. Corrupted backups, missing restore targets,
empty-directory masquerading as required rootfile, actual unsafe ACLs, real file/
directory links and writer handles refuse before earlier byte changes. A real
late source filename published after inventory is refused before the first copy.
Normal actual custom/fresh Program Files Inno installs, rollback of the installed
fixture, access-code/original/private-snapshot preservation and existing SEC01
passed. Fixed native-published rootfiles and fixture provider files were actually
removed on both normal uninstalls while originals stayed intact. These are minimal
disposable provider fixtures; actual full-provider/prior-release acceptance remains
in the final combined gate. Worker independent tests/review and local Windows
cross-build/vet completed.

Inno only extracts payload source; native publication runs inside the checked
Install path before code permissions/service start, so pre-PostInstall Inno errors
do not replace the application payload. A completed fullcode receipt enables
Install-failure payload restore BEFORE prior catalog/access-code/runtime restore
and any health claim. Copy-phase checked errors restore through retained handles.
A close/receipt failure or unverifiable recovery leaves the service stopped;
no automatic healthy recovery is claimed for these cases. Fresh failed Install
attempts restore of its exact created payload after its service is stopped/deleted.
New empty directories may remain after refusal/recovery; no path-based fallback
or recursive error cleanup is introduced.

Scope limits remain explicit: the fullcode backup is in Setup temporary storage,
not a durable all-provider power-loss/crash transaction. The original durable
three-file rollback remains; kill/power-loss/other interruption and full-provider
recovery closure stay in original SEC03 integrated/combined scope. Link-count and
namespace checks do not claim atomic protection against every interleaving or
retroactive revocation of pregranted administrative authority. Inno-generated
uninstaller journal/executable, registry and shortcuts remain built-in; their
boundary review and uninstall/cleanup hardening are still next. Fixed seven
UninstallDelete names preserve removal behavior without an app-root wildcard;
normal removal acceptance does not close the hostile-uninstall boundary.

This completes the bounded application payload installation/replacement checkpoint.
Forecast **4 to3 remaining passes**: uninstall/cleanup and remaining Inno metadata
boundaries1; integrated races/offline commands/interruption handling1; combined
real prior-release/full-provider/recovery validation1. No Group3 reclassification,
new dependency/provider/schema/version/release/cost or local PC action. SEC03
stays open, security4 closed/4 open, published0.1.39 unchanged. Server is not
Windows10 certification. Worker and focused CI finished; paused at this checkpoint.

### Native uninstall/cleanup candidate (acceptance pending)

Uninstall stages only compiler-hash/held-info-verified installed helper and two
scripts into Inno private temp, holds staged files against writes/deletion and
pins all actual app directory-chain components (nofollow, actual directory flag)
through DeinitializeUninstall. InitializeUninstall only stages/audits, so user
confirmation still precedes service stop/deletion. usUninstall invokes staged
management; whole-app read-only native audit precedes stop, then native selected
cleanup reacquires every removal target before any deletion. Remove exact fixed7
rootfiles, recursive media/ai, fixed4 rollback files via read/delete handles, close
children before parents, reject late names, preserve unknown root/rollback files,
state/originals and diagnostics. Delete only verified owned service registration
after successful file cleanup. Partial errors are explicit, not transactional
rollback/healthy restart claims. App/rollback empty dirs remain intentionally.

Remove path UninstallRun/UninstallDelete sections; overwrite Inno same-app log
rather than retaining older logged path-executing/deletion actions. Existing
fixed payload coverage and registry/icons remain current, but real prior-release
upgrade/removal stays final combined gate. Inno metadata removal remains builtin
under held app/ancestor directories; trusted loaded uninstaller integrity is a
premise, not retroactive tamper recovery. No root wildcard, new dependency, cost,
release/provider/schema/feature or Group3 work. Worker provided independent
normal/unsafe-alias/ACL/writer/late-name/overlap/persistent-pin tests. Local cross
build/vet passed; actual Windows and production Inno tests pending. Three-pass
forecast unchanged until actual acceptance.

First uninstall run41897021430/source42e011ad/job126809153565 passed every
native cleanup/audit/refusal control and all three actual hostile-uninstaller
controls (changed script, real hardlink, actual ordinary-user writable provider)
with same service PID, unchanged metadata/registration/original/outside bytes.
Normal custom uninstall returned nonzero, so this checkpoint is NOT accepted and
forecast remains3. Add fixed-stage/numeric/boolean tracing only to locate the
production failure; do not dump private log paths or weaken negative assertions.

Diagnostic run41897729332/source6daa7e14/job126811632412 confirmed actual
normal uninstall reaches directory_pin_ok and staged_payload_ok, then native
audit_result1; service remained Running and every watched payload stayed present.
A narrowly scoped root-only uninsNNN.dat/exe sharing-refusal audit can use held
READ_CONTROL/READ_ATTRIBUTES metadata access, actual descriptor/type/single-link
checks and identical nofollow/no-write/delete-sharing flags. This does not claim
data-writer exclusion for Inno's own live journal/image, never deletes those
metadata leaves, and is not a general payload sharing fallback. The normal strict
read-data audit still runs first; all other paths and nonsharing errors refuse.
Independent real-writer/unsafe-ACL positive/negative tests and actual Inno must
prove this candidate. Fixed audit exit classes aid diagnosis without raw paths.

### Accepted native uninstall and cleanup checkpoint

[Run41898776310](https://github.com/aataat12/immich-clone/actions/runs/41898776310)
completed success on Windows Server2022, exact source
c6d384eca9d675a5053f66e60d0354821fff1f61/job126815286190. Complete logs reach runner
cleanup without error/failing tests. Both custom-folder and fresh Program Files
actual uninstalls removed fixed payload/provider files, service/product registration
and preserved prior/fresh library original bytes. Actual modified-script marker,
hard-linked provider and ordinary-user-writable provider controls refused BEFORE
service stop (same PID), registration/uninstaller journal changes, earlier payload
deletion or outside/original-byte changes. Unknown root and rollback files stayed
intact on normal removal. Independent native controls cover nested Unicode
cleanup, actual aliases/ACL/writers, late-child refusal, physical role overlap and
persistent external directory-pin compatibility. Existing installation/rollback/
privacy and SEC01 suite also passed. Worker independent review/tests and local
Windows cross-build/vet completed.

Uninstaller bootstraps only exact compile-hash/type/single-link-verified helper
and management/security scripts into private temp under source read pins, then
holds and verifies staged copies. Never executes unchecked retained scripts.
InitializeUninstall performs read-only staging/audit; actual destructive work
runs at usUninstall after confirmation. Actual app/ancestor directories remain
nofollow/read-sharing pinned through builtin metadata cleanup and Deinitialize.
Whole-app read-only audit precedes service stop. Cleanup separately acquires ALL
fixed7 root files, recursive media/ai and only fixed4 rollback files with read/
DELETE authority before deletion; no DATA_WRITE, no write/delete sharing, no
path-based recursive fallback. Delete/close children before parent, reject changed
names/links/policy, preserve data/diagnostics/unknown root/rollback files. Only
remove verified service registration after cleanup success. Failure after partial
removal is not transactional and does not claim healthy restart; registration/
service review may be required. Empty app/rollback directories intentionally remain.

First run41897021430/source42e011ad/job126809153565 passed native and hostile
controls but normal uninstall failed. Diagnostic run41897729332/source6daa7e14/
job126811632412 proved directory pin/staging success and read-only audit refusal
with service Running/all selected files retained. Compatibility correction is
strictly root unins+3digits+.dat/.exe, ONLY after actual sharing refusal: relative
metadata-only READ_CONTROL/READ_ATTRIBUTES access, same nofollow/type/single-link/
owner/DACL rules. Independent actual held writers first prove normal DATA_READ
acquisition fails; trusted metadata audit succeeds, actual unsafe mutation ACL
still refuses with that writer live. Nested/lookalike names and all payload
writers retain strict refusal. This is integrity auditing of Inno's active own
journal/image, NOT a data-sharing pin or a payload-deletion exception.

Remove path UninstallRun/Delete actions. Overwrite same-app Inno logs so legacy
logged path actions cannot bypass native cleanup. Current fixed payload coverage
and registry/icons remain; genuine0.1.39 upgrade/removal stays final combined
validation (the current fixture represents current code, not a historical release).
Inno still owns its journal/image/registry/icons. Trusted loaded-uninstaller
integrity is a premise, not retroactive detection of an already-tampered loader.
Persistent directory pins do not claim atomic exclusion of every metadata-leaf
interleaving or retroactive revocation of pregranted admin authority. Those actual
concurrent boundary/offline-command/interruption checks remain original SEC03
integrated scope, not Group3. Earlier temporary fullcode/powerloss limits and
real-provider acceptance remain open as previously recorded.

Bounded uninstall/cleanup checkpoint complete; forecast **3 to2 remaining passes**:
integrated races/metadata-leaf boundaries/offline modes/interruption handling1;
combined genuine prior-release/full-provider/recovery closure1. SEC03 remains
open, original security4 closed/4 open. No new dependency/provider/schema/version/
release/cost, feature or Group3 work. Keep published0.1.39; no local PC action.
Server execution does not certify Windows10. Worker/CI complete; paused here.

### Integrated boundary tranche: command entrypoints and stationary metadata (pending)

Review found the remaining integrated row includes unfinished implementation, not
just execution. Do NOT blanket-refuse elevated recovery commands: published
recovery-guide restores/verifies in admin PowerShell and installed installer SID
is RX. This would break recovery. Current bounded patch: elevated interactive
serve and direct/non-LocalService service mode refuse before diagnostics; service
requires actual SCM context+LocalService token; unknown modes refuse before files.
show-code reads only an existing bounded validated code through held nofollow
root/leaf handles on Windows, never Store.Open/AccessCode creation/diagnostics.
Admin verify/backup/restore are unchanged and explicitly not closed yet.

Native read-only code audit now precedes Prepare service stop and builtin Inno
metadata writes, including stationary hardlink counts omitted by old PS audit.
Actual valid retained journal hardlink test must preserve outside bytes and PID.
Stronger audit requires split optional-note test: stationary alias refuses early;
separate no-alias injected failure must actually record verified-health failure
and rollback, not pass merely from some earlier nonzero Setup exit. Worker
independent early-mode/no-write/show-code tests exercise real aliases and retain
original init controls, without printing access-code contents. Windows build/vet
and Linux command tests passed; actual Windows gates pending.

Still original SEC03 work: safe admin verify/backup reads, supervisor AND pre-main
restore-worker path publication/cleanup, durable full-payload intent BEFORE first
overwrite and interrupted/mixed recovery, real-account concurrent metadata-leaf
substitution, and combined genuine prior0.1.39/full-provider recovery. Temp final
receipt currently cannot recover mixed partial payload after hard kill. Previous
two-pass forecast undercounted implementation; do not call this tranche full
integrated closure or silently remove these requirements from SEC03.

First tranche run41905055196/source081a5f08/job126837361941 stopped at the first
existing init exact-refusal assertion after independent tests switched to bounded
Start-Process waiting. Product init guard/text were unchanged. Test wrapper now
retains native process Handle before waiting, Refresh/Wait captures nonnull
ExitCode and Dispose follows output read; exact exit1/refusal assertions remain.
Failure-only diagnostics use match booleans/character counts, never code contents
or raw error paths. Actual rerun decides whether this is harness-only.

### Accepted entrypoint and stationary metadata checkpoint

[Run41905712864](https://github.com/aataat12/immich-clone/actions/runs/41905712864)
completed success on Windows Server2022, source
f8f1c0a1fbd7582b6fc54d2baa7294ca42fe58c5/job126839683829. Complete logs reach cleanup
with no errors. Real elevated init/serve/direct-service/unknown-mode direct,
junction, hardlink and absent-path controls preserved every selected/outside
byte and descriptor. show-code read a valid existing code despite a deliberately
non-database catalog, with no store initialization or diagnostics; missing code,
root junction and hardlinked leaf refused without generation/mutation. No access
code values were printed in evidence. Actual installed LocalService startup,
custom/fresh Setup, uninstall, privacy/original preservation and SEC01 passed.

Native code audit in Prepare now refuses a real valid retained unins000.dat hard
link BEFORE service stop or builtin overwrite: outside/selected journal hashes
and actual service PID remain unchanged. This closes stationary alias checking,
not concurrent post-Prepare substitution. Split optional-note alias early refusal
from a separate actual injected rollback; exact installer_failed stage verify
service health increased by one, proving the test reached health instead of
passing on an earlier refusal. Native failure-note alias controls remain intact.

First run41905055196/source081a5f08/job126837361941 failed the first preexisting
init assertion after the test wrapper changed process waiting. Retaining native
process Handle/Refresh/non-null exit capture fixed the harness; product bytes
were unchanged for the rerun and exact exit/text/no-write assertions stayed.
Windows build/vet, Linux command tests and worker independent review/tests done.
No additional dependency/provider/schema/version/release/cost/feature/Group3 work.
Keep0.1.39, no PC action; Server does not certify Windows10.

This is a bounded integrated tranche, NOT completion of the old integrated row.
The previous2-pass estimate undercounted unfinished implementation as validation.
Do not silently break the published administrative restore/verify workflow with
blanket elevated refusal: installer identity is RX, so existing read/write store
cannot simply run unelevated. Current patch leaves admin verify/backup/restore
and the pre-main restore worker unchanged; they remain open original SEC03 paths.

Corrected best forecast **5 additional completion passes** (planning range5–7):
1. Administrative verify/backup: true read-only Store/held catalog and original
   reads, exclusive held archive creation/cleanup, compatible RX-user positives.
2. Restore supervisor AND pre-main worker: held archive/parent, private-at-birth
   stage, verified worker handoff and no-escape publication/cleanup.
3. Persistent complete-payload intent synced BEFORE first overwrite, mixed/partial
   byte recovery after actual helper/controller kill and restart/retry, preserving
   the prior durable three-file snapshot until recovery is resolved.
4. Actual separate-account installer/metadata concurrent substitution controls,
   positive actor capabilities, protected outside bytes, normal install/recovery.
5. Genuine0.1.39 prior-release/full-provider/rollback/recovery combined acceptance.
These are the original SEC03 requirements decomposed, not added feature scope.
A pass remains a session ending in completion/pause, not each worker or CI run.
SEC03 open; security4 closed/4 open. Worker and CI complete; paused here.

### Administrative verify/backup candidate (acceptance pending)

User authorized two bounded workers for this checkpoint, superseding earlier
one-worker allowance for this session only. Scope is exactly the first of five
remaining SEC03 rows; no restore-path/durable-installer/concurrency/provider
implementation starts here. Preserve administrative recovery guide and browser/
service backup. New OpenReadOnly uses existing marker/catalog/schema and no
repair, queue rebuild, directories or diagnostics. Windows holds root/originals
and metadata leafs nofollow/singlelink/read sharing; bboltRO consumes the held
catalog file once. Offline backup streams Tx.WriteTo directly into ZIP without
source staging; held original reads validate catalog size+SHA256, and old entry/
configuration/catalog/pendingdelete bounds remain. Source input is not repaired.

New archive uses exclusive FILE_CREATE through held parent with creator/BA/SY
private protected descriptor at birth, refuses existing names and physical source
subtrees, deletes failed output through its exact held handle. Close/unverifiable
cleanup is reported incomplete/private archive may remain; no pathname fallback.
CLI verify/backup dispatch before diagnostics. Independent native tests cover
source/ACL/namespace immutability, aliases/writers/missing/unsupported state,
optional-code no-generation, bounded failures/cancel, private/exclusive output,
actual output aliases, exact failure cleanup and independent restore. Second
worker added actual distinct RX-account CLI success and cannot-write positives,
archive privateACL/manifest checks, admin recovery compatibility, source/no-log
immutability and ACL restoration before resuming service. Local Windows cross
build/vet and focused Linux baseline backup tests passed; actual Windows pending.
Five-item forecast unchanged until accepted execution. No new feature/provider/
dependency/schema/version/release/cost or Group3 work.

First offline run41912052062/source017b5130/job126861893332 passed native
read-only/source+output alias/writer/privatearchive/failure/independentrestore
tests, actual RX account cannot-write+verify+backup, exact source/ACL/namespace
no-log checks, archiveACL+manifest checks and administrator independent restore/
verify. It failed ONLY fixture teardown's original SDDL comparison after temporary
ACL grants were reapplied. Keep this as failed acceptance, not product success.
Test-only correction avoids installed state/executable ACL mutation entirely:
adminverify/backup operate actual installed source; ordinary account operates a
byte/namespace-identical stopped-library copy with RX. Exact source, clone and
installed executable checks remain, admin recovery and privateZIP checks remain.
No product bytes/assertions are weakened. Full gate rerun still required.
Automatic bounded catalog-scale baseline41912052098 also completed success; no
new scale work or repeated scale run is requested.

### Accepted administrative verify and backup checkpoint

[Run41913488167](https://github.com/aataat12/immich-clone/actions/runs/41913488167)
completed success on Windows Server2022, exact source
348dd18b790fcc47e6ccf5228a723be7f0fbd125/job126866916248. Complete logs reach
cleanup with zero errors/failing tests. Native TestOffline matrix passed: unchanged
source bytes/owners/DACLs/namespace including queued previews/staging, missing or
unsupported marker/catalog/buckets, real directory/file symlinks/hardlinks and
preexisting data-writer refusals, optional missing-code no-generation, cancellation/
writer failure, exclusive private output, existing/source-contained/aliased output
refusal, corrupt-original and declared-size mismatch exact-new-file cleanup,
ZIP/manifest readback and actual independent Restore+Verify. Native fd creation,
read-only mapping/locking and direct Tx.WriteTo were executed, not compile-only.

Actual administrator CLI verify/backup read the installed library with exact
source/permission/namespace/no-selected-log checks. A byte/namespace-verified copy
of that stopped library received RX-only permissions for a real separate ordinary
account; its catalog/original write attempts failed while CLI verify/backup
succeeded. Creator/BA/SY-only protected archive permissions and every manifest
payload checksum were independently checked. Existing administrator restore+
new verification recovered a distinct library from that ZIP with unchanged ZIP,
installed source and RX copy. Installed executable ACLs/source never changed.
Normal service resumed; remaining Setup, rollback, uninstall, privacy/original and
SEC01 compatibility passed. This is representative small synthetic recovery,
not the pending genuine prior-release/full-provider/large-volume combined gate.

OpenReadOnly never creates missing state, repairs schema/index/queue, generates
codes or initializes diagnostics. Windows marker/catalog/code/root/originals are
held nofollow/singlelink/read-only. Existing catalog passes once to bboltRO
OpenFile, with nonzero-size/schema/assets checks before use. CLI dispatch occurs
before diagnostic init; Verify hashes held originals. Offline backup streams
existing read transaction directly into ZIP, with no source staging and existing
pending-delete/count/config/catalog/original-size bounds, including actual vs
recorded size and original SHA256. Backup locking serializes shared marker/code
read offsets. Browser/service backup retains its existing path.

Archive parent is pinned and physical source-subtree output refused while valid
sibling output remains supported. FILE_CREATE never opens an existing name and
receives creator/BA/SY protected private DACL at birth. Failure removes only the
held newly-created file before close; no pathname cleanup. Unverifiable cleanup
or close failure is explicit and may leave a private incomplete archive, never a
success claim. Missing optional access-code format behavior is preserved without
creating one. Two authorized workers provided independent code review/native tests
and actual account/recovery tests. Local Windows cross-build/vet and focused Linux
baseline backup tests passed.

First run41912052062/source017b5130/job126861893332 had passed all product/native/
RX/ZIP/admin-recovery assertions but failed test-only ACL reapplication teardown.
Accepted rerun changes only fixture/docs: installed source/executable permissions
are never temporarily edited, exact original-source checks retained+expanded,
RX setup is confined to its verified copy. Product source bytes are unchanged.
Automatically triggered bounded catalog baseline41912052098 also passed; no extra
scale implementation or repeat was requested. All CI/workers completed.

This completes the SAME administrative verify/backup row; best forecast **5 to4
additional passes**. Remaining rows unchanged: restore supervisor/pre-main-worker1;
durable full-payload prewrite intent/interrupted mixed recovery1; separate-account
installer/metadata concurrency1; genuine prior-release/full-provider/recovery1.
Actual trusted archive restoration here preserves functional compatibility; it
does NOT close the untrusted restore path boundary. Those are original SEC03
obligations, not new features or Group3 work. No dependency/provider/schema/version/
release/cost or local PC action. SEC03 open, security4 closed/4 open; keep0.1.39.
Server results do not certify Windows10. Paused at this checkpoint.


## Accepted restore supervisor and pre-main worker checkpoint

Focused [run41922662936](https://github.com/aataat12/immich-clone/actions/runs/41922662936)
completed **success** on disposable Windows Server2022, source
`39d7a2e6f05086d5c329326749c0efa53d82f375`, job126898869323. Full job logs and
terminal run conclusion were checked. This closes the same restore row, not SEC03.

Supervisor pins the destination chain strictly before creating an exclusive,
protected stage. Elevated stage/files use Administrators ownership and BA/SYSTEM
private grants; non-elevated recovery uses its caller's private policy. Originals,
staging, previews and faces directories exist privately before the worker reads
ZIP input. The nofollow single-link archive is held read-only; redundant archive
ancestor pins are released only after that file is secured, permitting archive and
new recovery destination to share a parent without reopening the archive by path.

Native directory rename needs target-directory WRITE sharing. The final parent's
sharing changes only after a protected, DELETE-pinned stage makes it persistently
nonempty; old/new parent identities are checked while both are held. All ancestors
and final parent still deny DELETE sharing. Real FSCTL_SET_REPARSE_POINT succeeds
on an empty control, strict empty-parent pin excludes a writer, and the held-stage
parent rejects actual reparse conversion with DIRECTORY_NOT_EMPTY while retaining
identity. Stage publication is native relative no-replace with before/after object
identity checks. Existing and late-created targets remain unchanged. Failure
cleanup inventories/checks all held descendants before deleting; an unsafe late
alias produces explicit ErrRestoreCleanup and retained private staging, with no
outside change or claimed successful cleanup.

The pre-main Windows worker refuses the old caller-authored request-file protocol
before filesystem access. New requests use a bounded anonymous inherited pipe,
explicit inherited stage/archive handles and identities. Same-image parent and
actual immediate job limits are checked before consuming the request; unknown or
trailing JSON is rejected. Stage owner/DACL/path/identity and archive single-link
identity are checked before ZIP parsing. The existing hard memory/process/output/
time/cancellation/reaping limits remain. This is a resource-bounded worker with
checked filesystem capabilities, not a restricted-token sandbox.

Extraction retains the existing fixed-name allowlist, manifest/CRC/SHA256, actual
copy bounds, structural catalog checks, original verification and preview retry
behavior. Payload files are FILE_CREATE with private owner/DACL at birth. Store
initialization runs only inside this prepared private stage with all fixed paths
already created. Fixed private progress is synced/closed before its strict reader
can open it: live sharing conflicts mean not-ready, never partial JSON acceptance;
a valid bounded nonce/catalog-size record remains mandatory after confirmed reap.
CLI restore dispatch precedes diagnostics initialization and retains absolute
recovery destination/cloud-sync refusal. Existing recovery remains usable.

Actual tests passed: private-before-extraction controls; source/archive byte/ACL/
namespace preservation; independent restored originals; ZIP beside destination;
real symlink, hardlink, junction and preheld writer refusals; existing/late target
preservation; forged old and v2 invocations; unsafe cleanup retention; partial/live
progress exclusion; archive/catalog/entry/metadata/expansion bounds; corrupt/cyclic
catalog cleanup; timeout/output/hard memory/cancellation/reaping controls. The
existing actual administrator CLI recovery from a separate RX user's verified
backup passed along with Setup, rollback, uninstall and SEC01 compatibility. Local
Windows cross-build/vet and focused Linux restore baseline tests also passed. Two
authorized workers supplied independent review/native and boundary tests.

Intermediate evidence retained: sourcebb87cd6/run41919391293 failed both native
rename sharing controls; source8c77451/run41920376499 passed directory publication
and real reparse protection but failed progress rename sharing. These product
conflicts led to the current ordered-parent and strict closed-progress handoffs.
Source8bb334d/run41920973110 stopped at an existing PowerShell fast-process null
exit-status fixture; Process.Start now owns the creation handle and streams while
preserving exact exit/text/timeout/snapshot assertions. Sourceb46ceed/run41921399375
passed product/resource controls but failed junction fixture Go-mode detection;
source07beb1e/run41921938087 passed native junction detection but its old snapshot
helper tried reading a junction as a file. Native nofollow snapshots now retain
actual reparse object owner/DACL/raw data and independent outside hashes, without
traversing or omitting it. Accepted rerun passes the full unchanged requirements.
The final automatically triggered catalog baseline41922663060 passed; no scale
implementation was added. All CI and workers are complete.

Best forecast **4 to3 additional passes**. Original remaining rows are unchanged:
durable full-payload prewrite intent/interrupted mixed-copy recovery1;
separate-account installer/metadata concurrency1; genuine prior-release/full-provider/
recovery combined closure1. No new feature, provider, dependency, schema, product
version, release, cost, Group3 work or local PC action. SEC03 remains open and
security4 closed/4 open; keep published0.1.39. Server execution does not certify
Windows10. Paused at this checkpoint.
