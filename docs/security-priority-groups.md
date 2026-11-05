## Milestone 7 authorization — November 4, 2026 (America/Chicago)

The user explicitly approved resuming the five original open items SEC-02,
SEC-03, SEC-05, SEC-06 and SEC-08 for milestone 7. This supersedes their Group 2
work deferral, not their classification or closure criteria. SEC-01/04/07 remain
closed; Group 3 investigations and paid signing remain deferred. No parallel
workers or new charges are authorized. Final target acceptance remains distinct.

# User security priorities — November 2, 2026 (America/Chicago)

User authorizes continued Group 1 mitigation alongside milestone 6A. Pause Group 2
security mitigation; retain its findings and existing protections, without marking
it fixed or accepting unknown risk. This supersedes the former requirement that
all security backlog work finish before 6A development.

Groups 1 and 2 classify ONLY original SEC-01..SEC-08. Any newly identified
issue goes in Group 3 for recording only, with no investigation or mitigation
time. Security work time is authorized only for Group 1; milestone 6A functional
implementation and its normal correctness/data-preservation tests continue.

Group 1: an attack surface created by the installed/running app or its installed
support components which a remote Internet attacker can exploit by reaching the
machine. Include a malicious website reaching the loopback service through the
browser, if it can bypass the existing Host/Origin/session/request safeguards.
Group 2: other security findings, including local account/file/installer attacks
and crafted files which require deliberate local import/restore in this deployment.
A finding's group changes if a demonstrated remote reachability path is found.

Current evidence: app startup requires an IP loopback binding; default127.0.0.1:2283.
Repository inspection found no firewall/UPnP/public listener configuration. FFmpeg
is network-disabled and its calls permit only file/pipe. AI has no network listener.
Host/session/Origin/CSRF and CSP negative checks have actual Windows/browser evidence.
No open Group 1 exploit is demonstrated by the existing worklist. This is scoped
source/disposable-runner evidence, not a scan of the user's router/PC or a guarantee
against unknown vulnerabilities. No configuration/exposure expansion is authorized.

| Original item | Classification | Action |
|---|---|---|
| SEC-01 Child-folder and file permissions | Group 2 | Existing completed protection retained; no further mitigation work. |
| SEC-02 Safe upgrades | Group 2 | Closed in validated unpublished milestone7 candidate; retain controls. See installer-trust-milestone7.md. |
| SEC-03 Filesystem boundaries | Group 2 | Closed in the validated unpublished milestone7 candidate; focused concurrency and final genuine prior-release/provider/recovery gates passed. See sec03-combined-acceptance.md. |
| SEC-04 Hostile backup limits | Group 2 | Existing validated resource protections retained; no further mitigation work. |
| SEC-05 Native worker isolation | Group 2 | Active in milestone7. Local/imported-media worker authority remains open. |
| SEC-06 Remaining input review | Group 2 | Active in milestone7 under current authenticated local-import/catalog deployment. No demonstrated remote bypass. |
| SEC-07 Review closeout | Group 2 | Existing completed review retained; no broad new review. |
| SEC-08 Release validation | Group 2 | Active in milestone7; candidate combined checks passed, final target/release gates remain open. |

**Group 1 currently has no demonstrated open item from the original eight.**
This is a reachability classification for the current deployment, not a claim
that the app is invulnerable or that these items are all complete. Do not
invent or broaden an attack path merely to keep security work active.

## Group 3 — record only, no further work

| Previously identified additional issue | Recorded disposition |
|---|---|
| R-RESTORE-01 unexplained normal restore refusal | Separate reliability observation, not evidence that original SEC-04 security limits failed. Subsequent tests passed; cause unconfirmed. Deferred; no investigation now. |
| NR-01 OpenCV defects and accompanying mean correction | Already patched and native/application-validated in the held candidate. Retain existing work/evidence; no new mitigation work. |
| GitHub Actions embedded Node 24 exact patch version | Not established by existing evidence. Browser-test CLI22.23.4 was recorded. Defer further platform-runtime investigation. |

New 6A routes must retain common authorization, exact-origin mutation protection,
loopback binding and safe browser output; rerun applicable negative checks. Normal
feature bounds, cancellation, immutable originals, schema/backup correctness and
ordinary recovery remain engineering requirements, not a reason to restart the
paused exhaustive Group 2 mitigation backlog. No claim that Group 2 issues are fixed.


## Current original-item status — November 4

SEC-02/03 now join SEC-01/04/07 as closed after focused and combined genuine
prior-release/full-provider/rollback/independent-recovery acceptance. See
installer-trust-milestone7.md and sec03-combined-acceptance.md for exact scope.
Count is5 closed/3 open. SEC-05/06/08 remain open and authorized for milestone7. The earlier
November2 pause instructions above are superseded for those original open items by
the explicit November4 authorization at the top; Group3 investigations and paid
signing remain deferred. All original items retain their Group2 classification.
