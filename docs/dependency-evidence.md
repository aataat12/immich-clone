# Dependency and environment evidence

## Existing executed diagnostic

[Run 41320566461](https://github.com/aataat12/immich-clone/actions/runs/41320566461),
commit `0a55c52a137ed844f74aa70f143669c2e27c97ba`, passed on Server 2022.
Logs reconfirm compilation/publication, service health/restart checks, SQL insert
and successful persistence assertion, service deletion and 997-byte report upload.
Artifact ID 13089081612; report ZIP SHA256
`503cc50888c4d16f02e476e41ba4c9078a2f770a37cb872220c5d0d8d6cacb83`.
Prior report retrieval is recorded in the historical diagnostic document; this
session inspected logs, not the ZIP again.

| Candidate | Evidence | Remaining gate |
|---|---|---|
| .NET 8, self-contained win-x64; WindowsServices 8.0.1 | HTTP Windows service startup/restart on Server; SDK/runtime patch not pinned | Supported production version/horizon, exact Windows 10 build, clean prerequisites, license and service account |
| PostgreSQL 14.24 runner binaries | Cluster, SQL persistence across service restart on Server | Production version/horizon, redistributable source and license, unattended clean install, least privilege, backup/restore on Windows 10 |
| Media tool/library | FFmpeg BtbN LGPL shared candidate pinned below, not accepted | Exact binary/license, CPU/native prerequisites, actual image/video/metadata operations, HEIC/RAW/HDR corpus scope, Windows 10 service execution |
| AI runtime/model | None selected | Actual intended weights/hash/terms, preprocessing, inference/outputs/memory under CPU service identity; no inference claim |
| Installer | None selected | Redistribution, clean Windows 10 install/reboot/upgrade/rollback/uninstall |

Each selected dependency must record version/architecture, official minimum build,
native DLLs/runtimes/instructions, licenses/acquisition, unattended service behavior,
security horizon and replacement path before acceptance. Empty cells are blockers.

## Windows 10 cloud investigation — 2026-11-01

- [GitHub hosted runner choices](https://docs.github.com/en/actions/how-tos/write-workflows/choose-where-workflows-run/choose-the-runner-for-a-job): standard public-repository runners are free; no Windows 10 hosted label. Server tests supplement only.
- [Microsoft client images for dev/test](https://learn.microsoft.com/en-us/azure/virtual-machines/windows/client-images): eligible Visual Studio subscription and Azure offer required. Windows 10 Enterprise/N image access depends on offer; live tenant availability and matching build are unverified. Ordinary home-PC OEM licensing does not establish this entitlement.
- [Self-hosted runner requirements](https://docs.github.com/en/actions/reference/runners/self-hosted-runners): Windows 10 x64 automation is supported by the runner. No machine, subscription or Azure management connection is established here. Prefer a disposable restricted runner; never execute untrusted pull requests on it. Provisioning/registration requires authorized cloud access and secure credential handling, not committed tokens. Installer tests need administrator access and reboot orchestration.
- [Azure dev/test offer](https://azure.microsoft.com/en-us/pricing/offers/ms-azr-0148p/): eligibility is not a free compute entitlement. Region/SKU, VM hours, disks, networking and shutdown/deletion determine cost. No quote is possible without offer/region; no resources provisioned or spending approved. Deallocation alone leaves disk costs.
- [Windows consumer ESU](https://www.microsoft.com/en-us/windows/extended-security-updates): normal support has ended; current page states consumer ESU through October 12, 2027 for enrolled 22H2 devices. Edition/enrollment and applicable lifecycle must be confirmed; successful execution alone does not confer vendor support.

Recommended route: existing eligible licensed cloud Windows 10 VM, isolated and
disposable, with manual bounded jobs and reboot access. If unavailable, prepare a
specific licensed offer and cost cap for decision before procurement. No compliant
zero-cost Windows 10 cloud environment has been established. No OS switch assumed.

## New target information and support issue — 2026-11-01

Personal machine details and license/enrollment information must not be stored in
public records. Generic compatibility research follows.

[Current .NET 10 supported OS list](https://github.com/dotnet/core/blob/main/release-notes/10.0/supported-os.md),
last updated 2026-09-28, lists Windows 10 Enterprise/LTSC versions but omits Windows
10 Pro 22H2. [Windows installation guidance](https://learn.microsoft.com/en-us/dotnet/core/install/windows)
also conditions support on OS lifecycle. Do not claim .NET 10 vendor support for the
target. Clarify policy or choose a maintainable alternative before committing to
a runtime. Existing .NET 8 Server success does not settle this issue.

Synced-media import must handle placeholders: [Dropbox documentation](https://help.dropbox.com/sync/make-files-online-only)
states opening online-only files downloads them. Inventory is not evidence files
are resident or readable by a service. No automatic full-library hydration, sync
client dependence under service account, or source deletion propagation is accepted.

## FFmpeg candidate experiment

- Upstream [download page](https://ffmpeg.org/download.html) links BtbN as a Windows
  binary builder. [Builder documentation](https://github.com/BtbN/FFmpeg-Builds)
  states minimum supported Windows 10 22H2; UCRT required, x64. Exact CPU instruction
  requirements and native dependency closure remain unverified on target hardware.
- Build: `autobuild-2026-10-31-12-59`,
  `ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip`.
  SHA256 `5cef7bc48e513f3cee7c881e8b2999d835541edb1846c3e89443a67f8f8556ab`,
  obtained from GitHub release asset digest. Workflow verifies it before extraction.
  Dated monthly build retained two years per builder; no floating latest URL.
- LGPL variant excludes GPL-only libraries such as libx264/libx265. Inspect actual
  version/config/license and bundled DLLs during execution. No binaries committed
  or distributed. [FFmpeg license guidance](https://ffmpeg.org/legal.html) requires
  corresponding-source and attribution compliance; full transitive audit remains.
- Probe covers synthetic JPEG/PNG decode and thumbnails, MPEG4-Part2/AAC MP4
  decode and frame extraction, container title metadata, unchanged original hashes,
  and malformed-media rejection. It does not prove H264/HEVC, EXIF/orientation/GPS,
  HEIC/HEIF, RAW, HDR, real-camera corpus, service account or Windows 10 behavior.
- No fixed security support horizon claimed for third-party build. Patch monitoring,
  new pinned binary reruns and an ability to replace the subprocess provider are
  required. Production acquisition/reproducibility and update policy remain open.

## Runtime alternative to investigate

[Go minimum requirements](https://go.dev/wiki/MinimumRequirements) explicitly lists
Windows 10 for Go 1.21 and later; the [current download page](https://go.dev/dl/)
also lists Windows 10 or later for x64 binaries. This makes Go a candidate for the
service/API runtime if .NET support remains unsuitable. Not selected or executed:
exact toolchain/hash, license, native Windows service lifecycle, CPU baseline and
Windows 10 execution still need evidence. No application rewrite has occurred.

OS security support is separate from runtime compatibility: Windows 10 22H2
without ESU lacks ongoing normal security updates. ESU enrollment does not by
itself establish .NET 10 support or a cloud virtualization entitlement. No
unsupported runtime or internet-exposed deployment is accepted by implication.

## Initial media execution and harness correction

[Run 41322703178](https://github.com/aataat12/immich-clone/actions/runs/41322703178),
commit `d9022bd5f14154e2f5a08fb31d2df9e8e6657fcc`, executed all nine assertions
successfully on Server 2022. Job 124896115756 logs inspected. The job itself failed
because the expected malformed-file rejection left LASTEXITCODE=1, propagated by
the Windows PowerShell Actions wrapper. Reset that expected exit after asserting
rejection, then rerun; do not describe the first workflow as passed.

Executed binary identifies as n8.1.3-27-g8dca1586f4-20261031, GCC 16.2.0, LGPLv3
or later, `--enable-version3`, shared libraries, no GPL/nonfree config flags.
Bundled DLLs observed: avcodec-62, avdevice-62, avfilter-11, avformat-62,
avutil-60, swresample-6 and swscale-9. DLL inventory is not a dependency-closure
proof; target-machine prerequisites and corresponding-source audit remain open.

## Accepted for the Server prototype — Go / bbolt / Windows service

Executed [runtime run 41325465670](https://github.com/aataat12/immich-clone/actions/runs/41325465670)
proves actual HTTP/database operations across native LocalService restart, database
rollback/reopen, and JPEG/PNG encode/decode. No production database installed.
Go 1.27.3, bbolt 1.4.3 and x/sys 0.49.0 pinned with go.sum checksums; CGO=0 and
GOAMD64=v1 avoid a C runtime installer and AVX baseline requirement. Go's supported
Windows baseline is Windows 10; actual execution evidence here is Server 2022 only.

Go and x/sys use BSD-3-Clause licensing; bbolt is MIT. Include exact license texts
in packaged builds. Go supports a major until two newer majors exist; regularly
update patch releases and rerun acceptance checks. No fixed commercial SLA for
bbolt/x/sys; monitor upstream and vulnerability reports. Catalog migration path:
versioned JSON export plus independent SHA256-named originals. SQLite remains an
option if catalog query/scale requirements outgrow the embedded key/value store.

Local cloud toolchain used for focused Linux checks: actions/go-versions asset
`1.27.3-39437817176/go-1.27.3-linux-x64.tar.gz`, SHA256
`d0fbe16a6a03fe5fbf5577ab36f95f0725af587820dca8a78e6edd4eb4e19fd6`
verified before execution. This is a development tool, not a user prerequisite.

## Installer acceptance

Inno Setup 6.7.4 official release asset SHA256
`a549929319f9120716677c685535ca4671e4b937b5fdc87e2585f32d8eb30e8c`
verified before compiler provisioning. Official source license and noncommercial
compiler use checked for this personal project. Compiler is cloud-only, not a
user prerequisite. No purchase, signing certificate or paid infrastructure used.
Generated installer carries original Inno engine notices plus exact Go/x/sys/bbolt
license texts. Target Windows baseline is 10.0, x64-compatible.

[Run 41327909119](https://github.com/aataat12/immich-clone/actions/runs/41327909119)
executes the actual installers under Windows Server 2022: fresh install, upgrade
from 0.1.0 to 0.1.1, injected failed 0.1.2 upgrade with restoration to healthy
0.1.1, uninstall preserving original hashes/catalog/access code. LocalService
account and automatic startup configuration checked; full OS boot not exercised.
Code signing is absent; Windows SmartScreen/antivirus acceptance is unverified.
Win10 clean install and reboot remain local acceptance, not inferred from Server.

## Actual browser video execution

[Run 41328317537](https://github.com/aataat12/immich-clone/actions/runs/41328317537)
used the pinned FFmpeg candidate solely in CI to generate synthetic H264/AAC MP4.
Edge on Server decoded and played that stored original: readyState, 160px video
width and advancing currentTime asserted. The application serves original bytes
with range support and does not transcode or ship FFmpeg. This establishes that
fixture/browser combination, not all MP4 codecs, HEVC/HDR, target desktop/browser
behavior or generated video thumbnails. Other browser codec failures remain
possible; original download is independent.


## Metadata/timeline slice 0.1.5

[Run 41340063567](https://github.com/aataat12/immich-clone/actions/runs/41340063567),
commit `9748c1f0ca64d98c54fbb886f7daea6d6022cce6`, passed real synthetic JPEG
metadata decoding, eight orientation transforms, malformed/big-endian/cyclic
metadata bounds, immutable-original and metadata/source-alias backup restoration
tests. Edge executed oriented-thumbnail, details, sorting and folder-path checks;
installer fresh/upgrade/failed-upgrade rollback, independent recovery and actual
NTFS disk-full recovery passed on Windows Server 2022. Linux race/vet also passed.
No new dependency; broader camera formats and optional AI remain unverified.

Installer artifact `13096838238` ZIP SHA256:
`b32c8905a3f5a4d702b815ce2129b500ca235210714134390a328e0ea57e88b5`.
Extracted `ImmichClone-0.1.5-prototype-x64-setup.exe` SHA256:
`482b940661d69108f393ba2b195a9554b70470197e6e9b6fec0b43c28cbece8e`,
matched the packaged checksum list. Desktop/narrow synthetic screenshots inspected.
This build has not been tested locally on Windows 10. Previous user-reported
local acceptance belongs to 0.1.3; reboot remains deferred to final acceptance.

## BMP/TIFF/GIF candidate — bounded experiment

Candidate: `golang.org/x/image v0.47.0`, source tag
`c437a9130dcb43e6df402a034160826e29fd29b8`, requires Go >=1.26.0.
BMP/TIFF packages are pure Go; GIF uses the pinned standard library. Probe uses
CGO=0, GOAMD64=v1, CPU only, no GPU or new redistributable DLL/runtime. Windows
baseline follows the pinned Go toolchain (Windows 10); actual OS execution must
still be recorded separately. Module ZIP sum:
`h1:za/8gVz++PPZqmZJBNSISICN5Rrj5zixP/8K++fgmqo=`.
Official sources: https://pkg.go.dev/golang.org/x/image@v0.47.0 and
https://go.googlesource.com/image/+/refs/tags/v0.47.0/LICENSE . BSD-3-Clause;
exact license must accompany any integrated release. Source obtained via Go module
proxy and checksum verification. No fixed LTS commitment; monitor upstream Go
security fixes and re-pin/retest updates. Replacement remains a decoder provider;
originals, catalog and PNG-derived previews stay independent of this library.

Acceptance: independent System.Drawing synthetic BMP/TIFF/GIF pixel decoding,
PNG roundtrip, truncated/malformed rejection, original SHA256 preservation, actual
LocalService execution and restart on Server, plus focused Linux race checks.
Only first GIF frame / TIFF page will be considered; not animation, all TIFF
compressions, camera RAW, HEIC, HDR, color-managed rendering or advanced metadata.
Application integration remains gated on the service experiment. DecodeConfig
must guard dimensions before decoding; additional encoded-size bounds are needed.

[Service experiment 41342166588](https://github.com/aataat12/immich-clone/actions/runs/41342166588)
at commit `46a8113a3d516e4782f6fa79d95a8516dea07ac3` passed all stated
independent fixture/LocalService checks. Job `124954301974` logs inspected. Linux
race test and Windows x64 CGO-disabled cross-build also passed. This clears only
the prototype integration gate, not broad TIFF or Windows 10 compatibility.

Focused integration checks on `16d82bd7d0ac85c69e7b0220e8a7933950596e64`:
Linux `go test -race` and `go vet` passed; Windows x64 CGO=0/GOAMD64=v1 cross-build
passed. govulncheck 1.9.0, targeting Windows amd64 for `./cmd/immichclone`, reported
"No vulnerabilities found" after adding x/image. This is a reachable known-Go-
vulnerability check, not a security audit or a promise about malformed corpora.


## Additional image formats 0.1.7 acceptance

[Application run 41342541106](https://github.com/aataat12/immich-clone/actions/runs/41342541106),
commit `16d82bd7d0ac85c69e7b0220e8a7933950596e64`, job `124955411866`: all
steps passed on Windows Server 2022. Logs inspected: actual LocalService uploads,
BMP/TIFF/GIF previews and viewers in Edge, unchanged source copies and restored
original checksums, legacy failed-preview retry, bounded malformed rejection,
static first GIF frame, compressed TIFF, previous JPEG metadata/video playback,
diagnostic privacy, installer fresh/upgrade/injected-failure rollback/uninstall,
independent reinstall recovery and actual NTFS disk-full recovery. Synthetic
desktop/narrow screenshots inspected; browser asserted no overflow or JS errors.
Runtime regression run `41342541155` also passed following module changes.

Installer artifact `13098391274` ZIP SHA256:
`96fc1a2d9e9b4412c6e18911c7bf6fe39612a9e76a41003b211bf5128302e2df`.
Retrieved installer `ImmichClone-0.1.7-prototype-x64-setup.exe` SHA256:
`9840080516e759745f2ce086a74d519ec2d820aba3eb1828e85c6e7703e366be`,
matched the artifact checksum list. Exact x/image BSD license bundled. No new
local prerequisite, purchase or OS reboot required. Unsigned prototype; new codecs
have not been validated locally on Windows 10 or against broad real-camera
corpora. TIFF orientation/metadata, HDR, HEIC/RAW and video thumbnails stay deferred.

User-reported 0.1.5 local check: two JPEGs displayed expected date/time and
orientation. No media, paths, machine specifications or raw reports published.
This supersedes the earlier note that 0.1.5 had no local evidence; reboot remains
deferred to final acceptance.


## Minimal source-built video/HEIC candidate — 2026-11-01

Pinned FFmpeg 8.1.3, upstream mirror tag n8.1.3, source commit
`1041abdc962f4cc4f394aa8de9dc5236c0c3b9e7`. Official sources:
https://ffmpeg.org/download.html (upstream mirror and release verification),
https://ffmpeg.org/legal.html and https://ffmpeg.org/security.html .
Prior BtbN synthetic operations were reused as context, not mistaken for a
redistribution/native-closure audit. This source-built candidate excludes GPL/
nonfree flags, external codec libraries, networking, GPU and x86 assembly; CPU
baseline x86-64/SSE2 and target Windows macro 0x0A00. This sets a build target,
not a claim of official vendor support or actual Windows 10 execution.

[Source build 41344707161](https://github.com/aataat12/immich-clone/actions/runs/41344707161),
commit `a7c7b052bca855a5b9b1951a17b2ba36101d2a0c`, build job `124961940082`
passed on Ubuntu 24.04 cross-compiling Windows x64. Exact configure command and
compiler versions retained: GCC MinGW win32 13.2.0-6ubuntu1+26.1; binutils
2.41.90.20240122-1ubuntu1+11.4. This was cloud build tooling only. Configure
reported LGPL 2.1 or later. Corresponding unmodified source, source commit, LGPL
text, build command, compiler copyright/runtime exception and binary hashes are
in artifact `13098439723`. Build instructions are retained in
scripts; no bit-identical rebuild comparison has been performed. MinGW notices
and full distribution packaging remain to be completed before app delivery.
No fixed LTS/SLA claimed: track upstream security fixes, pin/retest updates, retain
a replaceable subprocess provider; originals/catalog/derived PNGs stay independent.

Initial Windows service job `124962409763` failed the rotation dimension check
after H264/HEVC passed. The synthetic MOV lacked its intended display matrix: the
fixture producer ignored the old rotate metadata option. Regenerated with explicit
`-display_rotation:v:0 90`; independently verified matrix and 120x160 output.
Parent run failed overall; only its successful source-build artifact was reused.

[Service retest 41345139004](https://github.com/aataat12/immich-clone/actions/runs/41345139004),
commit `0cb29a1aa25546a4125aea25dbc7a42646e8010e`, job `124963245598` passed
on Windows Server 2022 LocalService. Actual CPU H264/HEVC decode, rotated MOV
dimensions, single-image 8-bit HEIC decode, expected pixel values, bounded BMP pipe
output and PNG generation, unchanged original SHA256, malformed rejection, actual
child timeout termination/output flooding rejection, and service restart passed.
Protocol enumeration allowed only file/pipe. No credentials or personal media used.
PE imports observed: KERNEL32.dll, SHELL32.dll, bcrypt.dll and msvcrt.dll only; no
VC redistributable, external codec DLL, GPU SDK or local toolchain assumed. This
is static-import plus actual service evidence, not an exhaustive dynamic API or
clean-client prerequisite proof. Total memory limit/Windows job containment,
crash/queue integration, HEIC grids/bursts/metadata/HDR, broader camera corpus and
local Windows 10 remain unverified. Keep provider out of app until relevant gates
pass. The per-allocation FFmpeg bound is not a total process memory limit.

Artifact ZIP retrieved and SHA256 verified:
`66b802519318717859bd03786832496a5ef06b0e765617a2c5037bc7ca2c012f`.
Extracted hashes matched manifest:
- ffmpeg.exe: `3b596d5d7bffee11827869480029a01fb24ab4e845117ee56a4beb52c47d85fa`
- ffprobe.exe: `bb27a0417dac832d8b1d7dcc2d7b35ea07ae150ed6f45b83c31fd05f93faa09a`
- exact source archive: `57421c6be6861f26d1dc9edf384a9d1ab6410cc0b247b2502f415df83443fe4b`

No new installer or application dependency was delivered for this experiment.
Current 0.1.7 user-reported BMP/GIF/TIFF checks passed; previous JPEGs remain good.
No raw transcript, private paths, media or machine inventory committed.

## Windows resource containment and tiled HEIC gate

[Containment 41346172423](https://github.com/aataat12/immich-clone/actions/runs/41346172423)
passed on Server 2022 LocalService before integration. Child assigned at creation
with PROC_THREAD_ATTRIBUTE_JOB_LIST (Microsoft documents Windows 10/Server 2016+),
512 MiB committed-memory limits per job/process, one active process, kill on last
job handle close, inherited-handle allowlist, fixed private environment, bounded
stdout and deadline. This is resource containment, not a filesystem/token sandbox.
https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute
https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_extended_limit_information

Actual controls allocated 600 MiB and spawned a child outside the job, then proved
refusal inside the job; 64 MiB allocation still succeeded. Tests also forced child
crash, timeout, output flooding, and parent death while a child was active; no
orphan survived, and subsequent decoding/service restart succeeded. Windows peak
job commitment was queried, not inferred from configured limits.

[Grid source build 41346172469](https://github.com/aataat12/immich-clone/actions/runs/41346172469)
adds only built-in xstack/crop filters to the same upstream FFmpeg commit. No new
codec library/network/GPU dependency. [41346363023](https://github.com/aataat12/immich-clone/actions/runs/41346363023)
proved six-tile edge-cropped HEIC; [41346549214](https://github.com/aataat12/immich-clone/actions/runs/41346549214)
proved 4032x3024 / 48-tile HEIC within the same job limits. Four distinct quadrant
colors and full output dimensions were checked, so accepting one tile cannot pass.
Synthetic fixtures were independently read back by the pinned producer; originals
were hash-verified unchanged. HDR/high bit depth, bursts, advanced metadata and
broad real-camera coverage remain unsupported/unverified.

The experiment now imports the exact shared launcher from internal/media.
[41347993775](https://github.com/aataat12/immich-clone/actions/runs/41347993775)
passed that implementation under LocalService. Earlier shared-probe attempts
41347639755/41347840175 failed only because `go mod verify` treated the locally
replaced module as a downloaded archive. Root external archives are now downloaded
and verified before building against the pinned Git checkout; actual operations
passed after the harness correction. No dependency/runtime version was changed.

Source/binary artifact `13099572196` retrieved and ZIP SHA256 verified:
`9f4d5944fa6a8b6fa1014dea881e8392b09c96b86d4b00aaccda02de7f7e6851`.
Extracted hashes matched the artifact manifest:
- ffmpeg.exe: `7eb77d88e2e97afff0844d8df1408ec10f9ca66fac58f6971112cefd25fb6a0f`
- ffprobe.exe: `0ba24b17e5b42e02de441fb053dacca08aa52ac4f7f38e0845bda7df6b73a6e6`
- source archive: `57421c6be6861f26d1dc9edf384a9d1ab6410cc0b247b2502f415df83443fe4b`

PE imports remain KERNEL32, SHELL32, bcrypt and msvcrt only. Compiler/binutils
versions unchanged; MinGW headers/runtime 11.0.1-3build1. Exact MinGW copyright
and notices (ZPL-2.1/public-domain/permissive components), compiler runtime exception,
FFmpeg LGPL text, complete unmodified corresponding source and build instructions
are retained together and included in the provider's installer directory. No
third-party codec library is linked and no separate redistributable installer is
required by this tested candidate. This is prototype packaging evidence, not an
independent legal/security audit or a clean Windows 10 execution claim.

## Optional provider integration checks

Focused Linux race/vet tests passed; Windows x64 CGO=0/GOAMD64=v1 cross-build
passed. Reachable govulncheck 1.9.0 reported no known Go vulnerabilities for the
Windows command after adding the provider. New library tests verify missing
provider failure without original loss, cancellation retaining durable queued work,
restart/retry, source-dimension preservation, thumbnail pixel correctness and
independent restore/regeneration. Metadata routing tests cover audio-first video,
grid bounds, unsupported codec and HDR/high-bit-depth rejection. These do not
substitute for actual Windows decoder execution above.

First app run 41347639780 was superseded by diagnostic-stage instrumentation.
Runs 41347840160/41348278175 passed Edge media rendering, including single/grid
HEIC, HEVC/rotated-MOV thumbnails and preview viewers. They failed the new recovery
harness because a PowerShell REST array was wrapped in a second array, so ID
lookups returned null. Corrected enumeration; final recovery/installer acceptance passed below.
No failed run is described as a validated installer.


## 0.1.9 accepted Server prototype

[41348548366](https://github.com/aataat12/immich-clone/actions/runs/41348548366),
commit `f2ecfd8be6683db923fc5e3115c9a0844c8c9d04`, job `124973523947`: all
checks passed and logs inspected. Edge executed single/grid HEIC preview viewers,
HEVC and rotated-MOV thumbnails, H264/AAC playback, actual preview retry, prior
JPEG orientation/date/sort/folder features, backup download and narrow layout.
Actual provider removal/restart left login/storage/native PNG previews operational;
restoration/retry regenerated the 12MP HEIC with correct source dimensions.
Private diagnostic export distinguished provider absence/probe/decode stages
without path/name/code leakage. All synthetic original hashes survived independent
restore. Installed LocalService decoded HEIC from a path containing spaces;
upgrade/injected failure rollback/uninstall preserved originals, and restored
reinstallation regenerated HEIC previews with the old state folder unavailable.
Actual NTFS disk-full recovery passed. Service-stop cancellation retains queued
jobs in the focused test; resource controls were also rerun using the shared
launcher in service run 41347993775 (observed peak job commitment 61,505,536 bytes).

Installer artifact `13100442001` ZIP SHA256:
`2f978424fce33e1b58e2ccb0bdb3c7bf997bb21eba212b213f922a5621a2566d`.
Retrieved installer SHA256 matched manifest:
`594b29a07a67848f332d35438af7c025ab6c0a2c14c3e068f2dde31f692b11d0`
(`ImmichClone-0.1.9-prototype-x64-setup.exe`, 22,750,262 bytes). Synthetic desktop
and narrow screenshots inspected; artifact digests checked. Exact corresponding
FFmpeg source and build/license/runtime notices are bundled. No local SDK, external
redistributable setup, GPU or reboot needed. Windows 10 execution of this provider,
clean-machine acceptance and broader corpora remain unverified. This is unsigned
prototype evidence, not a reliability/security/support certification.


## 0.1.9 local MOV report and focused lifecycle check

User reports MP4 preview/opening succeeded; HEIC local validation skipped. MOV
apparently appears then vanishes from the browser; duplicate re-upload does not
show another card. No private file, filename, path, inventory or raw transcript
was published. Exact cause and original/catalog state for that MOV are unknown.

`go run ./experiments/mov-lifecycle` passed in the cloud Linux workspace using
Go 1.27.3 and current production storage code. Existing synthetic rotated-MOV
fixture uploaded, decoder rejection deliberately simulated, list/get/original
bytes checked, duplicate re-upload returned the same record, catalog reopened
and SHA256 verified. This proves only the generic failed-preview lifecycle. It
is neither a codec compatibility result nor reproduction of the private failure.
Existing Server synthetic H264 MOV rendering remains separate evidence.

Need private diagnostics and local catalog presence/preview/error observation
before attributing the user-visible disappearance to the renderer, queue, codec
or catalog. No speculative product patch or new installer has been issued.


## 0.1.11 version/visibility acceptance

[41352572001](https://github.com/aataat12/immich-clone/actions/runs/41352572001),
commit `944ee3197cd56d27d52d780f0ecd612c1c2dfca2`, job `124985743778`: all
steps passed on Server 2022. Logs inspected. Edge asserted the actual running
0.1.11 build label, narrow visibility, hide/hidden-view/restore and unchanged
original count; injected an older response omitting the synthetic MOV and verified
it could not replace the newer grid. This reproduces a stale-response hazard,
not the cause of the private MOV report. Existing image/video/HEIC, folder/import,
backup download, provider-failure recovery and logging checks passed.

Storage tests executed hidden state through preview jobs, duplicate re-upload,
catalog reopen and independent backup restoration with original integrity. HTTP
tests rejected unauthenticated/cross-origin visibility writes and preserved
authenticated original download while hidden. Actual installed health reported
the executable build. Full installer upgrade/failed-upgrade rollback, restoration,
uninstall retention and NTFS disk-full recovery passed. Focused Linux race/vet
and JavaScript syntax checks passed. No dependency versions changed.

Installer artifact `13102570178` ZIP SHA256 verified:
`870959f0f635595e46ef3072f114390e097132a95dde05d3457072f3ddc15bd9`.
Extracted `ImmichClone-0.1.11-prototype-x64-setup.exe` SHA256 matched manifest:
`1884a464c414ac8b788deab6fe9d5dac8997e89aa3ed6c6e10b1363dea5d6b85`
(22,751,597 bytes). Desktop/narrow synthetic screenshots inspected. Local Windows
10 acceptance of this build remains pending; no new machine/media data published.
Hide is reversible catalog metadata, not deletion. The MOV issue remains awaiting
private diagnostic/catalog-presence evidence; do not claim it is fixed.


## Derived preview-queue scale comparison

Baseline [41355763079](https://github.com/aataat12/immich-clone/actions/runs/41355763079)
at `37ead920458848597d77ebcfa9903a0ec1fb41d6`, Windows Server 2022,
50k metadata-only rows: idle poll median 131379 us / 87779616 Go allocation
bytes. The measured rows deliberately have no originals; this is not bulk-media
or real-library acceptance.

Optimized [41357496603](https://github.com/aataat12/immich-clone/actions/runs/41357496603)
at `d76c40db4f4e3dcc555af9a5ca4f43886e721919`, job `125000596363`,
Server 2022: idle poll 448 allocation bytes at 1k/10k/50k rows. Reported timer
values are below VM resolution, not zero execution cost. Same 50k test still
measured full listing 131474 us / 87774328 bytes and list+JSON 192427 us /
183266368 bytes. Paging remains necessary; no browser throughput claim.
Actual PNG upload, source-byte equality, dimensions and completed preview were
verified separately among each synthetic row set (single job, not throughput).

Linux focused race/vet checks passed. Queue tests rebuild stale/orphan indexes,
retain retries/hidden jobs, inject a queue publication failure and assert atomic
catalog rollback plus safe orphan-original reuse. Existing cancellation and
independent-restore tests use the same production queue implementation. Full
Windows installer acceptance is recorded separately when complete.


## 0.1.13 queue foundation Server acceptance

[41357496504](https://github.com/aataat12/immich-clone/actions/runs/41357496504),
commit `d76c40db4f4e3dcc555af9a5ca4f43886e721919`, job `125000596465`,
passed on Server 2022. Inspected logs confirm actual queue rollback/rebuild tests,
prior cancellation/restart/restore tests, LocalService/Edge image/video/HEIC
regressions, actual version display, hide/restore and stale refresh protection,
original checksum preservation, missing-provider recovery, permission diagnostics,
fresh install/upgrade/injected-failure rollback/uninstall, independent reinstall
restore/regeneration and real NTFS disk-full recovery. No dependency changed.

Artifact `13103768097` ZIP SHA256 matched GitHub digest:
`fc2d7194b4ce10a81d59a4d13d431a5a927abd9c5009c99e12065a9d985fda11`.
Retrieved `ImmichClone-0.1.13-prototype-x64-setup.exe` SHA256 matched manifest:
`91af2f75ade51c689f901dd5af7213be016cf970603a0fca33aec04222891e0c`
(22754319 bytes). Synthetic desktop/narrow screenshots inspected. No local reboot
or new prerequisite needed. This clears the derived-queue prototype gate only;
Windows 10 execution of this build, large-library ingestion, paging/endurance and
bulk transfer recovery remain separate. No user inventory/media/logs published.
