# Milestone 7: focused installer code trust acceptance

[Run 41801504714](https://github.com/aataat12/immich-clone/actions/runs/41801504714),
source `df25e3ee6a0d085938a38d533fc05fb4f55e5f07`, job `126478120619`: **PASS** on
Windows Server 2022 x64. This is a production installer-helper change with actual
Setup execution, not a documentation-only review or a product release.

## Implemented controls

- The new bundled helper checks the existing code tree's owners and ACLs before
  logging, ACL changes, service stop, rollback snapshots or elevated app execution.
  Unrelated-user/LocalService mutation rights, null DACLs, unsupported ACEs,
  untrusted owners and reparse entries are refused. Inherit-only write grants
  are included. Unsafe code permissions are not silently repaired.
- Setup uses its new extracted management payload for both Prepare and Install.
  It never selects a retained installed script for those actions.
- Setup opens both extracted helpers without write/delete sharing before hashing
  and holds them through Setup exit. An incompatible preexisting writer causes
  refusal. Existing compile-time hashes remain in force.

SYSTEM, administrators, Windows TrustedInstaller and the current installer
identity are trusted code owners/writers. CREATOR OWNER inherit-only rules resolve
on creation, while actual child owners/rights are checked. This does not claim a
boundary against the installing administrator's own account.

## Actual execution

The workflow compiled the production Inno 6.7.4 script with the current application
binary and explicitly disposable minimal provider placeholders. The resulting
0.0.0 fixture installer was neither retained as a distributable nor published.

| Gate | Observed result |
|---|---|
| Unsafe root/script/executable/provider permissions | A real separate standard-user token first opened each unsafe fixture for write. Actual Setup then refused each configuration before stopping/restarting the existing service. The same service PID, executable bytes, consistent logical catalog export and rollback snapshot bytes remained. The executable probe is a nonrunning copy, not a modified running binary or arbitrary elevated payload. |
| Staged-helper protection | The controller observed the actual extracted helper process during Setup. Write opens on both exact helper files failed with sharing violation 32 even for the elevated account; the separate standard-user token also could not open either for write. The actual staging directory passed the production trust audit. |
| Normal custom-path upgrade | Actual Setup selected the new bundled preparer, never executed the retained marker helper, preserved original/access-code bytes and data location, created real rollback snapshots, installed exact new script/helper bytes, and returned healthy LocalService. A further protected upgrade passed after the hostile controls. |
| Default Program Files inheritance | Actual Setup installed into a newly created Program Files child with its normal inherited permissions. The installed helper denied standard-user modification, LocalService became healthy and uninstall preserved the original. |
| Static junction refusal | Actual Setup refused before publishing files/service/state. Outside target ACL/bytes and retained code were unchanged. |
| Existing permission protection | The full existing SEC-01 descendant/owner/null-DACL/explicit-rights/denied-audit/junction and real standard-user negative/positive controls passed. Production LocalService upload/restart and original-byte preservation also passed. This retains its closed status; it is not a new security item. |

The live bbolt file is exclusively locked. Logical catalog preservation is checked
through the service's consistent export after the fixture's existing preview
settles; it is not reported as a raw live-database hash. Executable, closed rollback
snapshots and original/access-code comparisons retain their byte/hash assertions.

## Limits and next gate

**SEC-02 remains open**, pending the combined actual prior-release/full-provider
upgrade and injected rollback gate with this candidate and its remaining related
boundaries. This current-runtime fixture does not replace that transition.
**SEC-03 remains open**: stationary ACL checks and file-specific sharing locks do
not close concurrent directory/ancestor replacement or hard-link identity issues.
SEC-05 native authority, SEC-06 full remaining input review and SEC-08 final release
acceptance also remain open. Original count is still **3 closed / 5 open**.

No provider/model/dependency/schema/original format changed. The focused commit
marker skips the release-specific application workflow and runs the dedicated
installer gate, avoiding accidental rebuilding/republishing of 0.1.39. The shipped
0.1.39 prototype is unchanged. No new installer, Windows 10 certification, signing
setup, paid resource, worker or user-PC action is implied.

Earlier failed compile/fixture/harness runs are recorded in
[milestone 7](milestone7-hardening.md); they are not presented as passed acceptance.
