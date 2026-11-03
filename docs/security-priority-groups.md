# User security priorities — November 2, 2026 (America/Chicago)

User authorizes continued Group 1 mitigation alongside milestone 6A. Pause Group 2
security mitigation; retain its findings and existing protections, without marking
it fixed or accepting unknown risk. This supersedes the former requirement that
all security backlog work finish before 6A development.

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

| Original item | Current group / action |
|---|---|
| SEC-01 permissions | Group 2; existing tested protections retained. |
| SEC-02 safe upgrades | Group 2; mitigation paused, findings retained. |
| SEC-03 filesystem races | Group 2; mitigation paused, findings retained. |
| SEC-04 hostile backups | Group 2 under authenticated local restore; mitigation paused, not closed. R-RESTORE-01 is unexplained reliability behavior, not evidence of an Internet exploit. |
| SEC-05 native isolation | Group 2 under fixed models/local media import; paused. Reclassify if a remote input route is demonstrated. |
| SEC-06 remaining input review | Local imported media findings Group 2. Any unauthenticated/browser-origin bypass reaching parsers is Group 1 and remains active. NR-01 patch already validated is retained. |
| SEC-07 review closeout | Completed scoped review retained; no repeated broad audit as substitute for functionality. |
| SEC-08 release validation | Test Group 1 controls and normal installer/data/restore/functionality when delivering. Broader Group 2 attack experiments paused. |

New 6A routes must retain common authorization, exact-origin mutation protection,
loopback binding and safe browser output; rerun applicable negative checks. Normal
feature bounds, cancellation, immutable originals, schema/backup correctness and
ordinary recovery remain engineering requirements, not a reason to restart the
paused exhaustive Group 2 mitigation backlog. No claim that Group 2 issues are fixed.
