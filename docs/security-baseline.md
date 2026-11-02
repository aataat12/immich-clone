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

## S-004: Go dependency check (2026-11-02)

Actual cloud Linux source analysis targeting GOOS=windows, GOARCH=amd64,
CGO_ENABLED=0, Go 1.27.3: govulncheck v1.9.0, source/symbol mode against
./cmd/immichclone, database https://vuln.go.dev last modified
2026-10-27T21:16:18Z, zero findings. The JSON stream included 203 advisory records;
those are not 203 application vulnerabilities. Tool acquisition uses official
Go module checksum verification; it is a build/test tool, not a product dependency.
scripts/Test-GoSecurity.ps1 repeats this check on the Windows acceptance runner
and fails the job on scanner/provisioning/findings errors. This does not cover
C/C++, models, installer tooling, unknown vulnerabilities or excluded build tags.

Source: https://go.dev/doc/security/vuln/ (reviewed 2026-11-02).

## Native dependency review — selective, remaining work explicit

| Component | Checked evidence and present conclusion | Remaining gate |
|---|---|---|
| FFmpeg 8.1.3 | Upstream download page still lists 8.1.3 for the 8.1 branch. The pinned build disables network and all components except the documented decoder/demuxer/filter allowlist. HLS/TLS/TIFF paths are excluded. This narrows applicability, not a blanket security clearance. | Complete advisory-to-source applicability inventory for enabled MOV/Matroska/H264/HEVC/AAC/filter code; maintain replacement/retest procedure. |
| OpenCV 4.12.0 | Upstream CVE-2025-53644 disclosure identifies 4.12.0 as fixed; this worker also omits imgcodecs. Fixed weights and bounded BGR input are reused. Its security page is a reporting policy, not a support SLA or clean scan. | Wider current advisory review, runtime servicing and stronger token/filesystem isolation assessment. |
| Embedded protobuf 3.19.1 | Retain the previously verified CVE-2022-1941 parser backport. GHSA-h5g9-ghrj-76p5 says the JSON defect was introduced in 4.22.0; 3.19.1 predates it and the worker uses ONNX binary parsing. PHP/Python/Java advisories do not directly describe this C++ path. | Do not treat a single backport as comprehensive servicing; keep exact source changes and re-evaluate on provider changes. |
| Embedded zlib 1.3.1 | Upstream 1.3.2 includes audit-related changes. This is an open review item, not silently cleared because the worker tests pass. No native binary replacement occurred in this slice. | Determine compiled/reachable affected functions and whether update/backport or removal is appropriate; rebuild/retest provider if required. |
| Go modules and native providers | Existing exact versions, checksums, redistribution notices and replacement boundaries are retained in dependency-evidence.md and preparation scripts. | Signed release arrangements and static CRT servicing remain production gates. |

Primary sources reviewed 2026-11-02:
- https://www.ffmpeg.org/download.html
- https://github.com/FFmpeg/web/blob/master/src/security
- https://github.com/opencv/opencv/security
- https://securitylab.github.com/advisories/GHSL-2025-057_OpenCV/
- https://github.com/protocolbuffers/protobuf/security/advisories
- https://github.com/protocolbuffers/protobuf/security/advisories/GHSA-h5g9-ghrj-76p5
- https://zlib.net/

## OWASP Top 10:2025 applicability checkpoint

This is a review map, not certification or ten passed audits. Source:
https://top10.owasp.org/2025/ (reviewed 2026-11-02).

| Area | Project-specific evidence and remaining review |
|---|---|
| A01 access control | Common HTTP guard matrix passed; Windows ACL/reparse and handler-specific checks remain. |
| A02 configuration | Loopback binding, Host/Origin and CSP reviewed; installed configuration remains in acceptance. |
| A03 supply chain | Pinned modules/providers and Go scan; native/transitive applicability remains open. |
| A04 cryptography | Random access/session tokens and SHA256 integrity; private backups are not encrypted. Preserve documented local-only boundary; do not introduce an incompatible backup change silently. |
| A05 injection | Text rendering, bounded typed request handling and CSP; broader malicious metadata/path and stored-input tests remain. bbolt uses no SQL. |
| A06 design | Explicit trust boundaries, managed immutable copies and reviewed AI suggestions; abuse/concurrency and worker isolation remain open. |
| A07 authentication | Code-cookie sessions and expiry/forgery tests; brute-force/session-cap coverage remains to review. |
| A08 integrity | Pinned hashes, atomic catalog operations and independent restore tests; artifact privacy/integrity review remains. |
| A09 logging | Existing allowlisted private diagnostic records; new artifact/canary checks and retention review remain. |
| A10 exceptional conditions | Existing cancellation, restart, ENOSPC and rollback tests; extended/disconnected-storage abuse remains open. |

## S-005: stalled login blocks authenticated requests (fixed candidate)

