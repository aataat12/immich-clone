## November 5, 2026: 0.1.42 release candidate

Based on exact published 0.1.41 source e1212539511ee3413a7049e318dc0c2699e8ee0c. SEC-06 source/tests integrated from 8d3fa47fc5d2b2ff4fd5726da6a43e4f35eb852b without replacing later SEC-03 recovery controls. SEC-05 experiments are excluded. All five workstream owners confirmed the handoff; SEC-08 sanitized target notes are included, private receipts excluded.

Security closure becomes 6 closed / 2 open only after combined candidate acceptance: SEC-01/02/03/04/06/07 closed; SEC-05 paused/open and SEC-08 open. All original items remain Group 2; Group 1 has no demonstrated open original item; Group 3 stays record-only/deferred. Milestones 7 and 8 remain in progress. Signing remains deferred. Genuine 0.1.41 upgrade, injected rollback, full providers, browser, input regression and independent recovery must pass before publishing this prototype. Clean-machine Windows 10, reboot, endurance and final candidate target acceptance remain open.

# Remaining milestones and Build Notes acceptance

Published prototype baseline:0.1.41; next candidate:0.1.42. Combined local regression passed all exercised checks; [untested items remain explicit](validation-0.1.39.md#combined-user-result). [Milestone 7](milestone7-hardening.md) started at the user's direction; Original open Group 2 work is explicitly resumed; Group 3 investigations remain deferred. Milestones2 and6 have delivered their current active functional scopes after combined installed acceptance. Original remaining security items remain open under milestone7; Group3 work is deferred. Milestones4/5 now deliver their bounded cloud scope; milestone7 is active and milestone8 has partial target evidence and remains incomplete. Server evidence does not establish final Windows10 acceptance. See [0.1.39 acceptance](release-0.1.39-acceptance.md).

| Order | Milestone | Completion criteria | Status | Released In |
|---|---|---|---|---|
| 1 | Bulk local AI analysis | Analyze all unprocessed eligible photos; counts/progress, pause/cancel, restart recovery, no duplicate jobs, preserve names/corrections. Include the Build Notes page and navigation specified below. | Cloud accepted — local functional checks passed | 0.1.25-prototype |
| 2 | Security baseline and People & Tags improvements | Current active scope: false-face dismissal, safe reanalysis preserving corrections, larger-library review/search performance and production-native quality evaluation. Original Group 2/3 security work is deferred by the existing priority amendment. | Delivered active functional scope — index, corrections, native-quality screen and combined installed acceptance passed; deferred security items remain open | 0.1.38-prototype |
| 3 | Hidden Media management | Managed-file location, restore, confirmed permanent managed-copy deletion, durable recovery and job/backup coordination; external sources untouched. | Delivered — cloud-tested prototype; batch selection/toggle added in 0.1.36; target acceptance pending | 0.1.34-prototype |
| 4 | Large-library import reliability | Longer mixed-media imports, interruption/retry/duplicate/disconnected-storage tests, safe resume improvements and resolution of large-file limits. | Delivered bounded cloud scope — 4 GiB boundary, mounted-volume recovery, 4,096 receipts/1,536 mixed originals, serial preview drain and combined installed acceptance; final target/physical-hardware/endurance limits remain | 0.1.39-prototype |
| 5 | Large-library backup and recovery | Large ZIP/ZIP64, capacity and throughput, interruption handling, independent restoration and practical full-library backup workflow. | Delivered bounded cloud scope — ZIP64/count, installed >4 GiB cycles, service crash/volume recovery, independent hash/metadata restore and combined installed acceptance; final target/extended soak remain | 0.1.39-prototype |
| 6 | Media compatibility and resource limits | Representative camera/video and metadata checks, malformed media and finite native resource limits; execute the declared existing formats before adding providers. RAW/HDR expansion remains deferred. | Delivered declared functional scope — focused matrix and installed/restored acceptance passed; final target Windows gate remains | 0.1.38-prototype |
| 6A | HEIC and video AI | HEIC still-image face/object analysis; bounded timestamped video-frame analysis, durable progress/cancel/restart, correction preservation and independent restore. Actual patched-provider/decoder execution before integration; detailed acceptance in heic-video-ai.md. | Delivered — cloud-tested prototype; every-frame Full Scan deferred | 0.1.30-prototype |
| 7 | Installer, security, and update hardening | Dependency/security and diagnostic-privacy review, upgrade/rollback coverage and signing arrangements; obtain approval before new charges. | In progress — existing evidence reconciled and update procedure recorded; SEC-02/03 closed; SEC-06 integrated for combined acceptance; SEC-05/08 and signing/final gates remain open | — |
| 8 | Final Windows 10 acceptance and release candidate | Clean install without development prerequisites, final reboot/service startup, upgrade/uninstall preservation, extended operation, final restore drill and documented limits/recovery. | In progress — partial existing-installation target checks; final acceptance open | — |

Milestones2 and6 active functional delivery is complete in0.1.38. User authorized concurrent milestone4 and5 development while0.1.38 local validation is pending. Milestone4 serial preview scheduling and sustained mixed-copy acceptance passed; the assigned worker completed milestone5 installed large-backup/crash/volume/independent-restore acceptance. Combined0.1.39 installer delivery passed and published; exact finite evidence and remaining final gates are linked below. No local action is requested for these cloud checks. Milestone 6A's bounded HEIC/video analysis is delivered. Every-frame Full Scan remains deferred. Milestone7 explicitly resumes original open Group2 work; new Group3 investigation/mitigation remains deferred. The historical review matrix below does not authorize expanding that scope.

Current milestone4/5 evidence: [installed import volume and sustained workload](import-installed-scale-recovery.md), [installed large backup and independent recovery](installed-backup-recovery-acceptance.md). Neither document substitutes Server results for final target OS acceptance.

Evidence: [Hidden Media 0.1.36](hidden-media-milestone.md), [import acceptance](import-boundary-acceptance.md), [installed 4 GiB browser import](installed-import-boundary-acceptance.md), [ZIP64 entry count](backup-zip64-entry-count-acceptance.md), [large backup byte volume](backup-large-volume-acceptance.md), [extraction interruption](restore-interruption-acceptance.md). A [manual independent recovery guide](recovery-guide.md) explains the existing CLI without switching the live service to a restored folder. It is reviewed documentation, not a new user-PC validation result. [In-app recovery help and per-archive instructions](backup-recovery-ui.md) shipped in 0.1.37 after combined installed acceptance. [People scale acceptance](people-index-milestone2.md) and [supported-media matrix](media-compatibility-milestone6.md) record the accepted0.1.38 milestone2/6 evidence and its finite scope.

## Required in milestone 1: Build Notes

- Add a **Build Notes** button immediately next to **People & tags** on the
  Photos & videos page. It opens an application page containing the table above,
  including completion criteria, **Status**, and **Released In** columns.
- Put the common **← Photos & videos Home** button at the top right of Build
  Notes. Both navigation controls must work on desktop and narrow layouts.
- Maintain the table as milestones progress. Mark a milestone complete only when
  its acceptance criteria are met; identify outstanding validation explicitly.
  Populate Released In with the actual release delivering the completed milestone,
  never a forecast or a version merely containing partial work. Leave undelivered
  entries as an em dash. Link evidence in repository records.
- Keep the shipped page and this roadmap aligned at release preparation. The page
  describes its installed build and does not require network access or expose
  private machine information.
- Verify navigation, headings, all nine rows, status/release rendering and narrow
  layout in browser acceptance. Confirm the delivered milestone-1 row identifies
  its actual release and the remaining rows do not claim completion.

## Boundaries

User reported 0.1.25 backup and bulk AI functional checks passed; new-import
follow-up and repeated narrow-width checks were not performed.
Previously passed local checks need no routine repetition. Reboot remains deferred
to final acceptance. Windows Server evidence does not certify Windows 10.
Milestone 6A is delivered as bounded supported HEIC AI and sampled video analysis; detailed evidence remains in heic-video-ai.md. RAW/HDR expansion and incremental/scheduled backups
remain deferred possibilities. Mobile is excluded.

Milestone 1 evidence: Windows Server 2022 run [41406051070](https://github.com/aataat12/immich-clone/actions/runs/41406051070), source d558f7c. PR #1 merged into main.


## Required in the next milestone: security baseline and mitigations

Added at the user's request on 2026-11-02 after reading
[Engadget's safety article](https://www.engadget.com/2302658/vibe-coding-safety-practices/).
The following are application-specific acceptance requirements, not findings or
claims of implemented protection. Security work starts in milestone 2 rather than
waiting for milestone 7; final installer/security acceptance remains required.

| Review area | Planned mitigation and acceptance evidence |
|---|---|
| Persistent security requirements and change review | Record a compact threat model and OWASP Top 10 applicability checklist covering browser, HTTP endpoints, local files, installers and native workers. Link each finding to severity, affected surface, fix or documented limitation, focused test and residual risk. Retain review/build history without private data. |
| Secrets and private data | Scan tracked source and reachable history for credentials; check generated logs, diagnostics, browser assets and packaged artifacts for leaks using synthetic canary secrets. Exclude local secret files; keep authentication material in protected server-side storage, never client bundles or URLs. Document rotation/revocation if exposure is found; do not publish leaked values in reports. Treat full backups as sensitive because they contain originals and the access code. |
| Access and mutation controls | Audit every media, face, tag, export, import, backup and diagnostic endpoint for authorization. Test unauthenticated requests, expired sessions, cross-origin writes, forged Host/Origin and browser-to-loopback attacks. Preserve loopback-only binding and disabled LAN/public exposure. Confirm critical decisions are enforced server-side, not merely by hidden buttons. Assess session/cookie policy and TLS requirements before any future network-access expansion. |
| Untrusted inputs and storage boundaries | Test path traversal, Windows drive/UNC paths, reparse-point escapes, malicious filenames/metadata/tags, stored script injection, malformed requests and archive extraction. Keep filenames and labels as data, escape browser output, use argument arrays instead of shell interpolation, and enforce length/type/size bounds. Audit current bbolt key/transaction handling; parameterized SQL and row-level policies apply only if a future SQL/multi-user design actually uses them. No database/framework migration solely to imitate article examples. |
| Resource abuse | Measure and bound login attempts, request bodies, upload concurrency, expensive backup/export/AI requests, queue growth and child execution. Add appropriate rate limits/backpressure with clear retry feedback; test excess requests, cancellation and recovery without stranding accepted originals or blocking normal bulk import. Use session/account limits where available plus connection/IP limits as appropriate; do not trust forwarded-IP headers by default. CAPTCHA and email/phone enrollment require a separate demonstrated need. |
| Dependency and build supply chain | Inventory exact Go/native/static-runtime/model/installer components; check official advisories and package provenance, including transitive code. Reuse pinned hashes and evidence, assess vulnerable-code reachability, and rebuild/test affected native components when necessary. Do not blindly upgrade to latest or install invented/unverified packages. Record scanner versions/date/coverage and explicit exceptions; prohibit new paid services without authorization. |
| Least privilege and isolation | Verify service/data/secret ACLs and worker access on disposable Windows fixtures. Assess native parser containment beyond existing Job Object resource bounds; document remaining filesystem/token access and a practical mitigation if stronger isolation is deferred. Treat media metadata, repository text and tool output as untrusted data, never instructions to expose secrets or execute commands. |
| Verification and continued maintenance | Run focused static/dependency/secret checks and actual HTTP/browser/Windows negative tests against the shipped configuration. Manually review sensitive changes; bounded parallel agents require explicit user authorization (four workers for SEC-04/07 were authorized and completed). Resolve exploitable high/critical findings before release or stop and explain the blocker. Document remaining findings, remediation order, dependency servicing/replacement steps and release rechecks. A clean scan is not proof of complete security. |

Completion requires a repository security record with results, environment and
coverage limits; regression tests for implemented fixes; and Build Notes updated
to reflect this expanded next milestone. Use synthetic/public fixtures only.
Preserve original hashes, names/tags, backup/restore and upgrade rollback during
hardening. No private photos, machine inventory or support logs in public evidence.
Do not request a repeat of passed local checks without a relevant change.
