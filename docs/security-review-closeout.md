# SEC-07 review closeout — 2026-11-02

Scope is the complete original **review closeout** item: current threat and
applicability map, pinned component/advisory inventory, executed scan/control
coverage, owned residual decisions and repeatable servicing procedure. It is not
a declaration that SEC-02..06/08 or every vulnerability is fixed. This record and
its [independent verification](security-review-closeout-verification.md) supersede
older selective/pending review statements only for the coverage explicitly
completed here. The coordinator owns the fixed worklist and closure decision.

Reviewed baseline `fed8e7ef57095faea5c30e46452e0130896a1964`, current candidate
source changes and immutable provider identities below. No private media, raw
support reports or credentials were used. No native rebuild was needed. This is
Windows-native, loopback-only, single-user software; LAN/public/mobile access,
arbitrary AI models and cloud inference remain unsupported. A compromised host
administrator/OS is outside containment. An unrelated ordinary user or compromised
LocalService child is **not** silently treated as that administrator.

## Assets, boundaries and attack paths

| Boundary / asset | Attacker-controlled input and control | Evidence / unresolved obligation |
|---|---|---|
| Browser → loopback service; originals, names, tags, faces and sessions | Foreign web pages, malformed HTTP, injected labels, forged Host/Origin/cookie. Startup rejects nonloopback addresses; route guard, CSRF header, same-origin checks, exact script hashes, output escaping. | S-001/002 and five-page Edge/negative-script tests; later face mutations retain the common guard. Wider stored/malformed input is SEC-06. Inline styling remains permitted; CSP supplements escaping. |
| Login/session boundary; access code and cookies | Guessed code, expired/replayed sessions, stalled login. Constant-time check, ten attempts/minute, thirty-two live sessions, twelve-hour expiry, visible sign-out/current-token revocation and bounded body/read. | S-005 and auth tests; wrong-code/replay/re-login/reload passed on Windows and user-reported 0.1.29. Cookies are local HTTP, not a TLS/LAN design. Do not expand exposure without redesign. |
| HTTP → storage/queues; availability | Overlapping copies/exports, stalled readers/writers and idle sockets. One copy and one legacy snapshot/export admission, bounded requests, rolling transfer deadlines, sixty-four sockets. | S-006/007/008 tests, including real TCP stalls/recovery. Not a local DoS guarantee. Restore aggregate/catalog work is SEC-04; parent decode work is SEC-06. |
| Installer/elevated helper → state/log/rollback tree | Explicit/protected/inherited ACEs, arbitrary owners, NULL DACLs, static links, unreadable ACLs. Full stationary preflight, root normalization and final inherited-rights verification before service/content changes. | SEC-01 closed by all-green 41535508518. Unsafe trees are refused, not silently repaired. Concurrent swaps remain SEC-03. AppDir/staging trust and prior-installer recovery remain SEC-02. |
| Service → native child; access code/catalog/original integrity | Malformed native media or compromised provider. Pinned executable/models, minimal environment/handles, absolute executable/argument arrays, Job Object memory/process/output/lifetime limits. | Actual Windows canaries prove same-token file authority persists: WI-01/SEC-05 remains a release blocker. Job Objects are not a filesystem/token/network sandbox. |
| Encoded still image → Go service | Malformed JPEG/PNG/GIF/BMP/TIFF decoded before AI BGR conversion; size/dimension/context-aware reads constrain inputs. | WI-02/SEC-06: parent allocations/CPU are outside native child limits; broader negative corpus and actual worst-case behavior remain required. |
| Selected backup → extraction/catalog | Crafted ZIP names/modes, CRC/manifest, compression/resource abuse, malicious catalog. Exclusive staged writes, canonical payload names and original/catalog integrity checks. | S-009 seventeen-case and valid restore tests passed. SEC-04 must close aggregate/catalog budgets and failure recovery; its parallel implementation is not accepted by this review alone. |
| Backup/export/diagnostic → local files or release package | Backups contain access code/catalog/private organization; logs/archives may leak content. ACLs, allowlisted diagnostics, private-canary/signature scans and no automatic upload. | Backup ZIPs are deliberately not encrypted or authenticated against deliberate replacement. Keep copies private; do not claim tamper-proof authenticity. New payload scan is SEC-08. |
| Upstream/build → installed executable/providers | Altered modules, models, helper scripts, compiler/runtime or downloaded archives. Exact hashes, module checksums, source/notices, staged helper digest checks and rollback. | Current inventory and advisories below; builder tools are not end-user dependencies. Every new binary needs immutable identity, operation evidence and distribution scan. Unsigned prototype/signing, clean target compatibility and combined release remain SEC-08. |