Confirmed by a regression that fails against the preceding source: login held the
shared session mutex while decoding an unauthenticated body. A stalled request
therefore blocked existing-session authentication. Severity: medium availability
for a local client able to issue the required request header; no data-exposure or
remote exploit claim. Browser cross-origin/header restrictions remain relevant.

Reserve/rate-limit attempts under the mutex, release it before reading, then
acquire it only to publish a validated session. Enforce a 1024-byte login body,
10-second read deadline on the real HTTP connection, strict fields and one JSON
value; advertise Retry-After on attempt throttling. Normal login shape, code
storage and 12-hour session policy stay unchanged. Existing MaxBytesReader and
ResponseController are used; no new dependency. Deadline bounds a login only,
not media uploads. A synthetic reader verifies mutex independence; a separate real-socket
HTTP regression verifies the 10-second deadline and connection close. A Linux
standalone-app socket probe also passed with authenticated reads responsive.

Cloud Linux focused web race/vet and all internal-package tests passed, including
valid login, incomplete-body concurrency and extra/oversized/unknown-field JSON
rejection. Full Windows acceptance passed with this fix: run 41469530044, source
19b049ffeb7902fe4509a0c50a83b3c754901c8d, job 125359889950. The socket probe initially
found that clearing a failed read deadline allowed response cleanup to stall;
failed-login connections now close without clearing that deadline. Earlier Windows
acceptance caught a test hashing raw CRLF rather than HTML-normalized text; that
expectation was corrected before the rerun, without changing CSP behavior. No
failed-run installer is published. No project-plan conflict identified.

## 0.1.27 candidate package acceptance

[Windows Server 2022 acceptance 41469530044](https://github.com/aataat12/immich-clone/actions/runs/41469530044)
passed source 19b049ffeb7902fe4509a0c50a83b3c754901c8d, job 125359889950.
Actual Windows stalled-login socket test, request guards, source dependency scan,
LocalService/Edge media/AI/People/bulk/import/paging/backup flows, independent
restoration, real NTFS ENOSPC, actual checksum-verified 0.1.25 installer upgrade,
injected 0.1.28 rollback, restored reinstall and uninstall preservation passed.
The exact previously proven native providers/models were reused, not rebuilt.

Artifact 13153699093 retrieved: ZIP SHA256
2de7c9374df99ea0fb0c698236b64de8cd0afc4f955a04b3e804766fc7191d65.
Installer ImmichClone-0.1.27-prototype-x64-setup.exe, 102607252 bytes, SHA256
cffb19c052ac17cac30c62f845225cc6e30868f8d14389dc9e7ffd75876312ff,
matched its packaged manifest. Browser artifact 13154618150 ZIP SHA256
 ec567c2e9efdd3c4d1a7b5cba26bec9c4f6b4b9d927325fe02b08844cd8d23ca
verified; Build Notes desktop/narrow screenshots inspected. Narrow table remains
wrapped; no horizontal overflow in browser assertions. No full UI redesign.
Unsigned candidate, seven-day artifacts. No new local test/reboot requested.
Broader security/native servicing work above remains open; milestone 2 is not
marked delivered. No full Windows 10 or production security certification.


## Pending next release: explicit session controls

User-reported 0.1.27 functional checks passed; sign-in/out checks were not testable
with the saved-session UI. Add visible Sign out to all pages; /login explicitly
shows the code form; /logout is a non-mutating confirmation page. Only the
same-origin/header-protected POST /api/logout revokes the current server token
and expires its HttpOnly/SameSite cookie. Replaying that token must fail while an
independent session remains valid. Idempotent sign-out supports an expired session.
No code is put in URLs, browser storage, logs or artifacts. Existing Start-menu
View access code helper stays available. Leaving an active upload may interrupt
that transfer; accepted originals remain saved. No plan conflict identified.

Linux race/vet tests pass for route availability, GET non-mutation, token replay
rejection, unrelated-session preservation and cookie clearing. The focused Edge
probe covers logout confirmation, cross-origin rejection, incorrect code, re-login
and reload persistence. Bundle in the next broader release; do not publish a
login-only installer. Windows/browser and final package acceptance remain gates.

### zlib reachability follow-up

Reused the retained exact native build log and opencv-module-sources.zip; no native
rebuild. The log confirms zlib is compiled and core persistence is present, so it
cannot be dismissed merely because compressed-image codecs are disabled. Textual
review of 1,837 archived module-source files found gzopen in core persistence;
no direct inflateCopy, deflateCopy or crc32_combine calls matched in those module
sources. This is a textual search, not a complete binary call-graph proof.

The worker itself reads only three fixed ONNX byte buffers and bounded BGR frames,
uses the ONNX importer directly, and does not expose FileStorage or arbitrary
model/image loading. The Go wrapper verifies provider/model hashes. These are the
current mitigation boundaries; do not expand accepted inputs without repeating
the native review. Full transitive reachability/static-runtime servicing remains
open. No user media, dynamic model or original storage change was introduced.
