# 0.1.41 corrective installer acceptance

**PASS: published and checksum verified; exercised installed UI checks accepted
as reported observations. Broader target acceptance remains open.** Prior
releases/tags were preserved.

0.1.40 target update validation found a preparation compatibility defect: normal
Windows ProgramData parent creation/attribute grants were rejected as replacement
rights. Refusal preceded payload replacement and left the retained prior owned
service stopped and disabled. Private support evidence remains local, outside
public source, CI inputs and artifacts. Use the corrected 0.1.41 build.

## Correction and security boundary

The parent audit distinguishes sibling creation/attributes from DELETE,
DELETE_CHILD and permission/owner takeover. Owner/ACE validation, no-follow held
identities, private-child policies and existing race refusals remain enforced.
The namespace is audited before service/ACL writes. Fresh private state anchors
the namespace before later actions; contents initialize through LocalService.
The actual opened application root is verified even if another actor created
the name first. No parent ACL is relaxed. Recovery-stage diagnostics stay fixed
and sanitized. Dependencies, schema, providers, UI scope and storage budgets do
not change.

Access semantics: [file rights](https://learn.microsoft.com/en-us/windows/win32/fileio/file-access-rights-constants),
[file security](https://learn.microsoft.com/en-us/windows/win32/fileio/file-security-and-access-rights),
and [nonempty directory reparse refusal](https://learn.microsoft.com/en-sg/windows-hardware/drivers/ifs/fsctl-set-reparse-point).

## Exact accepted Windows evidence

- Focused source `dabe53da04e936edeb2f80607488590bbf593de7`,
  [run41984997929](https://github.com/aataat12/immich-clone/actions/runs/41984997929),
  job127103552474: native filesystem/restore controls, actual Inno Setup,
  real standard-account sibling creation/rename versus private-child replacement
  refusal, unsafe-parent refusal before side effects, metadata races, killed
  publication/retry, actual Program Files/ProgramData fresh install, LocalService
  metadata ownership, original preservation and closed permission controls PASS.
- Complete build source `e1212539511ee3413a7049e318dc0c2699e8ee0c`,
  [run41986219923](https://github.com/aataat12/immich-clone/actions/runs/41986219923),
  job127107600519: genuine checksum-verified published0.1.39 in actual default
  parents, initial injected rollback and successful direct upgrade both starting
  with its service stopped/disabled, automatic/running0.1.41 recovery, original/
  organization/ready-AI preservation, full native media/AI, package canaries,
  installed browser controls, late rollback, independent restore/reinstall with
  old state unavailable, and final uninstall PASS. Tests/vet, dependency security
  and bounded ENOSPC recovery PASS. Only workflow/handoff notes changed after the
  focused accepted checkpoint; runtime/installer/fixtures are identical.

Earlier candidates stopped on test-harness encoding, a mapped filesystem-right
expectation and the .NET directory rename access-denied exception type. Those
runs were not accepted. The corrected harness preserves actual required refusals.

## Verified deliverable

Artifact13396906359 `prototype-0.1.41-server-tested`, ZIP SHA256
`bfc1bf3165c2843296611fe4afa55ef262d22dad2ce84c6f6d41f32de4ad6e25`.
The ZIP was independently downloaded and verified before extracting four matching
files. Installer `ImmichClone-0.1.41-prototype-x64-setup.exe`,104634954 bytes,
SHA256 `2f61cca971e9d4bcbdd390f3809e9fcc55c7c1988ce40ce2e399dda799e2740a`.
It matches SHA256SUMS.txt. Notes match canonical source after Windows line-ending
normalization. Synthetic browser artifact13396282832 ZIP SHA256
`85fad206e71f076c9996bd6cb5bc78ff101ebe6eb0f7c11af6e3f5626bc2eb38`.
No installer was executed on the user's PC by the build agent.

Automatic approval review rejected a workflow dispatch that would also publish a
release under a build-only request. The publication step was removed; the gate
retained the tested package only. The user subsequently explicitly approved
publication, completed through the separate fixed-artifact publisher below.

Close failed0.1.40 Setup, retain any independent Ready backup, verify the corrected
checksum and use the same application/data folders. Setup handles service recovery;
manual permission edits, uninstall and file replacement are unnecessary. See
validation-0.1.41.md. Windows Server2022 evidence does not certify Windows10
clean-machine, reboot, endurance or physical power-loss acceptance. Milestone7
remains in progress; security5 closed/3 open, SEC05/06/08 remain. Existing-function
UI is retained; deferred Curation/Cinema/global review/Full Scan/RAW/HDR work does
not start here.

## Approved publication and public-download verification

The user approved publishing this tested build. Publication-only source
`dee45fc51338900ef5a6b26d541224af0fd2a4e0` passed
[run41991912500](https://github.com/aataat12/immich-clone/actions/runs/41991912500),
job127125881721. The publisher checks both accepted runs, exact artifact ZIP,
every file hash/size, absence of the new tag/release, and uploaded draft assets
before publication. No rebuild occurred. An earlier publisher attempt stopped
before any release creation on empty JSON-array handling; the corrected check
preserves the no-overwrite boundary.

[Published v0.1.41-prototype](https://github.com/aataat12/immich-clone/releases/tag/v0.1.41-prototype),
release434923147, is public, non-draft and prerelease. Its tag points exactly to
`e1212539511ee3413a7049e318dc0c2699e8ee0c`. All four public downloads were retrieved
again and independently matched the original tested-artifact bytes and hashes:

| Asset | ID | Bytes | SHA256 |
|---|---:|---:|---|
| ImmichClone-0.1.41-prototype-x64-setup.exe |712590790|104634954|2f61cca971e9d4bcbdd390f3809e9fcc55c7c1988ce40ce2e399dda799e2740a|
| SHA256SUMS.txt |712590789|110|e2f8cbabb6edf1804c5f3f84e96a55a96422e8a7c5564737cb8227676326a1c2|
| validation-0.1.41.md |712590793|6542|aea6cbea1a0fe2909cc8f88cabe2e6d76509560593c25556b5fdf0592207853c|
| release-notes-0.1.41.md |712590794|2374|d4d1c9256e79d32ae07cff7a10dcd4d9ea76be00cf8fc8b678e17eb299fdf420|

Build-time notes were deliberately retained unchanged; their pending-publication
wording records the build checkpoint. The release body documents subsequent
approval and completed publication. Only target-PC validation remains pending
for this release step. No local installer execution or service changes occurred.

## November 5, 2026: reported installed 0.1.41 UI validation

The user supplied a completed browser validation report for the running
`0.1.41-prototype`. This is reported observation, not a newly executed test by
the primary agent. Only this generic summary is public; private paths, library
counts, machine details and the raw report are excluded.

| Check | Recorded outcome and limit |
|---|---|
| Running version | PASS:0.1.41-prototype displayed. Installer/upgrade execution was not observed by this session. |
| Existing data | PASS, observed: existing originals and hidden media remained available; no destructive changes. No new original-hash comparison or full inventory audit. |
| Navigation and organization | PASS: requested sections opened; named-face, existing sampled-video-frame and search/filter controls visible. This does not prove fresh inference execution. |
| Bulk import | PASS: saved completed history and resume guidance visible. No new large import or outage/recovery execution. |
| Backups | PASS: Ready/verified status, checksum and recovery-instructions control visible. Backup restoration was not executed. |
| Hidden Media | PASS: existing items and restore/managed-copy controls visible. No restoration or deletion executed. |
| Build Notes | PASS: milestone/status table visible. No milestone completion inferred. |
| Media access | PASS: image/video preview dialogs with metadata opened; a TIFF original downloaded. This is not checksum proof. |
| Authentication | PASS: authenticated pages worked; sign-out returned to the access-code screen. No code read or exposure reported. Re-entry after sign-out was not reported. |
| Browser/layout | PASS: no console warnings/errors observed and current layout/navigation rendered correctly. No separately measured narrow-viewport acceptance inferred. |
| Manual corrections | NOT TESTED: existing names/tags were not changed. |
| Installer and platform | NOT TESTED by this UI session: installation, upgrade, rollback, clean-machine prerequisites, reboot and Windows10 compatibility. |
| Other recovery/endurance | NOT TESTED: backup restore and large imports. No physical power-loss/endurance claim. |

The validator reported consulting the authoritative requirements copy in its
source checkout after finding no copy at the workspace root. No machine-specific
path is published. Requirements remain authoritative.

This accepts the exercised non-destructive UI regression scope only. It adds to,
and does not replace, the independently recorded Server2022 installer/provider/
rollback/restore evidence. SEC08 remains open; milestone7 remains in progress,
security5 closed/3 open (SEC05/06/08). Keep0.1.41. No reinstall, permissions change,
manual correction or live restore drill is requested from this report. No new
runtime changes, workers, CI runs, providers or dependencies were started.
