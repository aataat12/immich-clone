# Security baseline — work in progress

Scope: loopback-only, single-user Windows photo library; not LAN or Internet
hosting. Assets include immutable originals, access code/session cookies, names,
tags/face embeddings, catalog, backups and diagnostic exports. Trust boundaries:
browser to HTTP service; untrusted media/labels to parsers/rendering; restricted
service to local storage/native workers; build inputs to installer. A compromised
administrator or host OS is outside application containment. Do not confuse Job
Object resource limits with a filesystem/token sandbox.

User instruction: ask before implementing a security mitigation that conflicts
with the agreed project plan. No conflicting change identified in this slice.

## S-001: overly permissive inline script policy

Observed: all HTTP responses authorized arbitrary inline JavaScript via
script-src unsafe-inline. This weakens defense in depth if an HTML injection is
introduced; this review did not establish an exploitable stored-XSS path.
Priority: medium hardening; not a confirmed original-media compromise.

Candidate fix: hash the exact immutable embedded scripts; authorize those hashes
without unsafe-inline, unsafe-eval or general same-origin script authorization.
Normalize CRLF/CR to LF as HTML parsing does, avoiding a Windows-checkout failure.
Add object-src none and same-origin form-action. Inline styling remains allowed
because existing UI layout uses it; this does not authorize inline JavaScript.
Media, metadata and names still require safe rendering independently of CSP.

Evidence: cloud Linux Go 1.27.3 (official archive SHA256 checked), focused
internal/web tests with race detector and vet passed. Regression verifies all
five pages authorize bundled scripts but not modified scripts; HTML line-ending
normalization has a separate regression. Existing HTTP tests ran in that package.
No dependency/model/original-layout/backup-format change.

Browser acceptance remains pending. Local Chromium launch failed at process
singleton socket creation (Operation not permitted) before opening the app.
This is an environment limitation, not a passing browser test. A reusable
scripts/Test-SecurityBrowser.cjs checks all five pages at desktop/narrow widths,
actual script initialization and rejection of injected scripts/event attributes.
It must execute in the Windows acceptance environment before release. No new
installer published from this slice. Windows 10 remains a separate target check.

Reference: OWASP CSP Cheat Sheet, consulted 2026-11-02:
https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html
CSP is a supplementary defense, not a replacement for output escaping.

## Remaining review gates

Not completed by this slice: endpoint authorization/expired-session matrix;
resource/concurrency abuse; complete source/history and artifact secret scans;
current dependency advisories and native/static runtime servicing review; Windows
ACL/reparse-point and archive extraction attacks; full threat/OWASP applicability
mapping; installer/browser/restore regressions and privacy canaries. Existing
prior evidence is reusable, but does not silently clear these new checks.
See roadmap.md for the required mitigation/acceptance matrix. No absence-of-
vulnerabilities or full-security claim. No paid scanner or parallel agents used.
