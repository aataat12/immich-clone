## November 5, 2026: 0.1.42 release candidate

Based on exact published 0.1.41 source e1212539511ee3413a7049e318dc0c2699e8ee0c. SEC-06 source/tests integrated from 8d3fa47fc5d2b2ff4fd5726da6a43e4f35eb852b without replacing later SEC-03 recovery controls. SEC-05 experiments are excluded. All five workstream owners confirmed the handoff; SEC-08 sanitized target notes are included, private receipts excluded.

Security closure becomes 6 closed / 2 open only after combined candidate acceptance: SEC-01/02/03/04/06/07 closed; SEC-05 paused/open and SEC-08 open. All original items remain Group 2; Group 1 has no demonstrated open original item; Group 3 stays record-only/deferred. Milestones 7 and 8 remain in progress. Signing remains deferred. Genuine 0.1.41 upgrade, injected rollback, full providers, browser, input regression and independent recovery must pass before publishing this prototype. Clean-machine Windows 10, reboot, endurance and final candidate target acceptance remain open.

# Milestone 7: installer, security and update hardening

## Milestone 7 authorization — November 4, 2026 (America/Chicago)

The user explicitly approved resuming the five original open items SEC-02,
SEC-03, SEC-05, SEC-06 and SEC-08 for milestone 7. This supersedes their Group 2
work deferral, not their classification or closure criteria. SEC-01/04/07 remain
closed; Group 3 investigations and paid signing remain deferred. No parallel
workers or new charges are authorized. Final target acceptance remains distinct.


Started November 4, 2026 (America/Chicago), after the user accepted the exercised
combined 0.1.38/0.1.39 functional checks and instructed proceeding with milestone 7.
This authorizes milestone work; it does not silently revoke the explicit Group 2/3
security deferrals. No workers or new charges are authorized.

## Completion gates and current evidence

