# Installer security review — delegated bounded review

Reviewed source at base `3973eade5d52e02328857c3499de964bc202c62c` on 2026-11-02.
This is source-review evidence and a focused acceptance plan, not a new Windows
execution result. No application, packaging, workflow or permission code was
changed by this review. No private input, paid resource or parallel subworker.

S-010's existing Windows Server 2022 result
[41513726020](https://github.com/aataat12/immich-clone/actions/runs/41513726020)
is reused only for what it tested: root explicit Everyone grant removal, inherited
fixture access denial, junction root/ancestor preflight, production helper fresh
initialization, LocalService upload/restart and browser regression. It does not
clear the cases below or establish Windows 10 compatibility.

## Findings and boundaries

| ID | Severity / status | Source observation | Consequence and evidence limit |
|---|---|---|---|
| IR-001 | Medium, open | `Storage-Security.ps1:19-39` replaces the selected root DACL. It does not inspect or replace explicit ACEs or protected DACLs on descendants. `Test-StorageSecurity.ps1:36-49` tests a descendant inheriting its permission from the root, not a descendant with its own explicit/protected grant. | Root hardening cannot be treated as a full-tree access guarantee. Existing file/directory grants may leave secrets/media accessible or mutable. A separate-account descendant reproduction is still required; this review does not claim that test passed. |
| IR-002 | Medium, open | `immichclone.iss:108-110` calls `RunManagement('Prepare')` before new files are installed; `RunManagement` selects `{app}\Manage-Service.ps1`. | The first upgrade from 0.1.29 invokes the retained older preparer before the new reparse/DACL checks. Testing only the new helper directly does not establish a protected upgrade entry point. No new installer transition was executed in this review. |
| IR-003 | High if the installation tree is writable by an unprivileged principal; open, conditional | `Manage-Service.ps1:81` only adds LocalService RX to AppDir, retaining other AppDir write ACEs. Setup accepts a custom install directory. The retained script is executed elevated during Prepare; the installed executable is invoked elevated for initialization at `Manage-Service.ps1:84`. | A writable retained script/executable is an elevated-execution trust boundary. The normal protected Program Files installation has not been shown vulnerable; no privilege-escalation exploit was executed. A hostile writable custom/preexisting installation must be refused or securely normalized and verified before executing its content. |

These cases are release gates for the combined candidate, not a reason to stop
independent People feature coding. No architectural conflict with the agreed
single-user/local-service plan was identified. Administrator/OS compromise remains
outside containment; an ordinary user or compromised LocalService worker must not
silently be equated with an administrator for this assessment.

## Minimal mitigation order

1. Stage the new management script **and its exact shared helper** from the new
   installer payload before Prepare, in a directory that the ordinary-user test
   token cannot modify. Invoke the staged script by its full path while retaining
   the real AppDir/DataDir arguments. Do not invoke the retained installed script.
   Fail before stopping the existing service or publishing any new application
   files if staging/validation fails. Do not claim Inno extraction/private-temp
   behavior from compilation alone; execute the candidate installer.
2. Validate the trust of AppDir and the known executable/script/provider paths
   before elevated execution. Refuse unsafe write permissions or perform a narrowly
   specified repair of installer-owned code after reparse checks and with actual
   permissions/identity tests. Avoid executing an old unverified binary merely to
   validate its own installation. Keep uninstall/rollback behavior and ownership
   rules explicit; do not widen LocalService rights to make tests pass.
3. Add a no-follow, bounded audit of existing application-owned state/diagnostic/
   rollback descendants. Begin with code, access code, catalog, diagnostic and
   backup/rollback paths. Detect protected DACLs and unexpected explicit grants,
   including arbitrary SID grants rather than only Everyone/Users. Either reject
   safely with a fixed diagnostic category or apply a deliberately specified,
   tested repair. Never follow a junction/symlink during a recursive ACL repair.
   Do not silently rewrite arbitrary external-source permissions or originals.
4. Treat concurrent replacement, existing sensitive child reparse points and hard
   links as separate filesystem-identity cases. A string-path preflight is not an
   atomic boundary. Avoid advertising race-resistant containment until handle-
   based operations or equivalent restrictions and adversarial execution prove it.

A first bounded implementation can fail closed on unsafe descendants rather than
perform a broad recursive rewrite. This avoids an unmeasured full-library ACL
migration and unreviewed ownership changes. Any later repair must preserve original
bytes and be independently recoverable; permission failures cannot be reported as
successful remediation.

## Focused acceptance matrix

Use disposable Windows fixtures and a real separate standard-user token. Emit only
fixed labels/counters, never runtime access codes, paths, credentials or raw ACL
inventories. Keep account/service cleanup in `finally`.

| Case | Required behavior |
|---|---|
| File with explicit Everyone read/write, inherited root grant removed | Positive control proves access before mitigation; after mitigation/audit, deny unauthorized secret read/new write or refuse before service/state changes. Test both unprotected and protected child DACLs. |
| Directory with explicit standard-user SID grant | Detect the actual account-specific ACE, not only broad groups. Verify required installer-user read and LocalService modify rights survive any repair. |
| Protected catalog/access-code/diagnostic/rollback child | Check confidentiality and write denial using the actual token. Confirm admin recovery remains usable and originals/catalog bytes remain intact. |
| Junction child inside a directory selected for audit/repair | Reject without touching target ACLs/bytes; do not recurse into the target. Parent/root-only junction tests are insufficient for this case. |
| Actual old 0.1.29 installer to new installer | Preserve existing names/tags/ready AI/originals; prove new staged Prepare runs before file replacement. A modified retained preparer containing a harmless marker must not execute. |
| Old installation with unsafe AppDir code permissions | Ordinary-user positive control demonstrates write ability in the disposable fixture. Candidate refuses or securely repairs before any retained code executes elevated; no arbitrary elevated payload is needed. |
| Staged-helper integrity | Unprivileged test token cannot modify script/helper; exact candidate bytes are selected, including Unicode/space paths. Missing/mismatched helper fails before service stop/new-file publication. |
| Injected failure after prepare and after file replacement | Existing service recovers with old binary/provider/catalog/code; originals unchanged. Verify private rollback ACLs, restart and current-session behavior. |
| Normal default/custom protected install, restore and uninstall | Fresh install and upgrade work as LocalService; independent restore works; uninstall retains state. No dependency install or OS reboot is introduced. |

Full combined installer acceptance remains required after the focused cases. The
previous passing installer run is reusable baseline evidence, not evidence for
new staging, descendant-permission or race controls. Keep clean Windows 10 and
user-reported checks separate from Server execution.
