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
login-only installer. Final broader-release package acceptance remains a gate.

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


Next-release authentication acceptance: focused Windows Server 2022/Edge run
[41475111595](https://github.com/aataat12/immich-clone/actions/runs/41475111595),
source e524d363dd2b42799c892309edd02f6b8ed94c3b, job 125378498262 passed HTTP tests,
vet/build, all five pages at desktop/narrow widths and actual CSP injection guards.
Browser flow passed non-mutating logout confirmation, foreign-Origin rejection,
current-session revocation and replay rejection, incorrect code feedback,
re-login, password-field clearing and reload persistence. /login can be opened
explicitly without silently ending an already valid session. The initial browser
run exposed generic 401 handling masking incorrect-code feedback; fixed before
acceptance. No installer was created for these changes. Bundle into the next
broader release per user instruction. No repeat of passed local checks requested.

## S-006: bounded original-copy admission (next-release candidate)

The store already serializes managed original copying. Concurrent authenticated
HTTP uploads could nevertheless accumulate behind that writer, holding requests
and sockets. Admit one original-copy request at a time across /api/upload and
bulk receipt uploads; reject overlap immediately with HTTP 429, Retry-After: 1
and Connection: close without consuming the rejected body. No waiting HTTP queue,
per-file size change, total library limit or throughput promise is introduced.
Normal sequential bulk import is unchanged; retry/reselect follows its existing
receipt verification. Accepted files are retained. Authentication runs first.

The blocked-copy test also found AI batch status waiting on the copy gate.
AnalysisBatch now uses its existing bbolt read snapshot plus atomic runtime
availability/import flags, without taking the copy gate. Mutating analysis
operations and provider replacement retain their existing synchronization.

Cloud Linux focused web race tests and batch/organization race tests passed.
Actual socket test rejects a second partial-body upload promptly while the first
copy is held. Gallery and AI batch status remain responsive; failure releases the
slot and a subsequent real original commits once with integrity verification.
This is not a complete resource-abuse audit. S-007 below subsequently addresses rolling transfer read-idle handling and
export/list/legacy-backup overlap. Global connection bounds, slow-response writes
and other expensive routes remain separate review items. No dependency/schema/provider change. No conflict
with the existing single-copy plan identified. Ship with a broader release,
including the already-tested login controls, not a separate control-only installer.


S-006 focused Windows acceptance: [41477493372](https://github.com/aataat12/immich-clone/actions/runs/41477493372),
source bcc0d1fbfa390ad7cf85184db76b6ade30266221, job 125386805373 passed full
library/web package tests and vet, executable build, real-socket overlap rejection,
copy failure recovery, responsive gallery/AI status, and the existing five-page
Edge session/CSP/desktop/narrow checks. Linux full library/web race tests passed.
The held-copy regression exposed the pre-existing status/copy-lock dependency;
a follow-up bounded its status assertion so regressions fail promptly rather
than waiting for the package timeout (Linux focused rerun passed). No application
code changed after the Windows run. No new installer was built. Final broader-
release upgrade/restore/import acceptance remains required before shipping.

## S-007: idle transfer and heavy-request bounds (next-release candidate)

Ordinary uploads now use the same rolling 60-second read-idle window as bulk
copies. Each read renews it; a progressing long transfer is not limited to a
60-second total duration. Both paths keep the failed read deadline and close the
connection on failure instead of clearing the deadline before response cleanup.
Successful copies clear it for connection reuse. Read-idle timeouts return 408;
failed/partial bytes never create an accepted original/receipt. The internal
short-window test setting is not an end-user configuration or a new total-size
limit. Existing 4GiB-per-file and bulk manifest bounds remain unchanged.

One shared admission slot covers legacy full-catalog listing, JSON export and
legacy complete-backup download. Overlap gets 429/Retry-After: 1 instead of
accumulating full snapshots/waiting tasks. The slot covers response writing and
releases on handler exit. Paged gallery, thumbnails, originals, managed backup
progress/control/range downloads and AI status are not included in that class.
This does not cap the memory/size of one export or every open HTTP connection.

Focused Linux race/vet tests passed for actual idle socket closure, a progressing
copy exceeding the short idle window, retry and original checksum validation.
Bulk socket timeout retains an earlier receipt, records no success for partial
input, and permits retry plus receipt replay without double counting. A held
export writer rejects overlapping heavy requests while gallery and job-status
reads remain available. Package-wide Linux web race/vet checks also passed. Windows acceptance is recorded below.
No dependency/schema/model changes, separate installer or project-plan conflict.


S-007 focused Windows acceptance: [41481008162](https://github.com/aataat12/immich-clone/actions/runs/41481008162),
source 97b6e1b2762dde297a56a8625462c91c98ae965a, job 125398974000 passed fresh
library/web package tests, vet and executable build, including real short-idle
socket rejection/recovery, a progressing transfer longer than its idle window,
bulk receipt timeout/retry/replay accounting and held-export overlap rejection.
Actual Edge session, replay rejection, incorrect-code feedback and all five-page
CSP/desktop/narrow checks also passed. The short windows exercise the same reader
and production route; the default stays 60 seconds. No installer was created.
Bundle with the held login/resource work; final broader-release install/upgrade,
restore and browser import acceptance remains required. Full security, global
connection/write-time limits, native servicing and full-library acceptance remain
open. No project-plan conflict found or local action requested.

## Native zlib replacement candidate (not integrated)

2026-11-02: advance the embedded zlib servicing item to a bounded replacement
experiment. Pin official zlib 1.3.2 tar.gz SHA256
bb329a0a2cd0274d05519d61c667c062e06990d72e125ee2dfa8de64f0119d16;
license remains the upstream zlib license. Official release page documents audit-
related changes: https://zlib.net/ . Download digest verified in cloud Linux.
A source-built static C++ operation test passed version identity, compression/
decompression roundtrip, inflateCopy continuation and malformed-input rejection.

Windows build first compiles/tests that same small library, then links the
unchanged OpenCV/model pipeline against the explicit static installation rather
than OpenCV's old bundled zlib. Retain replacement source, notice, build evidence
and actual worker-linked version; reject fallback to old bundled zlib. Existing
native workflow supplies PE import audit and subsequent actual LocalService
inference/parity/recovery/restart tests. The native build remains capped at 28
minutes, followed by a separate bounded service job. Compilation is not acceptance.

Installed app/provider hashes remain unchanged. Do not update application provider
IDs, model definitions or installer inputs until execution/packaging evidence
passes. Static CRT fixes require this kind of rebuild/retest; no automatic runtime
servicing claim. No end-user dependency installation or new installer requested.

## Package privacy checking (candidate tooling)

Add bounded credential-signature and supplied-canary checks over distribution
payloads, including nested ZIP and TAR source archives, without extracting them.
Limits: 128MiB per entry, 512MiB total scan bytes, 50000 scanned items and three archive
levels. Missing, corrupt, over-budget or linked physical inputs fail closed.
The scanner prints only counters and fixed reason codes, not paths, matched
credentials or supplied canaries. Unknown secret formats, images containing text,
semantic private content and encryption remain outside its coverage.

Six Linux test cases (including binary/ZIP/TAR canaries, nested credentials,
invalid input, budget/depth and CLI-redaction controls) passed. Actual retained ai-worker.exe and its module-
source ZIP passed: 3650 inputs/members, 54130158 checked bytes, no selected
credential signatures. Reachable source/history signature check: 736 text inputs,
zero matches. These counts are bounded scan evidence, not privacy certification.

Next installer acceptance creates a unique synthetic private media marker, scans
installed program/provider/source payloads for that marker and its runtime access
code, and fails on a match. The installer-generated uninstaller and ACL-protected
runtime rollback snapshot are deliberately outside distribution-payload scope;
rollback legitimately retains the catalog/access code and must not be deleted or
uploaded as a mitigation. Existing diagnostic canary checks remain required.
The new installed-payload check is not accepted until that broader release run
executes it. No standalone installer requested or built for scanner tooling.


Native replacement isolated acceptance: [41483644241](https://github.com/aataat12/immich-clone/actions/runs/41483644241),
source 30fc135bc0f6bd1d5af40ddaed17527caf91e219. Build job 125408169198 and
LocalService job 125411689735 passed. Exact external zlib 1.3.2 was found, static
roundtrip/inflateCopy/malformed tests passed, and worker version reports 1.3.2.
PE audit imports only kernel32.dll. Actual face/object inference, embedding parity
(minimum cosine 0.9999996401), malformed frames, missing-model/cancellation recovery
and Unicode/space-path restart passed. Peak commitment 271380480 bytes below the
512MiB job limit; input hashes unchanged. No raw embeddings/media in build output.

Artifact 13161790873 retrieved: ZIP SHA256
6bb2cbe69687501b0323f1d40fe1419109cf083e40bae219cb03f029f5986d3c.
Worker 7365120 bytes SHA256
cb35f0ccb3e38a6120a6f84fe91d17beaa719f9dcb9329c192b2c885bfb121ba
matched its manifest. Replacement source digest, license, operation evidence,
version report and PE-import record verified. Actual candidate archive privacy
scan and separate Windows privacy workflow are recorded when complete.
Application provider hash/path and installed build are still unchanged. Integration,
installer/upgrade/restore and target Windows 10 checks remain separate gates.
