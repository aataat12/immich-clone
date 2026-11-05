## Milestone 7 authorization — November 4, 2026 (America/Chicago)

The user explicitly approved resuming the five original open items SEC-02,
SEC-03, SEC-05, SEC-06 and SEC-08 for milestone 7. This supersedes their Group 2
work deferral, not their classification or closure criteria. SEC-01/04/07 remain
closed; Group 3 investigations and paid signing remain deferred. No parallel
workers or new charges are authorized. Final target acceptance remains distinct.

# Fixed security worklist

These are the original eight items. Keep these IDs and titles stable. New findings
belong under the relevant item; genuinely additional work gets a new explicit ID,
never a replacement list. A diagnostic or source review alone cannot close an item.
Security scope progress: **5 closed, 3 open**. SEC-06 closed in the validated
unpublished input-hardening candidate; see sec06-input-hardening.md. SEC-02 closed in the validated
unpublished milestone7 candidate; see installer-trust-milestone7.md. Original SEC-04 protection remains
implemented/validated; the later normal-restore observation is separately recorded
as deferred Group 3, not represented as a demonstrated security-limit failure.
All original items are currently classified Group 2 under security-priority-groups.md;
paused does not mean fixed. Only original items may be in Groups 1 or 2.

Closed means the stated protection is implemented/enforced, negative and normal
controls passed, and evidence was reviewed. For SEC-07, the complete review and
scan controls are finished with every known finding explicitly owned; remediation
of those findings is not implied. Release delivery is separately item 8.

| ID | Original item | Status | Owner / completion requirement |
|---|---|---|---|
| SEC-01 | Child-folder and file permissions | Closed — implemented and Windows-validated | Inspect every application-owned state/log/rollback descendant and root ownership/DACL policy; unsafe explicit/protected/inherited configurations or unreadable ACL/enumeration must fail closed; never follow child links. Prove ordinary-user denial on correct trees, read-only refusal before mutations and final verification before service/content changes, unchanged originals and outside targets, and usable LocalService. No silent recursive permission/ownership rewrite. |
| SEC-02 | Safe upgrades | Closed — implemented and Windows-validated candidate | Actual staged-helper token/sharing denial, unsafe static code-tree refusal, protected custom/Program Files Setup and genuine 0.1.39-to-candidate/full-provider/injected rollback passed. See installer-trust-milestone7.md. Concurrent replacement remains SEC-03; final target/release gates remain SEC-08. |
| SEC-03 | Filesystem boundaries | Open | Protect against concurrent junction/reparse replacement during privileged operations; static preflight does not close this item. |
| SEC-04 | Hostile backup limits | Closed — security protections validated; Group 2 further mitigation paused | Aggregate/catalog/process protections remain enforced and passed hostile/valid controls. R-RESTORE-01 is separate deferred Group 3 reliability evidence; no root-cause fix is claimed. |
| SEC-05 | Native worker isolation | Open | Enforce required-input/scratch-only authority; current Windows canaries confirm same-token secret/catalog/original access remains possible despite working resource limits. A probe is not a fix. |
| SEC-06 | Remaining input review | Closed — implemented and Windows-validated candidate | Malformed media/filename/metadata/tag/catalog and parent Go review/mitigation passed negative and valid controls, actual native/compatibility/restore checks and measured parent decoding. See sec06-input-hardening.md. NR-01 patched-provider evidence retained; final combined installer/target acceptance remains SEC-08. |
| SEC-07 | Review closeout | Closed — complete review and executed controls | Complete threat-model/applicability checklist, dependency/secret-scan coverage record, residual-risk decisions and update/recheck procedure. |
| SEC-08 | Release validation | Open | Validate the combined real installer against actual prior package, injected failure/rollback, full providers, backup/restore/browser and applicable Windows 10 gates. |

SEC-01's stationary permission policy does not claim resistance to a concurrently
swapping privileged actor (SEC-03) or a compromised same-identity native child
(SEC-05). Refusing a preexisting unsafe tree does not retroactively revoke its old
permissions: it prevents configuration from being accepted as secure and requires
explicit administrator remediation before proceeding. No private inventory or
raw ACLs are published. General Windows 10 compatibility/signing/endurance claims
remain evidence-gated regardless of any single closed item.


## SEC-01 closure evidence

