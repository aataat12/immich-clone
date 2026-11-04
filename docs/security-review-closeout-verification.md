# SEC-07 independent verification — 2026-11-02

This is the independent evidence companion to [the review closeout](security-review-closeout.md). It reviews original SEC-07, not the completion of SEC-02/03/05/06/08 or Windows 10 certification. The review must retain every unresolved mitigation as an owned closure gate. A clean signature/advisory scan does not establish that the application has no vulnerabilities.

## Executed verification

The review used the active candidate based on `fed8e7ef57095faea5c30e46452e0130896a1964`, including the revised source/history scanner. Application changes being integrated by the release owner still require combined candidate checks.

| Execution | Actual result and scope |
|---|---|
| `python3 -m unittest tests.test_secret_scan tests.test_artifact_privacy -v` | **18 passed**, 2026-11-02, Linux build workspace. Twelve source/history controls and six artifact controls. These are scanner tests, not Windows application tests. |
| `python3 scripts/Check-Secrets.py` | Actual reachable local Git blobs and current nonignored files scanned. Receipt: **zero findings; 900 text inputs / 6,903,607 bytes; binary=0, large=0, linked=0, other=0** (before this receipt text and companion finalization). The local checkout is not a claim about unreachable remote history or all branches. |
| `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOTOOLCHAIN=local govulncheck -json ./cmd/immichclone` | **Exit 0, zero findings**. Official Go 1.27.3; govulncheck 1.9.0; source/symbol mode; vulnerability DB `2026-11-02T22:01:43Z`. Scanner installed as a build-only tool, not a runtime dependency. Includes the selected Windows application build; excludes native libraries/models/installer and unbuilt tags. |
| Existing selected release payload scan | Reused accepted `41487369408`: 28,454 items / 384,104,155 bytes. This is the existing 0.1.29 payload only. The combined new installer does not exist yet; its actual contents must be scanned under SEC-08. |

Control-source SHA256 receipts: `Check-Secrets.py` = `cd5a76a061540b3073e712ba3075195a9b51fa8396ee29e855fa138125a27568`; `Check-ArtifactPrivacy.py` = `dc080275a2fbea1250d5dbbde062fc7577d034d86e45d62c01ba5bfeaedd5838`; `test_secret_scan.py` = `7edb17177b7db12bb423089a338d54ac5a6fe2b509223aaf30d88bc271982160`; `test_artifact_privacy.py` = `8afa022a42cd9faab50df5e73d9bf55467049b9054061fa157c773c3c1a09dc3`.

The source controls actually test all three signature classes, clean/untracked/deleted-history inputs, ignored/binary/large/link exclusions, Git/read/budget failures and redacted diagnostics. Additional real-process controls verify whole-operation timeout, bounded output and a configured fsmonitor hook that executes under ordinary Git but not under the scanner. Git system/global configuration, fsmonitor/hooks and object replacements are disabled; blob reads do not enable filters/textconv. Independent source inspection confirms timeout applies to both stream collection and process completion, followed by bounded kill/cleanup. Artifact controls actually exercise binary and ZIP/TAR canaries, nested credentials, clean controls, missing/corrupt inputs, budget/depth failures and output redaction. Operational failure is not converted into a clean result. The signatures cover supported GitHub tokens, AWS access IDs and private-key headers, not arbitrary passwords, every credential format or private content. Artifact scans also require explicit synthetic canaries. Neither scanner proves the absence of private media, names or logs by semantic inspection.

Reproduce with Python 3 and Git; provision the pinned official Go toolchain and `golang.org/x/vuln/cmd/govulncheck@v1.9.0`, then run the commands above from repository root. Do not publish raw scanner JSON/source excerpts containing user data. Recheck fresh candidates after dependency, source or packaging changes.

## FFmpeg: actual enabled inventory and current published-advisory disposition

Pinned FFmpeg is n8.1.3, source commit `1041abdc962f4cc4f394aa8de9dc5236c0c3b9e7`, archive SHA256 `57421c6be6861f26d1dc9edf384a9d1ab6410cc0b247b2502f415df83443fe4b`. Existing binary SHA256 values are `7eb77d88e2e97afff0844d8df1408ec10f9ca66fac58f6971112cefd25fb6a0f` (ffmpeg) and `0ba24b17e5b42e02de441fb053dacca08aa52ac4f7f38e0845bda7df6b73a6e6` (ffprobe). No native rebuild was necessary for this review.

