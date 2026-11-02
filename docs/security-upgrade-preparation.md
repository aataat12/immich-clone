# Bundled upgrade preparation candidate

Scope: IR-002 from `security-installer-review.md`. Source base for this change is
`9c8dcbd`. No version, original format, provider, model, dependency or schema change.
Focused Windows Server 2022 execution passed as recorded below; broader release
acceptance remains separate.

## Change

`packaging/immichclone.iss` now extracts `Manage-Service.ps1` and its sibling
`Storage-Security.ps1` from the new Setup payload before application file copying.
They are the first `[Files]` entries to avoid decompressing the provider payloads
before preparation when solid compression is enabled. Inno computes each source
SHA256 at compile time and checks the extracted file against it before execution.
Extraction, read/hash, or helper failures return a fixed preparation error before
application-file publication.

Prepare invokes the extracted management script by full path with the real
AppDir/DataDir arguments. Its `$PSScriptRoot` resolves the exact extracted sibling
helper. It never invokes the retained installed preparer. Existing service
ownership, stop/snapshot, rollback-directory, data-location and post-install
management semantics remain in the production helper. The new validation also
runs on fresh installation instead of waiting for retained script detection.
No changes were made to `Manage-Service.ps1` or its ACL policy in this slice.

Installed-script use for post-install/uninstall is unchanged; this is not a claim
that a writable AppDir, existing child explicit ACLs or concurrent path replacement
is safe. Those findings remain release gates. Hash checks do not authenticate an
installer against deliberate replacement of the installer itself, and cannot
replace trusted staging-directory permissions or solve a post-check write race.

## Actual Inno execution probe

Run on a disposable administrator Windows runner with no existing ImmichClone
service/application/installer registration:

```powershell
./scripts/Test-UpgradePreparation.ps1 -Binary "$env:RUNNER_TEMP/security-app.exe"
```

The script may receive `-Compiler <path-to-ISCC.exe>` when the existing pinned
compiler was provisioned by the workflow. Otherwise it acquires the same Inno
6.7.4 setup already selected in `Build-Installer.ps1`, checking SHA256
`a549929319f9120716677c685535ca4671e4b937b5fdc87e2585f32d8eb30e8c`
before unattended provisioning. It does not add an end-user prerequisite.

The probe copies and compiles the production `.iss`, without replacing its Pascal
logic or injecting a test-only preparation hook. It uses a current executable and
minimal synthetic provider placeholders in a disposable `0.0.0` package. This
package must never be uploaded or delivered as a product installer. It does not
download/build native providers or claim their packaging/redistribution acceptance.
The fixture path includes a space and Unicode character.

Required observations from execution:

1. Setup presented with a junction state path rejects during bundled preparation.
   A retained old helper that would create a harmless marker is not executed.
   The retained executable/helper bytes remain unchanged, no new sibling helper
   is installed, target ACL/bytes remain intact, and no service/state registration
   is published.
2. The test constructs a real existing LocalService and library, with the current
   executable and a deliberately retained marker preparer. An authenticated upload
   establishes an immutable original before Setup starts.
3. Actual Setup extracts/executes the new bundled preparer, snapshots the real prior
   executable/catalog/access code into the real rollback directory, installs the
   candidate helper bytes, and returns a healthy LocalService. The retained marker
   must remain absent. Managed original/access-code bytes and data location remain.
4. The disposable installer uninstalls successfully and retains original bytes.
   `finally` removes only the owned fixture service, uninstall registration/files
   and junction before deleting its temporary tree. No raw Setup log or credential
   is printed or uploaded; only fixed assertions are emitted.

This tests more than compilation or hand-copied extraction logic. It still uses a
current-runtime existing-service fixture: it is not the actual accepted old
0.1.29 package, not a cross-provider upgrade, and not a full failed-upgrade or
independent-restore test.

## Focused Windows acceptance

[Run 41525562509](https://github.com/aataat12/immich-clone/actions/runs/41525562509),
source `bb602630fe1e49f905b51635ab174b97264be54c`, job `125552255121` passed
on Windows Server 2022. The primary agent inspected the successful job evidence.
The pinned Inno 6.7.4 setup checksum/provisioning and compilation of the production
`.iss` passed, followed by actual Setup execution rather than compilation alone.

The junction case rejected before replacement application files, service or state
registration were published. The retained preparer marker was absent; retained
app bytes and outside target ACL/bytes remained unchanged. The normal existing
LocalService/library fixture completed Setup without executing its retained marker
preparer. Real prior binary/catalog/access-code rollback snapshots existed, exact
candidate management/helper bytes were installed, original/code/data location
were preserved, and the LocalService returned healthy. Actual uninstall succeeded
and retained the managed original bytes.

This clears the focused staged-preparer execution gate only. The current-runtime
fixture/minimal provider limitations above still apply. No product installer,
actual accepted older-installer transition, full-provider upgrade or complete
failed-upgrade/restore acceptance is claimed by this probe.

## Acceptance still required
- Prove the staging directory and extracted helpers are not writable by a separate
  standard-user token. The SHA256 match alone is not that evidence.
- Build the broader real candidate from its pinned payloads, upgrade the actual
  accepted prior package, inject startup/configuration failure, and exercise
  rollback, independent restore and uninstall retention. Include missing/corrupt
  staged-helper failure controls where practical without adding production bypasses.
- Keep permissive custom AppDir trust, explicit/protected child ACLs and concurrent
  reparse replacement work open. No broad recursive ACL migration was added.
- Maintain separate Windows 10/clean-machine/signing/reboot gates; Server success
  cannot certify the target system.

Primary implementation references checked on 2026-11-02:

- [Inno ExtractTemporaryFile](https://jrsoftware.org/ishelp/topic_isxfunc_extracttemporaryfile.htm)
- [Inno preprocessor GetSHA256OfFile](https://jrsoftware.org/ishelp/topic_getsha256offile.htm)
- [Inno Pascal GetSHA256OfFile](https://jrsoftware.org/ishelp/topic_isxfunc_getsha256offile.htm)