| Gate | Evidence / remaining work | Disposition |
|---|---|---|
| Accepted starting point | Published 0.1.39 source `1be1d013529635e96043549c7cb80c0f76d38a3a`; combined run [41784610044](https://github.com/aataat12/immich-clone/actions/runs/41784610044). November 4 read-only GitHub check reconfirmed completed/success, matching release target and installer asset digest/size. | Existing evidence reused; no rerun or new package. |
| Local functional regression | [Combined user result](validation-0.1.39.md#combined-user-result). All exercised checks passed; omitted checks remain NOT TESTED. | Recorded; final target acceptance remains separate. |
| Dependency provenance and servicing | Existing [dependency evidence](dependency-evidence.md), preparation scripts and [review closeout](security-review-closeout.md) retain exact source, models, hashes, licenses and static-runtime replacement obligations. The accepted run checked Go dependencies. | Reused dated evidence, not a fresh advisory audit. Broad renewed review remains deferred. |
| Diagnostic privacy | Existing fixed-schema logging, offline export and installed synthetic access-code/media canary checks passed in the accepted combined run. `internal/diagnostics/log.go`, `packaging/Export-Diagnostics.ps1` and `scripts/Test-Installer.ps1` were inspected for this inventory. | Existing bounded evidence retained; no claim of exhaustive privacy certification or new security closure. |
| Upgrade and rollback correctness | Actual 0.1.38 installer, injected initial failure, successful 0.1.39 upgrade, injected 0.1.40 rollback, independent restore/reinstall and uninstall preservation passed. [Acceptance](release-0.1.39-acceptance.md). | Functional correctness evidence exists; it does not close SEC-02 privilege/trust boundaries. |
| Update procedure | [Manual update guide](update-guide.md) records verification, backup, same-folder upgrade and failure handling. No automatic updater is introduced. | Documentation prepared; next installer must pass acceptance for its actual source and previous release. |
| Signing arrangements | Current release is unsigned. No certificate, signing account, identity validation or timestamp service has been configured or verified. Existing unsigned prototypes may continue within the user's prior bounded approval. | Final signing decision is open. No purchase or trusted-publisher claim. |
| Safe upgrades: SEC-02 | Staged helper token/sharing and static code trust controls plus genuine prior-release/full-provider/rollback passed. See installer-trust-milestone7.md. | Closed in validated unpublished candidate; SEC-03 races and SEC-08 final gates remain separate. |
| Filesystem boundaries: SEC-03 | Held path/identity/ACL, private administrative recovery, durable interruption and actual Setup/metadata concurrency plus genuine prior-release/full-provider/independent recovery passed. | Closed in validated unpublished candidate; sec03-combined-acceptance.md. |
| Native isolation: SEC-05 | Same-token children retain service-readable authority beyond required input/scratch. | Active under explicit milestone7 authorization; implementation/evidence remains. |
| Remaining input review: SEC-06 | Full declared malformed media/filename/metadata/tag/catalog review, including parent Go decoding. | Active under explicit milestone7 authorization; review/mitigation remains. |
| Release validation: SEC-08 | Final target/clean-install/reboot and broader acceptance gates; routine prototype gates are already covered above. | Open and authorized; final target gates remain with milestone 8. |

Milestone 7 is **in progress, not delivered**. No delivered release number is assigned.
Current original security status is **5 closed / 3 open**: SEC-01/02/03/04/07 closed;
SEC-05/06/08 open. SEC-02/03 are validated in the unpublished candidate. Group 1 has no demonstrated open original item; all eight
are currently Group 2. New issues remain Group 3, record only.

## Original scope decision (now approved)

The original scope decision was explicitly approved at the top of this document.
SEC-02/03 have since passed their focused and combined gates. Next bounded work is
SEC-05, then remaining SEC-06/08 gates; no additional permission is required within
that approved scope. Group3 and signing/infrastructure purchases remain deferred.

## Preparation verification

November 4: clean starting workspace and remote head both matched acceptance-record
commit `380ad2e76504da30e87487cd59ff0da36e34ca84`. Current workflow triggers and
publication steps were inspected. Documentation-only changes do not justify a new
installer/full CI run. No application, installer, workflow, dependency, schema,
provider or shipped Build Notes changed. The installed page continues to describe
the accepted 0.1.39 build; the repository roadmap tracks subsequent work.


## SEC-02 code-trust candidate

The management helper now audits the application code tree's ACLs/owners before
logs, ACL changes, service stop or snapshots. Unrelated users and LocalService
may read/execute code but may not hold data/append/EA/attribute/delete/delete-child/
DACL/owner or generic write/all rights. Null DACLs, unsupported ACEs, untrusted
owners and reparse children are refused. Inherit-only grants are checked too.
SYSTEM, administrators and the current installing identity remain trusted; this
does not establish a boundary against the installing administrator's own account.
No automatic repair of an untrusted existing code tree is performed.

Actual Setup now invokes the extracted new helper for both Prepare and Install.
Read-only handles opened without write/delete sharing pin both extracted scripts
from before checksum verification through Setup exit. These are file-specific
sharing locks, not a claim of complete directory/ancestor race resistance.
SEC-03 remains open.

The focused Windows workflow compiles the production Inno script with explicit
nondeliverable minimal providers, runs actual Setup, proves separate standard-user
write access on unsafe root/script/executable/provider fixtures, requires refusal
before stopping a healthy service or changing catalog/rollback/code bytes, then
requires a normal protected custom upgrade and uninstall preservation. The prior
closed descendant permission suite is retained as compatibility coverage. This is
not the full real-provider/actual-prior-release SEC-08 gate. No new product version
or release is assigned. [Focused Windows execution passed](installer-trust-milestone7.md);
full original-item closure still requires its remaining gates.

First focused run [41799863387](https://github.com/aataat12/immich-clone/actions/runs/41799863387),
source `8f3b02b081058c5b4480b133564ef7b8043edb3f`, failed at Inno compilation:
the pinned 6.7.4 Pascal engine does not recognize NativeInt. PowerShell parsing and
application build passed; no installer execution/closure was established. The
Win32 Inno external handle declarations are corrected to Integer for this pinned
32-bit Setup engine; the x64 application target is unchanged.

Focused runs [41800266109](https://github.com/aataat12/immich-clone/actions/runs/41800266109)
and [41800691811](https://github.com/aataat12/immich-clone/actions/runs/41800691811)
compiled the production installer and passed static-junction refusal/preservation.
The positive fixture inherited an untrusted writer from the disposable temporary
parent and was correctly refused. The latter run called the exact code audit
directly and established that refusal before Setup. Positive fixtures now create
explicit protected trusted-writer roots; hostile controls still deliberately grant
a separate ordinary user's real write access. The production policy is unchanged.
No failed run is release/closure evidence.

Run [41801039834](https://github.com/aataat12/immich-clone/actions/runs/41801039834),
source `1b3124d9bca22617fa9a05e7a5d5795b4bf53baa`, passed real Setup, preserved
original/code/rollback snapshots, observed the real staged helper process and
proved both extracted helpers refuse elevated and standard-user write opens.
It then failed a new harness Get-FileHash on the live, exclusively locked bbolt
catalog. That read is replaced with the service's consistent logical export;
rollback snapshots and executable/original hashes remain byte comparisons. This
is not a production storage defect or a relaxed unsafe-upgrade assertion.

The code policy explicitly trusts Windows TrustedInstaller as an OS code writer,
and acceptance additionally installs/uninstalls under actual Program Files
inheritance. The prior temporary-root fixture is not presented as that evidence.

Accepted focused execution: [41801504714](https://github.com/aataat12/immich-clone/actions/runs/41801504714),
source `df25e3ee6a0d085938a38d533fc05fb4f55e5f07`, job `126478120619`. Actual
Setup/standard-user/root/script/executable/provider controls, pinned helpers,
custom/Program Files installation, preservation and existing SEC-01 compatibility
passed. Detailed evidence and explicit limits: installer-trust-milestone7.md.
All five original open items remain open until their full gates pass.

## Unpublished combined candidate gate

Prepare 0.1.41 with genuine checksum-verified 0.1.39 as the prior installer and
0.1.42 solely as the failed-upgrade injection package. 0.1.40 remains only the
previous rollback fixture. Preserve providers/models/schema/budgets. The combined
workflow's publication step is removed for this unfinished milestone: passing
artifacts are cloud acceptance candidates, not an offered release. Build Notes
marks milestone7 in progress and reflects the explicit security-resume decision.
The fixture root now has explicit protected trusted-writer permissions rather
than inheriting the disposable runner temp parent's permissive code ACLs; the
separate hostile-root and normal Program Files controls remain required evidence.


## Current combined acceptance and original-item status

[Combined run 41802642179](https://github.com/aataat12/immich-clone/actions/runs/41802642179),
source `1c9d5e8329759b6c26db8c63f8dea733066ac74b`, passed actual prior-release
upgrade, both failure/rollback paths, full providers, independent restore/reinstall
and preservation. Candidate ZIP/installer checksums were independently verified.
[Exact receipts and limits](installer-trust-milestone7.md#combined-candidate-acceptance-and-sec-02-closure).
SEC-02 is closed in the validated unpublished candidate; SEC-03/05/06/08 remain
open and authorized. Current count: **4 closed / 4 open**. Milestone7 remains in
progress; next implementation gate is SEC-03. 0.1.39 remains the published prototype.