The accepted source-build configure output was independently fetched from [run 41346172469](https://github.com/aataat12/immich-clone/actions/runs/41346172469), job 124966350848. This is stronger than inferring enabled components solely from command-line switches:

| Component family | Actual enabled output |
|---|---|
| Libraries/programs | avcodec, avformat, avfilter, avutil, swscale, swresample; ffmpeg, ffprobe |
| Decoders / parsers | aac, h264, hevc / aac, h264, hevc |
| Encoder | bmp |
| Demuxers / muxers | image2, matroska, mov / image2, image2pipe, null |
| Protocols | file, pipe; network disabled |
| Filters | aformat, anull, atrim, crop, format, hflip, null, rotate, scale, transpose, trim, vflip, xstack |
| External libraries / hwaccels / bitstream filters / input-output devices | None listed |

This explicitly includes auto-enabled filters and shared library code. Decoder exclusions do not exclude common swscale/avutil/avformat code. DLL imports in the accepted build are KERNEL32, SHELL32, bcrypt and msvcrt; this media build does not use the AI worker's static MSVC CRT. OS servicing remains required.

The retained build-tool receipt independently identifies GCC `13.2.0-6ubuntu1+26.1`, binutils `2.41.90.20240122-1ubuntu1+11.4` and MinGW headers/runtime `11.0.1-3build1`; static linkage does not remove their runtime relevance. [MinGW bug #1039](https://sourceforge.net/p/mingw-w64/bugs/1039/) concerns near-SIZE_MAX arithmetic in its fallback aligned allocator. Both exact hash-verified executables were independently inspected with `objdump -p`: `_aligned_malloc`, `_aligned_realloc` and `_aligned_free` are imported from Windows `msvcrt.dll`. The primary report excludes native Windows CRT implementations from this defect. Pinned `libavutil/mem.c` additionally checks the allocation ceiling before the aligned calls; both application commands pass `-max_alloc 67108864`. This specific fallback condition is excluded by actual import/source evidence, not by a claim that every MinGW routine is safe. Compiler updates still require binary provenance and provider validation; builder-only binutils is not a runtime decoder.


Primary sources checked on 2026-11-02: [upstream security list](https://ffmpeg.org/security.html), its [versioned source](https://github.com/FFmpeg/web/blob/master/src/security), and [maintenance releases](https://ffmpeg.org/download.html). The list is community maintained and is not an exhaustive guarantee. The download page lists 8.1.3 as the latest 8.1 maintenance version; 9.0.2 is a separate newer major. Review covers the published master/9.0/8.1 entries, with the 8.1 baseline fixes explicitly checked against the pinned source. Historical earlier-release fixes are inherited maintenance evidence, not newly fuzzed proofs.

| Published identifier | Primary fix / affected code | Disposition for this build |
|---|---|---|
| CVE-2025-59733 | [0469d68](https://github.com/FFmpeg/FFmpeg/commit/0469d68acb52081ca8385b844b9650398242be0f), EXR decoder | EXR decoder not built. |
| CVE-2025-59734 | [d311382](https://github.com/FFmpeg/FFmpeg/commit/d311382c38df9c2237b33a9e8e860a5da7d2895d), [c41a70b](https://github.com/FFmpeg/FFmpeg/commit/c41a70b6bb79707e1e3a4b0e31950cd986b9f50e), SANM decoder | SANM decoder not built. |
| CVE-2025-63757 | [0c6b7f9](https://github.com/FFmpeg/FFmpeg/commit/0c6b7f9483a38657c9be824572b4c0c45d4d9fef), common swscale output | Relevant library; **fixed**. GitHub compare against the pinned commit reports ahead 2,390 / behind 0, merge-base equal to the fix. Both fixed unsigned-multiplication expressions are present in pinned `libswscale/output.c`. |
| CVE-2026-8461 | [c23d4da](https://github.com/FFmpeg/FFmpeg/commit/c23d4da3128c279b714b282e6ec292e8755007e3), [5806e8b](https://github.com/FFmpeg/FFmpeg/commit/5806e8b9f34f1b0663b3017ef9dd1aa5d08116d1), [374b726](https://github.com/FFmpeg/FFmpeg/commit/374b726ffa878ee1cadb987bd1e1e20cc7ed8845), MagicYUV decoder | Decoder not built; also fixed in 8.1.2 and branch fixes verified ancestors below. |
| CVE-2026-12706 | [2f60af4](https://github.com/FFmpeg/FFmpeg/commit/2f60af465ad78ae4be85e0bbde4067846d80b582), RASC decoder | Decoder not built. |
| CVE-2026-30997 | [1a2c16f](https://github.com/FFmpeg/FFmpeg/commit/1a2c16fe514b60e1860829c42ce199de77a007e5), AV1 decoder | Decoder not built. |
| CVE-2026-30998 / 30999 | [18b83f2](https://github.com/FFmpeg/FFmpeg/commit/18b83f2d0a0f9bcbafb0001a2911327c4b8df056), [144af8f](https://github.com/FFmpeg/FFmpeg/commit/144af8f81abc1385631b4e1f4672cd415a9c6e05), tools/zmqsend | Tool not built; only ffmpeg/ffprobe shipped. 30999 also has a verified 8.1.2 ancestor fix. |
| CVE-2026-66038 | [7c0b8c8](https://github.com/FFmpeg/FFmpeg/commit/7c0b8c8594501dd904ebbb3b45bb3f6e2c745b7d), [e7cbfd1](https://github.com/FFmpeg/FFmpeg/commit/e7cbfd1c507b57a806a5825b87d609963e862c8c), LCL decoder | Decoder not built. |
| CVE-2026-70629 | [a5fe21a](https://github.com/FFmpeg/FFmpeg/commit/a5fe21a1a410a680fe93c33b0dd696b7e1c3aea4), [cd1f545](https://github.com/FFmpeg/FFmpeg/commit/cd1f545cf27ba08f6f5b31b1e92665d7874d4fd7), RSCC decoder | Decoder not built. |
| CVE-2026-70631 | [3c287af](https://github.com/FFmpeg/FFmpeg/commit/3c287af3affe1286350faa69c02bcc5d49de18bb), [2f234ea](https://github.com/FFmpeg/FFmpeg/commit/2f234ea34c81288e3840fca632dd16481d8de39f), TIFF decoder | Decoder not built; zlib disabled in this media build. |

All nine fixes listed for 8.1/8.1.2 were independently compared to the pinned commit using the GitHub compare API: every comparison returned `ahead`, `behind_by=0`. They are: `b8d5f65b9e89d893f27cf00799dbc15fc0ca2f8e` (CVE-2025-12343); `faa382e5b196f2acb5bc5b159aaf06682291328b` and `554ae5ada98cd44686ade995102d3aa2295d7b09` (2025-67306); `8abeb879df66ea8d27ce1735925ced5a30813de4` (2025-69693); `d03483bd265b68db00c9b90f6f48dcf61c5c300d` (2026-30754); `a991b3e1102824730de6b0b2afd1c7e41fbdf03d`, `9516e6900a8294c5a9e0da8d4ad88956776d6666`, `e302bafe79e0153f4dc7faecec7203836dcd71a7` (2026-8461); and `874cd67a3c560fcc2edbfc38940777610670b604` (2026-30999).

The current published entries reviewed above leave no unclassified FFmpeg finding for the built component set. This does not excuse SEC-05's demonstrated same-token filesystem authority or SEC-06's malformed-input work. An affected enabled decoder/shared component discovered on a subsequent feed check requires a replacement/backport plus pinned hashes, source/license archive, full media parity and hostile-input checks before release. Disabling a newly affected supported codec would change supported functionality and requires the user's decision. The release owner maintains this gate.

## Independent threat and remaining-work cross-check

| Original item | Severity / affected surface | Evidence and remaining mitigation owner / closure gate |
|---|---|---|
| SEC-01 | High confidentiality/integrity if managed files have permissive ACLs | Closed by accepted full-tree Windows tests run41535508518; installer/security owner must retain refusal-before-mutation and ordinary-user-denial tests. Stationary ACL closure does not close SEC-03/05. |
| SEC-02 | Conditional high; privileged custom installation/preparation and upgrade recovery | Existing installer review identifies unsafe custom AppDir trust/staging permissions and real old-package recovery gaps. Installer owner must prove trusted temporary/preparer inputs and previous-installer failed-upgrade recovery. Open. |
| SEC-03 | Conditional high; privileged filesystem operations | Static reparse checks leave replacement races. Filesystem owner must implement and prove operation-time boundaries with concurrent swaps. Open. |
| SEC-04 | Medium availability; untrusted restore archives/catalogs | Aggregate extraction and processing bounds are assigned to the other current worker pair; release owner must require their full negative/valid restore evidence. Review does not close their implementation. |
| SEC-05 | High conditional confidentiality/integrity after native compromise | Actual canaries confirm native child can read synthetic secrets/catalogs and modify synthetic originals under the service identity despite JobObject resource controls. Native-runtime owner must enforce input/scratch-only authority and negative authority plus provider parity tests. Open; resource caps are not a sandbox. |
| SEC-06 | Medium demonstrated resource exposure; malformed media/input review | Parent Go image decoding occurs outside native JobObject limits. Input owner must finish malformed media/filenames/metadata/tags/catalog checks with negative and valid parity tests. No unproved parser RCE is asserted. Open. |
| SEC-07 | Review coverage rather than a new exploit | This independent scan/control execution and FFmpeg applicability table support the paired closeout's complete threat/advisory/risk record. Wider native inventory is in that companion. Review closure requires no missing or unowned finding; it does not silently accept open security mitigations. |
| SEC-08 | Delivery/compatibility gate | Release owner must run actual combined prior-installer upgrade/failure/rollback, providers, backup/restore, browser and applicable Windows 10 checks; scan the actual new payload. Open. |

Private backup confidentiality/authenticity, unsupported remote exposure, static-runtime updates and operating-system servicing must retain explicit treatment in the closeout and user-facing update procedure. No residual risk above is accepted on the user's behalf. Historical records showing narrower or pending reviews are superseded only for the explicitly completed coverage in this record and its companion.

## Final independent closeout inspection

The completed companion was independently read in full after its final timeout/hook controls. It covers assets and trust boundaries, all ten OWASP 2025 applicability areas, the complete current compiled/transitive native inventory, seven protobuf vendor advisories, released/current OpenCV safety-fix groups, model operator applicability, static/runtime servicing, installer and privacy limitations, every remaining stable finding, and mandatory update/recheck gates. No finding is treated as implicitly accepted risk.

**SEC-07 review coverage is complete for the declared current candidate and pinned components.** NR-01 is conservatively potentially affected: the retained OpenCV NaryEltwise pointer-buffer defect and compiled generic-core fixes require a patched/replacement provider and actual parity/negative tests. Fixed model hashes and previous successful inference do not clear it. NR-01 remains an explicit release blocker under SEC-06/08; its investigation is completed, its mitigation is not. This review does not authorize shipping the vulnerable provider or close the remaining original implementation/release items. Final frozen-candidate source/payload/Windows checks remain the release owner's mandatory gates.

### Portable hook control correction

The first Windows run exposed a test-fixture portability failure: its Python executable shebang did not run the configured positive-control hook. The correction uses Git's `/bin/sh` interpreter with shell-quoted, forward-slash Python/helper paths and explicit LF endings. Independent inspection confirms the positive assertion that the hook actually executes and the subsequent scanner-denial assertion both remain mandatory, with no Windows skip. The corrected exact test hash is recorded above. Independently rerun on Linux: all 18 controls passed in 0.786 seconds. This test-only correction does not establish Windows success; the actual Windows rerun is a separate receipt to be appended by the release owner.

### Actual Windows final acceptance receipt

The independent verifier retrieved the final job log for [run 41547447887](https://github.com/aataat12/immich-clone/actions/runs/41547447887), job `125626387613`, source `1f1cdb912b223634dec0350e58e9ae86b11893b0`. The corrected **18 scanner controls passed on Windows in 4.620 seconds**, including the actual runnable fsmonitor positive control and scanner-denial assertion. The pinned govulncheck 1.9.0 Windows source scan reports **No vulnerabilities found**. The combined job also passed Go tests/vet, permission checks, bounded restore memory/actual CLI checks, disposable Setup preparation and Edge desktop/narrow/authentication checks. This corroborates the final control fixture; it is Windows Server evidence, not Windows 10 compatibility certification or the actual previous-installer release gate.

The coordinator's fresh post-fetch candidate source/history scan reports **924 text inputs / 7,167,666 bytes, zero findings, all exclusion counts zero**, before final documentation changes. The earlier independent receipts remain historical execution records rather than being relabeled as this later count. Final test-source SHA256 remains `7edb17177b7db12bb423089a338d54ac5a6fe2b509223aaf30d88bc271982160`. SEC-07 review closure is now supported by actual Windows execution as well as independent review; NR-01 and remaining implementation/release blockers remain open.