## OWASP Top 10:2025 applicability

This is a project-specific disposition of all ten areas, not ten security passes.
The [official index](https://top10.owasp.org/2025/) defines the categories.

| Area | Applicable controls and residual gate |
|---|---|
| A01 Broken Access Control | HTTP/session/CSRF guard and SEC-01 stationary ACL enforcement; SEC-02 privileged-code trust, SEC-03 operation-time boundaries and SEC-05 child authority remain blocked. |
| A02 Security Misconfiguration | Loopback-only startup, fixed scripts/providers, least-privilege service and explicit limits. No LAN/TLS, database framework or CAPTCHA is silently introduced. Installer configuration is revalidated under SEC-08. |
| A03 Software Supply Chain Failures | All current compiled/runtime components, signatures, license/source identities and advisory dispositions are recorded below. Native finding NR-01 is not excused by clean Go scans. |
| A04 Cryptographic Failures | Random local access/session values and SHA256 integrity checks. ZIP hashes are not origin authentication; backup confidentiality relies on ACL/private handling. No encrypted-backup or signed-installer claim. |
| A05 Injection | Argument arrays/no shell for native work, canonical paths, escaped browser data and CSP. bbolt uses no SQL, so parameterized-SQL/RLS advice is not an applicable implementation. SEC-06 retains malformed/stored-input review. |
| A06 Insecure Design | Immutable originals, independent backup/restore and review-required AI corrections. SEC-03..06 and documented restore/native servicing blockers cannot be accepted merely by this checklist. |
| A07 Authentication Failures | Rate/session/body bounds, expiry/revocation, replay and negative-code tests; local cookie model only. Network/multi-user changes require a new design and tests. |
| A08 Software or Data Integrity Failures | Pinned downloads, model hashes, staged helper hashes, atomic catalog/job updates and staged verified restore. SEC-02/03/04/08 retain code-path, race, hostile-catalog and actual release integrity gates. |
| A09 Security Logging and Alerting Failures | Fixed diagnostic categories, bounded local retention and authenticated export; no raw private paths/messages or auto-upload. Secret/privacy control failures fail the build; no claim of continuous remote monitoring. |
| A10 Mishandling of Exceptional Conditions | Cancellation/restart/ENOSPC/restore/rollback negatives exist. Unreadable permission trees fail closed; SEC-04/06/08 retain hostile resource, storage-disconnect/endurance and complete combined-release checks. |

## Current component and advisory inventory

Review date: 2026-11-02. A disposition means fixed, excluded, not reached under the
fixed worker interface, or an explicitly blocked remediation. It never means
unknown vulnerabilities do not exist. Default CMake cache options are not proof
that a module was built: accepted configure output's **To be built** list and PE
imports determine the inventory.

| Component / exact selection | Reviewed surface / primary evidence / disposition |
|---|---|
| Go 1.27.3, bbolt 1.4.3, x/image 0.47.0, x/sys 0.49.0 | Actual Windows-target source/symbol govulncheck 1.9.0, DB 2026-11-02T22:01:43Z: zero findings/exit 0; independent receipt linked below. Standard Go decoders remain SEC-06 behavior work, not certified by a dependency scan. |
| FFmpeg n8.1.3 commit 1041abdc962f4cc4f394aa8de9dc5236c0c3b9e7 | Exact enabled libraries, MOV/Matroska/image2, AAC/H264/HEVC, auto-enabled filters, file/pipe, BMP output, MinGW/OS imports and all current published master/9.0/8.1 findings are individually classified in the [independent inventory](security-review-closeout-verification.md#ffmpeg-actual-enabled-inventory-and-current-published-advisory-disposition). Relevant swscale CVE-2025-63757 and all nine 8.1/8.1.2 fixes verified ancestors; excluded codec/tool findings are not called absent globally. No external codec/network libraries. |
| OpenCV 4.12.0 commit 49486f61fb25722cbcf586b7f4320921d46fb38e | Actual compiled modules: core/imgproc/dnn/objdetect plus calib3d/features2d/flann. Static CPU SSE2, no dispatched ISA/GPU/OpenCL/IPP, imgcodecs/videoio/gapi/highgui or Python. Current vendor policy, 4.13/4.14 safety fixes and current primary reports reviewed below. **NR-01 blocks provider security acceptance**. |
| protobuf C++ 3.19.1, three-file upstream backport 0299c03005fbfe086d8394fb7a873a8a21fe327f | ONNX binary parser compiled in. All seven vendor published advisories classified below. CVE-2022-1941 backport is present; this is not an arbitrary-model permission or an upstream 3.19 support promise. |
| zlib 1.3.2 static | Installed 0.1.29 provider already uses this replacement, not 1.3.1. Official February 2026 release/audit changes were compared with exact build and linked version; roundtrip/inflateCopy/malformed execution passed in 41483644241. Current vendor release is 1.3.2. No fallback bundled old zlib; future fixes require provider rebuild/retest. |
| FlatBuffers 23.5.9 builtin headers | Accepted configure reports this transitive DNN dependency. TFLite importer exists in the module, but the worker exclusively calls ONNX byte-buffer APIs. Vendor advisory page has no published GHSAs; current size-verifier change #8740 and unverified-buffer hardening report #9295 were classified as TFLite/FlatBuffers paths not reached by this fixed worker, not a library-wide clearance. Arbitrary/TFLite models stay disallowed. |
| OpenCV embedded SoftFloat/FDLIBM/FLANN and Windows Concurrency backend | These are source portions of the exact pinned OpenCV module archive, not independently downloaded versions. Their implementation is retained with notices; current related FLANN/SIMD/floating-point safety fixes are covered below. Default ADE download is not a linked G-API dependency: gapi is disabled by build-list resolution. |
| GCC/MinGW media toolchain and runtime shims | Retained exact receipt: GCC 13.2.0-6ubuntu1+26.1, binutils 2.41.90.20240122-1ubuntu1+11.4, MinGW headers/CRT 11.0.1-3build1. GCC CVE-2023-4039 is AArch64, not this x86-64 target; glibc findings are not Windows msvcrt. Current MinGW aligned-allocation report #1039 was checked against both hash-verified PE imports: aligned malloc/realloc/free resolve to native msvcrt.dll, which the report excludes, not the defective MinGW fallback. FFmpeg also applies its 64 MiB allocation guard. Exact evidence and primary references are in the independent companion; no global MinGW/GCC runtime clearance. |
| MSVC 19.44.35229.0 / v143 14.44.35207, static /MT CRT/STL/Concurrency | Replacement build job 125408169198 logs independently confirmed both compiler identity and toolset path. PE imports only kernel32.dll; no redistributable update can replace this embedded runtime. Microsoft 17.14 vendor security tables/lifecycle reviewed: no entry there identifies this selected CRT as an affected runtime; IDE/.NET/Git/curl/Brotli/MsQuic/PDB tooling entries are not shipped native dependencies. This is a bounded vendor-record result, not binary CVE certification. Latest serviced toolset/runtime must be reviewed when rebuilding; NR-02 below preserves this obligation. |
| OS DLLs / PowerShell | AI imports kernel32; media imports KERNEL32/SHELL32/bcrypt/msvcrt. Windows PowerShell management/security modules are OS-provided, not bundled. OS lifecycle/patching is an explicit external requirement, not cleared by Server 2022 tests. No Linux/WSL/VM/local SDK dependency is introduced. |
| Inno Setup 6.7.4 | Current 6.x vendor release, dated 2026-10-28. COM/variant over-release and child RedirectionGuard inheritance fixes are in this pin. App does not expose COM scripting; elevated Setup helper invocation/upgrade entry is still SEC-02/03/08. No claim that Inno's mitigation removes application path races. |
| Build/test-only Python, NumPy/OpenCV reference wheels, CMake/compiler executables, Node/Playwright 1.62.1, govulncheck | Not included as app runtime prerequisites. Native/reference/tool source and licenses remain in existing evidence. Use serviced disposable builders, checksum-pinned downloads and pinned actions; do not transfer a test-wheel vulnerability or OS library into the standalone worker inventory without linkage evidence. Current payload must be inspected, not inferred from this table. |

Immutable AI worker identity remains
`cb35f0ccb3e38a6120a6f84fe91d17beaa719f9dcb9329c192b2c885bfb121ba`, provider
`opencv-4.12.0-zlib-1.3.2-cb35f0ccb3e3`. Exact source SHA256, model identities,
licenses and execution receipts are in `dependency-evidence.md` and preparation
scripts. This review does not change a dependency, model, ABI or provider hash.

### OpenCV finite released/current safety-fix disposition

The vendor has no published repository GHSAs; that is not a clean bill. Review
also examined the official 4.13/4.14 changelog, primary fixes and the current
fastGemmThin report. References below are upstream issues/PRs. Grouping related
fixes does not imply their functions are all reachable.

| Upstream identifier(s) | Code / disposition for this worker |
|---|---|
| CVE-2025-53644, GHSL-2025-057 | JPEG2000 defect fixed in 4.12.0; imgcodecs is not built. |
| #28272 | DNN NaryEltwise pointer-buffer overflow. **Present in exact retained 4.12 source**, `AutoBuffer<char>` cast to `char**` at lines 469/515. Pinned models contain arithmetic operators; importer selects NaryEltwise. Potentially affected; **NR-01, release blocked pending patched provider and parity/negative tests**. No exploit/reachability-to-corruption execution claimed. |
| #28446, #28579, #28581, #28959, #29447 | Core error/arithmetic/UB/lifetime fixes. Core is compiled; no blanket unreachable assumption for internal inference calls. Conservatively include in NR-01 update/backport review and provider acceptance. |
| #28620, #29348, #28923 | FileStorage/YAML/compressed-file handling. Worker loads fixed ONNX buffers/BGR, not YAML/XML/JSON/FileStorage files. Not reached through current exposed worker tasks. |
| #29141 | VBLAS wide-SIMD store fix; selected build SSE2 only/no AVX2 dispatch. The reported wide-lane store condition is excluded by current ISA configuration; no general BLAS correctness claim. |
| #28386, #27582, #28233/#28250, #28259 | Median/connected-components/stackBlur/bilateral functions are not called by worker interface; ordinary resize, alignment and inference remain tested. Do not enable these APIs without re-review. |
| #28379, #29151, #29296, #28040/#27847, #28249/#28196/#27750, #29113 | Image-codec/own-MJPEG paths are not built (imgcodecs/videoio excluded). Parent Go decoders and separate FFmpeg are different components, still reviewed separately. |
| #29503, #29240, #28814, #29378, #29539, #28285, #28009 | Saved FLANN index, BRISK, ArUco/QR/ChArUco and calibration APIs not invoked by the fixed face/object entry points. Compiled module presence is acknowledged; no arbitrary index/dictionary/calibration input accepted. |
| #28397, #28226 | WASM SIMD and video optical-flow paths excluded by target/module set. |
| #29418, #29345 | ONNX Einsum/Tile bounds fixes. Checksum-verified three shipped model graphs contain neither operator; custom models are prohibited. Recheck on any weight change. |
| Current #30617 / #30502 | Dynamic fastGemmThin overflow report. Exact retained 4.12 `fast_gemm.cpp` and `gemm_layer.cpp` contain no fastGemmThin implementation, so this later-introduced path is not in this worker. Do not confuse the report with all Gemm operators. |

Model applicability was checked by downloading only the already-pinned public
three ONNX files, verifying their SHA256s, and parsing ModelProto/GraphProto node
operator fields with bounded protobuf-wire traversal (no inference/runtime change).
YuNet: Add/Conv/MaxPool/Relu/Reshape/Resize/Sigmoid/Transpose. SFace:
BatchNormalization/Conv/Dropout/Flatten/Gemm/Mul/PRelu/Sub. YOLOX:
Add/Concat/Conv/MaxPool/Mul/Reshape/Resize/Sigmoid/Slice/Transpose. This is supporting
operator evidence, not proof that NR-01 cannot trigger; the unsafe NaryEltwise
source remains a blocker. Existing successful inference is not a memory-safety
proof and does not resolve it.

### Protobuf vendor advisory disposition (all seven currently published)

| Vendor advisory | Disposition |
|---|---|
| [GHSA-8gq9-2x98-w8hf](https://github.com/protocolbuffers/protobuf/security/advisories/GHSA-8gq9-2x98-w8hf), CVE-2022-1941 | C++ applicable to stock 3.19.1; the exact official 3.19 backport is retained/applied to three parser files, previously operation-tested in accepted worker. Rebuild/retest if replacing this patch/version. |
| [GHSA-h5g9-ghrj-76p5](https://github.com/protocolbuffers/protobuf/security/advisories/GHSA-h5g9-ghrj-76p5), CVE-2024-2410 | C++ streamed JSON defect introduced 4.22.0; pinned 3.19.1 predates it and worker consumes ONNX binary, not streamed JSON. |
| [GHSA-8qvm-5x2c-j2w7](https://github.com/protocolbuffers/protobuf/security/advisories/GHSA-8qvm-5x2c-j2w7), CVE-2025-4565 | Pure-Python backend; not bundled in standalone C++ provider. |
| [GHSA-p2gh-cfq4-4wjc](https://github.com/protocolbuffers/protobuf/security/advisories/GHSA-p2gh-cfq4-4wjc), CVE-2026-6409 | PHP parser; not bundled. |
| [GHSA-735f-pc8j-v9w8](https://github.com/protocolbuffers/protobuf/security/advisories/GHSA-735f-pc8j-v9w8), [GHSA-h4h5-3hr4-j3g2](https://github.com/protocolbuffers/protobuf/security/advisories/GHSA-h4h5-3hr4-j3g2), [GHSA-wrvw-hg22-4m67](https://github.com/protocolbuffers/protobuf/security/advisories/GHSA-wrvw-hg22-4m67) | Java implementations; not bundled. |

## Executed secret/dependency/privacy controls and exact limits

The [independent receipt](security-review-closeout-verification.md) records actual
current commands, results and control-source SHA256s: Windows-target Go source/
symbol scan exit 0/zero findings; **twelve source/history tests plus six artifact
tests passed**. Final precommit source/history scan receipt, independently reproduced before
companion finalization:
**900 text inputs / 6,903,607 bytes; zero findings; all exclusions=0**.
Counters change when documentation/history changes; the final frozen candidate
must rerun and retain its own receipt. Current native/model binaries are unchanged;
no claim is made that an as-yet-unbuilt new installer payload was scanned.

`Check-Secrets.py` scans all blob objects reachable from **local** Git refs, plus
tracked/untracked nonignored current files. It reports only class and immutable
blob ID or `working-file`, never source paths/matched bytes. Three defined classes:
GitHub token formats, AWS access-key IDs and PEM private-key headers. Limits:
8 MiB per eligible input, 512 MiB total bytes read, 100,000 Git inventory/combined
input count and 16 MiB Git listings. Each Git operation bounds collection and process wait to
thirty seconds plus bounded kill/cleanup; a real blocking subprocess and output
flood control exercise timeout/cap failure. Repository hooks/fsmonitor and object
replacement are disabled, and raw blobs never request external filters/textconv.
A configured fsmonitor hook is proved runnable in a separate Git positive control,
then is not executed by the scanner. Unreadable/changed/malformed inventory,
Git/tool errors or budget exhaustion return **2/incomplete**, never a clean pass;
signatures return **1**; complete covered scan with no match returns **0**.
Binary/oversized/linked/nonregular inputs are counted exclusions, not passed
content. Working links/reparse ancestors are not followed. This does not promise
a race-resistant read jail. Current deleted-history, untracked, unborn-repository,
Git/read/budget/timeout/hook/error-redaction and all-signature positive controls are
executed.

Excluded: ignored files, unreachable/unfetched/deleted remote history, arbitrary
passwords, unknown credential formats, semantic private media/names/logs, binary/
large/linked inputs and packaged artifacts. These are explicit coverage boundaries,
not silent exemptions from privacy requirements. Review source for secret-handling
and private-data flows; never publish private support material in the first place.

`Check-ArtifactPrivacy.py` independently scans bytes (including binaries), ZIP/TAR
members and names without extraction, three signatures and supplied literal
canaries. Limits: 128 MiB per entry, 512 MiB total accounted bytes, 50,000 scan
items, depth three; missing/corrupt/linked/budget/depth errors fail. Canary lengths
16..1024 bytes, input file <=8192 bytes; emit fixed reasons/counters only. Six
controls passed, including nested signatures/binary canaries and redaction.
Existing **0.1.29** actual payload evidence 41487369408 scanned 28,454 items /
384,104,155 bytes; old protected rollback secret/catalog snapshots were deliberately
excluded from redistribution payload, not deleted. Every future candidate must
scan its actual intended distribution tree with fresh code/media canaries under
SEC-08. These are not comprehensive secret detection or privacy certification.

## Owned residual decisions — no implicit risk acceptance

| Finding / stable item | Severity / decision and required closeout evidence |
|---|---|
| S-001/002/005..010, IR-001 / SEC-01 | Tested controls/current stationary ACL closure retained. Original route/CSP/login/copy/response/restore-root findings have exact historical evidence, not new vulnerabilities. Run relevant regressions whenever their code changes; no general Windows 10 or release claim. |
| IR-002/003 / SEC-02 | Conditional high privileged-code trust. Staged new Prepare exists; AppDir/staging token trust and actual old-package failed-upgrade recovery remain blocked. Installer owner must prove before elevated retained-code execution, no weakening of LocalService/rollback rules. |
| Path replacement / SEC-03 | Conditional high privileged file boundary. Filesystem owner must enforce operation-time no-escape/identity under concurrent swaps; static checks and stationary ACL closure do not suffice. Blocked. |
| Aggregate/archive/catalog / SEC-04 | Medium availability/integrity. Restore owner must finish and validate complete byte/metadata/entry/catalog/time/memory enforcement with hostile and valid independent restoration. Parallel candidate does not become accepted merely through this record. |
| WI-01 / SEC-05 | Confirmed same-token authority, high potential impact after native compromise. Native-security owner must enforce required-input/scratch-only authority and prove canary denials plus real provider parity/recovery. Blocked; no AppContainer/restricted token is silently selected. |
| WI-02 / SEC-06 | Medium malformed-input/resource concern. Input owner must finish parent decoder and filename/metadata/tag/catalog negative/normal tests. Supported media must not be silently removed to evade review. Blocked. |
| NR-01 / SEC-06, SEC-08 release stop | Native-runtime owner must patch/replace the present NaryEltwise buffer defect and review the compiled generic-core safety fixes above, pin a new immutable worker/source/notices, execute actual model parity/malformed/cancellation/service/peak checks and full integration/restore before release. Potential high native-memory-safety impact; no exploit claim. **Open; no waiver or trusted-model excuse.** Review/applicability investigation is completed here; remediation is not. |
| NR-02 / SEC-08 | Static-runtime servicing risk. Release owner must verify the latest applicable MSVC toolset/runtime fixes when rebuilding, retain compiler/lib provenance and re-execute provider/installer checks. Installing a VC redistributable cannot service /MT code. No long-term servicing guarantee claimed. |
| Backup confidentiality/authenticity | Private local unencrypted/unsigned backups are an existing explicit design boundary, not Internet-safe secrets containers. Storage/export owner retains private ACLs and user-facing handling/restore warnings; a cryptographic format change needs a concrete design and compatibility decision. No new user risk acceptance is inferred. |
| SEC-08 | No combined release acceptance yet. Release owner must test actual accepted previous installer, new provider changes, injected failed upgrade/rollback, independent restore/ENOSPC, browser, payload privacy/integrity and applicable Windows 10/clean-PC/reboot/signing gates. Other closed items cannot clear it. |

## Mandatory update/recheck procedure

1. Freeze candidate commit and exact build configuration. Inventory compiled and
   automatically enabled components, transitive static code, OS imports, model
   hashes and builder versions. Retain source archives/patches/notices, artifact
   digests and exact provider directory; do not overwrite an old provider in place.
2. Fetch configured refs needed for the declared history scope. Run source/history
   controls and scan; treat any finding, read error, unsupported inventory or budget
   failure as a blocker. Investigate exclusions; do not raise budgets or omit files
   merely to get green. Rotate/revoke real exposed credentials and assess history/
   artifact exposure before any redacted remediation; never print the credential.
3. Run pinned govulncheck against the **actual Windows build target** with current
   DB timestamp and record versions/exit/result. Query each native vendor's current
   security and release notes, including shared utilities/auto-enabled components;
   compare fixes to exact source, not just major versions. Track each published
   applicable or potentially applicable finding with severity/owner/fix/test gate.
4. Native/runtime changes require actual representative operations before reuse:
   verify linked version and imports, model/codec parity, malformed input,
   Unicode/space paths, memory/process/output/cancellation/death controls and
   LocalService restart/recovery. Maintain source/license redistribution evidence.
   Do not repeat a native rebuild merely for a documentation/harness-only change.
5. Build and inspect the intended distribution payload with fresh private-code/
   synthetic-media canaries; run scanner negative controls, verify output is
   redacted, inspect exact installer/manifests and retain SHA256 receipts. Do not
   scan a substitute directory and claim the actual installer was examined.
6. Review all stable SEC IDs and this threat map. A new exposure mode, dependency,
   worker authority, restore format or privileged path reopens its affected
   acceptance. Resolve exploitable high/critical findings before delivery or stop
   and explain the blocker; never transform an open finding into accepted risk by
   editing a status label. Ask the user before a mitigation conflicts with plan.
7. Execute SEC-08's combined upgrade/failure/restore/provider/browser/target checks.
   State Server evidence and Windows 10 evidence separately. Until those pass,
   retain prototype labels and the previous accepted build; no automatic local
   dependency installation, private upload, signing purchase or cost commitment.

This procedure runs at every release candidate, dependency/provider/model/tool
change and newly relevant advisory. It is an active release gate, not a promise
of an unattended scheduled monitor. The primary agent owns coordination; bounded
workers may investigate/verify only within user-authorized scope.

## Primary review sources

All retrieved/reviewed 2026-11-02; existing execution receipts are linked in
`dependency-evidence.md`, `security-baseline.md` and the independent companion.
The independent reviewer inspected this complete companion and found no missing
known finding, owner or closure gate. Its final control receipts include scanner
SHA256 `cd5a76a061540b3073e712ba3075195a9b51fa8396ee29e855fa138125a27568`
and source-test SHA256 `fde1cd9618cb6891e793c059c35473cf67291c658351bf15a1713b9621f6aa4f`.

- [OpenCV security policy](https://github.com/opencv/opencv/security/),
  [official changelog](https://github.com/opencv/opencv/wiki/OpenCV-Change-Logs),
  [GHSL-2025-057](https://securitylab.github.com/advisories/GHSL-2025-057_OpenCV/).
- [NaryEltwise upstream patch](https://github.com/opencv/opencv/pull/28272.patch),
  [VBLAS SIMD fix](https://github.com/opencv/opencv/pull/29141),
  [current fastGemmThin patch](https://github.com/opencv/opencv/pull/30617.patch).
  Other numbered fixes above resolve under `https://github.com/opencv/opencv/issues/NUMBER`.
- [Protobuf complete vendor advisory list](https://github.com/protocolbuffers/protobuf/security/advisories)
  and the individual primary advisories linked above.
- [zlib release/audit changes](https://zlib.net/),
  [FlatBuffers security policy](https://github.com/google/flatbuffers/security),
  [FlatBuffers #8740](https://github.com/google/flatbuffers/pull/8740),
  [FlatBuffers #9295](https://github.com/google/flatbuffers/issues/9295).
- [Microsoft 17.14 security/release notes](https://learn.microsoft.com/en-us/visualstudio/releases/2022/release-notes),
  [C++ runtime lifecycle](https://learn.microsoft.com/en-us/lifecycle/faq/visual-c-faq),
  [runtime redistribution/servicing](https://learn.microsoft.com/en-us/cpp/windows/redistributing-visual-cpp-files?view=msvc-170).
- [Inno 6 revision history](https://jrsoftware.org/files/is6-whatsnew.htm),
  [Inno security measures](https://jrsoftware.org/is6help/topic_securitymeasures.htm).
- FFmpeg official security/download/source-fix URLs and exact source comparisons
  are enumerated in the independent companion, not replaced by third-party feeds.


## Final integrated acceptance

Root and the independent verifier accepted complete SEC-07 review coverage after
all-green [41547447887](https://github.com/aataat12/immich-clone/actions/runs/41547447887),
source 1f1cdb912b223634dec0350e58e9ae86b11893b0, job 125626387613. All 18
controls ran on Windows (4.620s), including actual hook positive/denial controls;
govulncheck 1.9.0 found no Go vulnerabilities. Source/history receipt after fetching
that candidate and before final evidence edits: 924 text inputs / 7,167,666 bytes,
zero supported signatures and zero exclusions. Final test SHA256:
7edb17177b7db12bb423089a338d54ac5a6fe2b509223aaf30d88bc271982160.
The earlier Windows run failed the test fixture's nonportable Python shebang; the
portable shell wrapper was independently reviewed and rerun successfully, without
skipping or weakening either assertion. Production scanner unchanged.

SEC-04's independent Windows implementation acceptance also passed in this run;
its controls are closed by their separate evidence, not merely by this review.
NR-01 remediation remains an explicit SEC-06/08 release blocker. No new payload
was built/scanned or provider updated, and no Windows 10 certification is implied.


## 2026-11-03 servicing update

The reviewed NR-01 finding now has a patched, actual native/application-validated
candidate: opencv-nr01.md. This supersedes the earlier selected-binary identity
for future packaging only; the installed 0.1.29 worker is not silently changed.
Broader input/isolation and actual installer gates remain. R-RESTORE-01 was found
in later validation and SEC-04 is explicitly reopened (restore-revalidation.md).
Node CLI22.23.4 was recorded in app acceptance; GitHub Actions embedded runtime's
exact Node24 patch version remains separately unverified. Review coverage does
not imply these pending mitigations/release checks are complete.
