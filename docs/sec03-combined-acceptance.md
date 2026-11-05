# SEC-03 combined validation

Final result: **ACCEPTED**. Original SEC-03 is closed in this unpublished
Server-validated candidate. Security count: **5 closed / 3 open**; SEC-05/06/08
remain open. SEC-03 has zero remaining passes. This latest checkpoint supersedes
the older partial-checkpoint forecasts, not their historical evidence.

Candidate source `c8873e9cdbb197bc9d06af19c8985e69b132c8a0`, branch
`work/sec03-combined-validation`, combines the accepted interrupted-upgrade,
full installer/metadata concurrency and existing-function UI checkpoints.
Windows Server 2022 x64 combined [run41976076284](https://github.com/aataat12/immich-clone/actions/runs/41976076284),
job127075119754, completed success. Every required workflow step passed.
The same source passed the full focused Windows gate41975733374,
job127074014301: native/read-only/restore controls, actual Setup/rollback/retry/
uninstall/separate-account metadata races, unsafe-parent refusal and SEC-01
permission compatibility. Both runs used this exact source revision.
The workflow has no release-publication step. Published 0.1.39 is unchanged.

## Inputs and prior focused evidence

- Genuine published 0.1.39 installer, permanent `v0.1.39-prototype` release,
  SHA256 `543e5742a00e2cf79522e4ddc7bd2c6a9647426c46bb0ca0564644f7d3ea3605`.
- Existing media artifact run41346172469, AI artifact run41551816976; pinned
  worker/tool/model hashes enforced by the acquisition and preparation scripts.
  No new provider/model/runtime or dependency was selected.
- Durable interrupted-upgrade recovery: checkpoint32749aa, actual Setup
  process-tree kill/retry source649c145/run41947440346/job126983031492.
- Full concurrency: source dcc54c1f8ba2846626c1e58c7d9671d9dba9561e,
  run41962976317/job127033425847. The versioned fail-closed manifest verifies
  exact successful source/run/job and recovery ancestry through GitHub's API.
  Actual separate-account capability controls, Setup/rollback/retry/uninstall
  metadata attempts, preservation and Program Files compatibility passed there.
  Unsafe actor-writable namespace parents are refused before privileged writes;
  the policy does not repair or claim to make those parents safe.
- Existing-function UI: d1a47efe/run41958781493/job127019911895; documented
  visible-text-only cleanup and real LocalService fixtures are preserved.
  See [UI scope and evidence](ui-redesign-acceptance.md).

## Restored-library service enrollment fix

The final combined attempt before takeover, run41972378239/job127063353906,
passed the installed visual and functional checks but refused reinstalling an
independently restored library during Prepare. The displayed service-health note
was stale from an earlier deliberate failed upgrade; this attempt never reached
service startup. Restore had protected every descendant's admin-only DACL, while
the installer intentionally refuses to rewrite protected descendants lacking the
final service policy. This was an interaction between accepted security changes.

The published restore root remains protected/private. Child folders and files
carry the same private grants with inheritance enabled, allowing the existing
held-tree installer normalizer to grant LocalService Modify and installer RX.
No service access is granted by elevated restoration itself. Deliberately protected
subtrees elsewhere retain the existing installer refusal/preservation rules.

Native file creation retained protection even when the supplied descendant
descriptor omitted it (catalog control0x9004 in the focused diagnostic). The new
inheriting creation variant still uses private exclusive FILE_CREATE, validates
the actual nofollow/single-link file, and enables inheritance through that same
held handle before returning it for any content writes. It verifies the actual
protected bit is cleared. Only this variant requests WRITE_DAC; existing private
archive/file creation retains its previous behavior. No path-based ACL update,
ownership rewrite, blanket ACL repair or untrusted-grant exception was added.

New Windows regression tests perform an actual independent restore, check a
protected root/private inheriting descendants before enrollment, apply the
unchanged native service-policy normalizer, and verify exact LocalService Modify,
unchanged owners and every selected file's bytes. An unprotected restore root,
unknown child reader and deny rule remain refused; exclusive creation preserves
an existing file's bytes. Installer enrollment requires elevation; normal
unelevated private-restore tests remain in the suite.

Microsoft references: [descriptor control flags](https://learn.microsoft.com/en-us/windows/win32/secauthz/security-descriptor-control),
[inheritance propagation](https://learn.microsoft.com/en-us/windows/win32/secauthz/automatic-propagation-of-inheritable-aces).
The actual Windows tests, rather than those descriptions alone, establish behavior.

## Combined checks and remaining limits

The combined gate passed actual prior-release upgrade and injected rollback,
full installed native media/AI inference, original/code/organization preservation,
private-package canaries, installed browser workflows, independent offline
restore/reinstall with the old state unavailable, restored authentication/media/
faces/reanalysis/corrections and final uninstall. It also passed Go tests/vet,
dependency verification/vulnerability scanning and bounded NTFS ENOSPC recovery.

Earlier UI fixture corrections use actual LocalService for hidden-navigation,
isolate its disposable state before single-asset tests, and sample the inspected
1280x900 installed People page's white background at1200,400. They do not weaken
the elevated entrypoint guard or remove functional assertions. Only synthetic
browser PNGs and the successful unpublished candidate/checksum may be retained.

## Retained artifact verification

- Candidate artifact13392376825, `milestone7-server-tested-candidate`,
  ZIP SHA256 `40f685d0d3cdb5881575f2436b776338565a3024e6a23cff8e65b5d5cd44c049`.
- The ZIP was downloaded and hash-verified. Its only entries are the unpublished
  0.1.41 installer and SHA256SUMS.txt. The installer was streamed from the ZIP and
  independently hashed without executing it:104636634 bytes, SHA256
  `13aa5366a9021c7b57ae66c4a60f5d30dfbe0fa19674b2304f8d71b6e3bef707`,
  matching the packaged checksum file.
- Synthetic browser artifact13392212226 ZIP SHA256
  `f09910deb89eff4a505e1b636b9d410298106422892932ef0a4bab8816dafaf6`.
  Both artifacts expire November12 UTC; they are test retention, not a release.

The independent restored-library startup assertion that failed in run85 now
passes, followed by restored-secret authentication and original SHA256, HEIC
regeneration, named faces, 6A observations/corrections, unapplied proposal restore
and nonduplicating apply, sampled video context, face crops/custom tags, the
24-image restored compatibility matrix and clean uninstall. The original state
directory remained unavailable throughout this independent recovery proof.

Windows Server results do not certify Windows10, physical power-loss, final
clean-machine prerequisites, reboot/endurance, signing or broad recognition
quality. Curation/Cinema/durable review counts/global filters remain deferred.
SEC-05/06/08 remain separate original security work; Group3 stays deferred. No
new charge, release publication, feature or PC action is authorized by this gate.
