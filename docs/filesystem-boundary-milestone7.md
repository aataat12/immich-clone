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
actual held rename/write/new-hardlink refusals and no outside alias, release/retry,
and refusal of a preexisting data writer without ACL/byte changes. The own-image
test remains required to prove executing-code compatibility with the corrected
read authority. This is the existing SEC-03 leaf-pin requirement, not extra scope.
Reference: Microsoft [CreateFile sharing and attribute requests](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew).
