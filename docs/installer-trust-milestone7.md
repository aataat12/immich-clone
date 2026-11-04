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

## Focused-stage limits (combined follow-up below)

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


## Combined candidate acceptance and SEC-02 closure

[Run 41802642179](https://github.com/aataat12/immich-clone/actions/runs/41802642179),
source `1c9d5e8329759b6c26db8c63f8dea733066ac74b`, job `126481883239`: **PASS**,
first combined execution of the unpublished 0.1.41 candidate, Windows Server 2022
x64/LocalService/Edge. It installed the genuine checksum-verified 0.1.39 release,
rejected an injected initial 0.1.41 failure and restored 0.1.39, upgraded to 0.1.41,
then rejected injected 0.1.42 and restored 0.1.41. Originals, access code, names/tags,
ready AI, sampled video observations, corrections and proposals survived. Full
providers, installed payload privacy canaries, 24-image installed/read-only-restored
matrix, backup downloads, independent restore with old state unavailable, restored
reinstall and both uninstalls passed. Go tests/vet/current Windows-target scan and
actual NTFS ENOSPC recovery passed within their declared coverage.

Downloaded artifact `13308328311` (milestone7-server-tested-candidate) was independently
verified: ZIP SHA256 `3faa5e8ac9877316486cc4cdc1482aa1ffa98a8bb3e0e6e4d4ab2524a238b273`;
installer **104,203,207 bytes**, SHA256
`e1f03e3f8a69caf9639ce2f4fa7860a576e065c5631af7dc3f8d6d68a05f4378`, matching
its packaged SHA256SUMS.txt. The synthetic browser ZIP digest also matched; desktop
and narrow Build Notes screenshots were inspected. Artifact retention is seven days;
this is an unsigned unpublished candidate, not a new supported release.

**SEC-02 is closed in this validated candidate**: bundled staged preparation and
post-install management, actual staged-file token/sharing controls, unsafe static
code-tree refusal, normal custom/Program Files acceptance and genuine previous-
installer upgrade/failed rollback have implementation plus executed evidence.
This closes the original safe-upgrade item within its stated boundary, not all
installer security. Concurrent filesystem replacement/hard-link identity remains
SEC-03; native authority SEC-05, remaining input review SEC-06 and final release/
Windows 10 gates SEC-08 remain open. Installing administrator/OS compromise is not
contained. Original status is now **4 closed / 4 open**: SEC-01/02/04/07 closed;
SEC-03/05/06/08 open. Milestone 7 remains in progress. The next implementation gate
is SEC-03 filesystem identity/race protection. No new Group 3 work, paid signing,
worker or target-PC installation is authorized by this closure.

Publication was removed from this candidate workflow. **0.1.39 remains the published
prototype**; 0.1.41 is not offered for installation and 0.1.42 is only failure injection.
Do not rerun the passing combined gate without a relevant change or failure.
