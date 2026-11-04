# SEC-03 handle-bound filesystem candidate

The existing stationary checks do not prevent replacement between validation and
privileged filesystem access. `internal/winfs` is a candidate primitive, currently
not imported by the application or installer. No new external dependency, model,
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

Primary API references: Microsoft [NtCreateFile](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/nf-ntifs-ntcreatefile),
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
