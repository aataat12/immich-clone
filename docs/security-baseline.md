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

Focused browser acceptance passed on Windows Server 2022/Edge:
[run 41465443630](https://github.com/aataat12/immich-clone/actions/runs/41465443630),
job 125344953818. Module verification, web tests/vet and executable build passed.
The actual five-page probe verified desktop/narrow script initialization, no
page-script errors or horizontal overflow, and rejected injected script elements
and inline event attributes. This tests the standalone app with synthetic empty
state, not native inference, service-account installation or an upgraded library.
The earlier Linux Chromium socket limitation remains an environment finding;
Windows supplies actual browser evidence. No installer published. Full installer/
restore acceptance and Windows 10 evidence remain separate gates.

Reference: OWASP CSP Cheat Sheet, consulted 2026-11-02:
https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html
CSP is a supplementary defense, not a replacement for output escaping.

## Remaining review gates

Not completed by this slice: handler-specific resource validation;
resource/concurrency abuse; comprehensive source/history and artifact secret scans;
current dependency advisories and native/static runtime servicing review; Windows
ACL/reparse-point and archive extraction attacks; full threat/OWASP applicability
mapping; installer/browser/restore regressions and privacy canaries. Existing
prior evidence is reusable, but does not silently clear these new checks.
See roadmap.md for the required mitigation/acceptance matrix. No absence-of-
vulnerabilities or full-security claim. No paid scanner or parallel agents used.

## S-002: endpoint guard coverage (no bypass found in tested matrix)

Added a 35-route-family HTTP matrix covering gallery/media, faces/tags, analysis,
imports, backups, export and diagnostics. Each case rejects missing, forged and
expired sessions and foreign Host; mutation cases reject foreign/null/wrong-port
Origins and a missing CSRF header. A valid session can still read a valid gallery
query. Linux Go 1.27.3 web race/vet checks passed. The initial positive-control
fixture omitted required gallery filters (400), corrected before recording pass.
Nonexistent IDs exercise the common guard; these tests do not establish every
handler's resource-level validation or private-network protection. Loopback-only,
single-user design is retained. No authorization bypass was found in this matrix.

## S-003: limited credential-signature baseline

scripts/Check-Secrets.py checks reachable local Git blobs and nonignored working
text for GitHub-token, AWS-access-ID and private-key-header signatures, printing
only counts/category/blob IDs. Latest run: 670 text inputs, zero matches; no large/
binary exclusions encountered. This is a narrow supplemental scan, not TruffleHog
or a comprehensive secret audit. Arbitrary passwords, unknown formats, ignored
files, packaged artifacts and unreachable remote history remain outside coverage.
Added .env/.env.* ignore rules, with deliberate .env.example exception. Ignoring
a file does not remove an already tracked secret or replace credential rotation.
Synthetic positive/negative signature controls passed.
No user secrets were requested, printed or uploaded.