The original child-folder/file-permission item is closed in the candidate source:
[41534740456](https://github.com/aataat12/immich-clone/actions/runs/41534740456),
source 81b0f96a51548926c46e4098cbc5ec518c0bdb65, job 125583595153.
The full stationary-tree matrix, actual ordinary-user negative/positive controls,
LocalService/Setup/original/rollback preservation and uninstall controls passed.
All final inherited permissions are verified after safe root normalization.
See security-permissions-completion.md and security-permissions-tests.md.
That run later failed a separate new photo-dialog asynchronous-cleanup assertion;
it is not claimed as an all-green job. A corrected UI-wait rerun subsequently passed in full as recorded below.
At that checkpoint SEC-02 through SEC-08 remained open. No installer release was implied.

Final all-green acceptance: [41535508518](https://github.com/aataat12/immich-clone/actions/runs/41535508518),
source 77d1f1b87411b0ed51834bd054ef2603c616ea0f, job 125586209991. The unchanged security implementation
passed again: 21 unsafe configurations rejected, real-user positive/negative
controls, final inherited-rights checks, LocalService/Setup/byte preservation,
Go tests/vet, native-authority canary and all browser checks. SEC-01 is closed;
At that checkpoint SEC-02..SEC-08 remained open. No release installer or Windows 10 certification.


## SEC-04 and SEC-07 closure evidence

Four authorized workers paired implementation and independent verification for
these two complete original items. Root reviewed source, controls and actual
Windows results: all-green [41547447887](https://github.com/aataat12/immich-clone/actions/runs/41547447887), source
1f1cdb912b223634dec0350e58e9ae86b11893b0, job 125626387613.

SEC-04: whole archive/catalog work executes under the production supervised
worker; aggregate writer/metadata/count/catalog budgets, hard wall/Windows
committed-memory bounds, reaping and failure cleanup passed hostile and valid
controls. Actual CLI default/explicit schema-1 restoration preserved originals,
ready AI results, names/undo/dismissal/tags and access code; seven low-budget
refusals cleaned staging without publication. The 512MiB Job allowed the 32MiB
positive helper and refused 600MiB allocation. Catalog growth monitoring has
latency and is not an atomic disk quota. See security-restore-limits.md and
security-restore-limits-tests.md.

SEC-07: complete threat/OWASP applicability, compiled/transitive native inventory,
current published advisory dispositions, scan/control coverage, all owned residual
findings and repeatable update/recheck gates were independently reviewed. All 18
scanner controls and the current Windows-target Go scan passed. Source/history
scan of declared local refs/current files found zero supported signatures; its
explicit exclusions remain documented. See security-review-closeout.md and
security-review-closeout-verification.md.

The completed review found NR-01, a present OpenCV NaryEltwise defect and compiled
core safety fixes requiring provider remediation and actual validation. This is
an explicit SEC-06/08 release stop, not accepted risk or unfinished review moved
out to obtain closure. Known filesystem races, upgrade trust/recovery, native
authority and broad input/release gates remain open under their original IDs.
Only SEC-01/04/07 are closed. No new installer or Windows 10 compatibility claim.


## 2026-11-03 superseding status

SEC-04 is reopened because 41553917677 observed a valid-restore refusal, with no
explained cause. The original item/title has not moved. Bounded extraction and
processing controls are still enforced, and later repeated/full tests passed;
neither fact is presented as a root-cause fix. restore-revalidation.md records
R-RESTORE-01 and its closure gate. Only SEC-01/07 currently remain closed.

NR-01's provider defects are remediated in the held candidate and actual native
plus application execution passed; see opencv-nr01.md. Broader SEC-06 input review
and SEC-08 actual combined installer/provider upgrade/rollback/Windows 10 gates
remain open. No new release or local installation is implied.


## User classification clarification — November 2, America/Chicago

This supersedes the earlier combined security/reliability reopening status: Groups
1/2 contain only the original eight items, all currently Group 2 given the recorded
loopback/authenticated local-input deployment; no Group 1 exploit is demonstrated.
R-RESTORE-01 is deferred in Group 3, with its failure and unknown cause preserved.
Original SEC-04's security controls are not shown as failed or undone by that
normal restore refusal. No extra security work on Groups 2/3; 6A implementation
continues. This is not an assertion that normal restoration can never fail.


## Current combined acceptance and original-item status

[Combined run 41802642179](https://github.com/aataat12/immich-clone/actions/runs/41802642179),
source `1c9d5e8329759b6c26db8c63f8dea733066ac74b`, passed actual prior-release
upgrade, both failure/rollback paths, full providers, independent restore/reinstall
and preservation. Candidate ZIP/installer checksums were independently verified.
[Exact receipts and limits](installer-trust-milestone7.md#combined-candidate-acceptance-and-sec-02-closure).
SEC-02 is closed in the validated unpublished candidate; SEC-03/05/06/08 remain
open and authorized. Current count: **4 closed / 4 open**. Milestone7 remains in
progress; next implementation gate is SEC-03. 0.1.39 remains the published prototype.
