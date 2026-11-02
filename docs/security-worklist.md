# Fixed security worklist

These are the original eight items. Keep these IDs and titles stable. New findings
belong under the relevant item; genuinely additional work gets a new explicit ID,
never a replacement list. A diagnostic or source review alone cannot close an item.
Closed means the stated protection is implemented/enforced, negative and normal
controls passed, and evidence was reviewed. Release delivery is separately item 8.

| ID | Original item | Status | Owner / completion requirement |
|---|---|---|---|
| SEC-01 | Child-folder and file permissions | Open — assigned implementation and independent verification | Inspect every application-owned state/log/rollback descendant and root ownership/DACL policy; unsafe explicit/protected/inherited configurations or unreadable ACL/enumeration must fail closed; never follow child links. Prove ordinary-user denial on correct trees, unsafe-tree refusal before service/ACL/state changes, unchanged originals and outside targets, and usable LocalService. No silent recursive permission/ownership rewrite. |
| SEC-02 | Safe upgrades | Open | Bundled preparer is implemented/focused-validated; staging token permissions, unsafe custom AppDir trust and actual previous-installer failed-upgrade recovery remain. |
| SEC-03 | Filesystem boundaries | Open | Protect against concurrent junction/reparse replacement during privileged operations; static preflight does not close this item. |
| SEC-04 | Hostile backup limits | Open | Bound aggregate extraction and hostile catalog-processing work beyond per-file limits, with failure cleanup and valid independent restore. |
| SEC-05 | Native worker isolation | Open | Enforce required-input/scratch-only authority; current Windows canaries confirm same-token secret/catalog/original access remains possible despite working resource limits. A probe is not a fix. |
| SEC-06 | Remaining input review | Open | Complete malformed media/filename/metadata/tag/catalog review and applicable negative tests; parent Go decoding is included. |
| SEC-07 | Review closeout | Open | Complete threat-model/applicability checklist, dependency/secret-scan coverage record, residual-risk decisions and update/recheck procedure. |
| SEC-08 | Release validation | Open | Validate the combined real installer against actual prior package, injected failure/rollback, full providers, backup/restore/browser and applicable Windows 10 gates. |

SEC-01's stationary permission policy does not claim resistance to a concurrently
swapping privileged actor (SEC-03) or a compromised same-identity native child
(SEC-05). Refusing a preexisting unsafe tree does not retroactively revoke its old
permissions: it prevents configuration from being accepted as secure and requires
explicit administrator remediation before proceeding. No private inventory or
raw ACLs are published. General Windows 10 compatibility/signing/endurance claims
remain evidence-gated regardless of any single closed item.
