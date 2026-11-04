# Dependency and environment evidence

Historical candidate table below describes the original diagnostic. Later executed
component and release checkpoints supersede its initial unselected status.

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


## People/object candidate priority

User deferred additional import features/tests to prove recognition/organization.
See `ai-feasibility.md` for exact candidates, terms, accepted Linux operations and
remaining Windows/quality/packaging gates. No AI runtime/model is integrated into
the application and no new installer is delivered for this experiment.


AI probe [41360159900](https://github.com/aataat12/immich-clone/actions/runs/41360159900)
at `8475b90e88bd4a85f4c9eed2b568f1d8509d39dd`, job `125008637332`, passed
Windows Server 2022 actual foreground and restricted LocalService CPU inference
plus restart. Inspected logs confirm Python 3.12.10, OpenCV 4.12.0 wheel revision
88 and NumPy 2.2.6; wheel hashes enforced, seven model/public-input hashes checked
before/after execution. Face detections, 112x112 alignment, 128-dimensional finite
embeddings, tiny-fixture grouping/separation/negatives and expected person/cat/cup
YOLOX outputs passed. Peak committed bytes 479862784 within 536870912 cap.
No broad accuracy, target-PC performance, Windows 10 or standalone prerequisite
claim. Exact preprocessing, limits, terms and unresolved quality/distribution gates
are in `ai-feasibility.md`. No runtime/model added to application dependencies.


## Distinct-photo face screen

[41361882205](https://github.com/aataat12/immich-clone/actions/runs/41361882205),
commit `c347e99566fc564ef052523dd4dac2139a35fdb0`, job `125013785155`:
inspected logs confirm Windows Server 2022 LocalService actually evaluated 400
ORL/AT&T distinct photographs with pinned YuNet/SFace/OpenCV/NumPy. Corpus source,
research attribution/limits, SHA256, identity-disjoint split, preprocessing and
criteria were recorded before inference in `experiments/face-quality/PROTOCOL.md`.
The corpus is downloaded from its official research archive, never redistributed
or shipped. No user media, personal details or raw embeddings enter outputs.

100 calibration and 300 held-out faces detected; 180/180 enrolled-person queries
correct, 100/100 unenrolled queries rejected. Frozen cosine 0.363; 0.05 ranking
margin. All 1350 same-person pairs passed, but 15/43500 different-person pairs also
crossed the threshold: pair similarity alone cannot authorize automatic merging.
Same counts observed on cloud Linux. Windows 18.36s, peak commitment 266977280 bytes
under 512MiB, inputs unchanged. Four accounting regressions passed (missing faces,
unknown false accepts, ambiguous ranking and collapsed calibration embeddings).
This is a narrow mostly-frontal grayscale screen, not broad-library accuracy,
safe blind clustering, Windows 10 or standalone native packaging acceptance.
Application/installer unchanged; no import regression run or local action needed.


## Self-contained native AI worker feasibility

OpenCV 4.12.0 commit `49486f61fb25722cbcf586b7f4320921d46fb38e`, exact upstream
ZIP SHA256 `a7382f4e2fd33b987f09f89866c50416a7b3647a95bab207e0dc271bab607999`.
Three bundled protobuf 3.19.1 parser files receive official upstream backport
`0299c03005fbfe086d8394fb7a873a8a21fe327f` (CVE-2022-1941), patch SHA256
`5747ab6cf1dcdefc649fbfc6b8db1fdce546683c2ebad84519d3887944fd582e`.
Unmodified archive identity and source alteration are recorded separately.
OpenCV Apache-2.0, protobuf BSD, zlib 1.3.1 zlib license; embedded
BSD/SoftFloat/FDLIBM/FLANN notices are retained via complete compiled module
sources. Required notice files and patch digest were verified after retrieval.

Build job `125019815669` in [41363876953](https://github.com/aataat12/immich-clone/actions/runs/41363876953)
passed on Server 2022 with MSVC 19.44.35229.0 / v143 14.44.35207. Static /MT
runtime/OpenCV, baseline SSE/SSE2, no dispatched instructions, IPP/OpenCL/GPU or
compressed-image codecs. Built modules: core/imgproc/dnn/objdetect and internal
calib3d/features2d/flann dependencies; bundled zlib/libprotobuf. PE audit found
only kernel32.dll, no OpenCV/VC runtime/Python DLLs. Static import/Server evidence
is not a clean Windows 10 API/prerequisite certification. CRT fixes require
provider rebuilds; no assumption of automatic redistributable servicing.

Original run failed overall in its service step: Python filename APIs could not
read Unicode fixture paths, before native inference. Corrected the reference
helper to use byte-buffer APIs; no C++ rebuild or model change. Retest
[41364545372](https://github.com/aataat12/immich-clone/actions/runs/41364545372),
service source `003a2f71c44723f2994bcc9ee830c9c18371dc9d`, job `125021810477`,
passed actual native LocalService and restart. Minimum Python-reference embedding
cosine 0.9999996401; native same/different scores 0.9414400/0.1518818, expected
person/cat/cup outputs, four malformed frames, missing model and cancellation
recovery, Unicode/space paths and input SHA256 preservation passed. Peak committed
bytes 271413248 under 536870912 cap. No raw embeddings/test media in artifacts.

Artifact `13106531939` retrieved; ZIP SHA256 matched GitHub digest:
`57f801e48e619bfb6779d5eec4901ab349fbcaa5734a4e928b2cee35a632cb5e`.
`ai-worker.exe` SHA256 matched its manifest:
`a83dcef02ebef33beccea8a3b21b88ed2bc18b298040f592be31ee9e72191ec6`
(7364096 bytes). Artifact remains labelled build-not-accepted; service evidence
above clears native feasibility only, not a product installer. Prior 41363072339
was cancelled for notice packaging; 41363375078 compiled but failed copying the
redirected output path, now fixed. Neither establishes runtime acceptance.

No new application dependency or installer yet. Current patch/model distribution
review, source-to-analysis-frame integration, persistent AI/user correction data,
broader quality, clean Windows 10 and independent AI restore remain gates. Models
remain exact pinned candidates in ai-probe; no arbitrary user-supplied model input.
Replaceable provider and independent originals/catalog remain the migration path.
Sources: OpenCV source/configuration/license links in native-ai/README and
https://learn.microsoft.com/en-us/cpp/windows/deployment-in-visual-cpp .


## Native AI pilot packaging review — 2026-11-02

The candidate packages the previously executed worker without rebuilding or
changing its hash, alongside the same three hash-pinned weights. Model-directory
LICENSE and README notices from the immutable Zoo commit, selected OpenCV module
sources/notices, protobuf backport and build provenance are installed together.
OpenCV is Apache-2.0, YuNet MIT, SFace/YOLOX Apache-2.0; model-directory terms are
evidence of published redistribution terms, not an audit of training-data rights.
The source-built static runtime has no promised security servicing horizon.

Current primary-source review: OpenCV's security page lists its private reporting
policy, not a supported-version SLA or a clean bill of health. GitHub Security
Lab's CVE-2025-53644 JPEG2000 report identifies 4.12.0 as the fixed release; this
worker also excludes imgcodecs and consumes only bounded BGR frames. Protobuf's
2026 GHSA-p2gh-cfq4-4wjc concerns its PHP parser, which is not bundled. The C++
parser backport recorded above remains explicit. Only the three bundled,
checksum-verified model files are loaded; arbitrary user model files are not
accepted. These scope checks do not establish absence of other vulnerabilities.
Rebuild and retest the replaceable provider for native/runtime security updates.

Primary sources checked:
- https://github.com/opencv/opencv/security
- https://securitylab.github.com/advisories/GHSL-2025-057_OpenCV/
- https://github.com/advisories/GHSA-p2gh-cfq4-4wjc
- https://docs.opencv.org/4.12.0/db/d05/tutorial_config_reference.html

The application additionally verifies original hashes before native decoding,
limits encoded inputs to 64MiB/40MP and working frames to 960px, runs one native
task at a time within the existing 512MiB Windows Job Object and child deadlines,
and validates normalized geometry and embeddings before a catalog transaction.
Names, custom tags and undo are independent from provider results. No GPU, Python,
SDK or local model download is an end-user prerequisite. Windows Server runner
evidence does not establish clean Windows 10 or broader face/object quality.


Pilot acceptance: application run 41374756786, job 125052477428, source
`98a6688d502e8ce3ca696487a5554027cd02b735` passed the full Windows Server 2022
Go-to-native/browser/installed-service/recovery/independent-restore chain with
this exact worker. Installer retrieved and SHA256 verified:
`f99e4d68ae45aaf067886a59fa49e38c9fdea17d92b08276510c995e36e51623`.
No native rebuild or model change; no paid infrastructure. Clean target Windows
10, reboot, broad quality and extended reliability remain unaccepted.


## Bulk import 0.1.17 application acceptance

No dependency version, native worker or model change. Reused accepted media/AI
artifacts without source recompilation. [41378462423](https://github.com/aataat12/immich-clone/actions/runs/41378462423),
job 125063798577, source `386cbb244bdf243cf908e84fc28f67e788f1551b` passed Windows
Server 2022 actual 120 mixed native-image originals/previews, partial-body socket
disconnect/reselect/receipt replay, service restart and independent receipt restore,
actual import ENOSPC/retry on bounded NTFS, native AI/media/browser regressions and
actual accepted 0.1.15 installer upgrade to 0.1.17/failed-upgrade rollback/reinstall.
Previous executable installer hash was verified before execution; old build was
not a relabeled executable from new source. Final installer retrieved and SHA256
verified: `1db93bc321919c51abe0205e10075d3c6ab09fbbd93d51b0d0caa460cb95c6c7`.
Server evidence does not clear Windows 10, large-library endurance or large backups.


0.1.19 navigation-only release: no dependency/storage change. Server 2022 run
[41386973401](https://github.com/aataat12/immich-clone/actions/runs/41386973401),
source `387e9763b90f6596c83f3b7c9ed43c857485f15e`, job 125089842083 passed Home
navigation from both pages plus the complete existing installer/media/AI/import/
independent-restore/real-ENOSPC chain, upgrading the actual accepted 0.1.17 package.
Installer retrieved and SHA256 verified:
`dc1b8b760521b02f03a8b3dae5c0dcd9961d44f96f7db31e65d5859ec2ab5a7d`.
User separately reported all twelve 0.1.15 review-flow checks and 0.1.17 basic bulk
import passed locally; original-photo comparison, not a new SHA256 test. No raw
private report retained. Full Win10 compatibility/endurance/reboot remain open.


## Indexed gallery scale probe (0.1.21 candidate)

[41391526782](https://github.com/aataat12/immich-clone/actions/runs/41391526782),
source `dd59c6b1261d8871b28449ba34f99431ad81fe20`, job 125103776560 passed
on Windows Server 2022. Existing full-list baseline above was reused; this run
measured indexed 60-row first/next pages, JSON and a separately verified real PNG.

| Synthetic metadata rows | First page Go allocations (bytes) | Page plus JSON allocations (bytes) | Index rebuild (one sample, ms) |
|---|---:|---:|---:|
| 1,000 | 81,344 | 184,224 | 39.095 |
| 10,000 | 85,776 | 188,656 | 142.978 |
| 50,000 | 76,120 | 188,592 | 827.413 |

Next-page allocation at 50k: 86,616 bytes; idle worker 480 bytes. Some measured
query durations were below VM timer resolution, not zero work. Allocation figures
are Go TotalAlloc deltas, not RSS/peak memory; rebuild is a single cold-open
sample and has separate temporary index-construction memory/work. These rows have
no originals and do not establish a 50k-media import or full-library acceptance.
Actual 600 mixed-format originals and browser media operations are separate checks.
Linux race/vet and focused cursor/index/restore/rollback guards passed. Sorted
bulk key insertion replaced an observed local random-insertion rebuild bottleneck.
No dependency, native model/worker or original-layout change.


0.1.21 full application acceptance: [41391526713](https://github.com/aataat12/immich-clone/actions/runs/41391526713),
job 125103776422, same source `dd59c6b1261d8871b28449ba34f99431ad81fe20`, passed
600 actual mixed native-image originals/previews, interruption/replay/independent
restore, actual 121-PNG browser paging/completeness/sort/hide/stale recovery with
no full-list gallery reads, existing AI/media/import regressions, actual 0.1.19
installer upgrade, failed upgrade rollback, restoration/reinstall and real NTFS
import ENOSPC/recovery. Screenshots inspected. Retrieved installer SHA256 verified:
`924ced26664cc542fdcc6fbad3bc0c1cfde3733d261fbd14f41af4149ef31ed7`.
No new dependency or local setup. No full-library, broader AI-quality, clean
Windows 10/reboot or large-backup acceptance implied.


## Backup capacity/progress 0.1.23

No new dependency/model/native-provider version or backup format. Existing x/sys
Windows GetDiskFreeSpaceEx and Linux Statfs report available bytes locally; no
capacity inventory or filesystem paths are added to support events. Context-aware
bbolt snapshot writing, ZIP/sha256 read-back and retained/range download exercised
actual operations, not just compilation.

[41398447659](https://github.com/aataat12/immich-clone/actions/runs/41398447659),
source `66a698d066c437219c6e1b23fe8dbd5ed01ad577`, job 125124797979 passed Windows
Server 2022 LocalService/Edge capacity, queued cancellation during blocked upload,
verified archive/SHA256/range/restart/discard, both downloaded archives independently
restored with old state unavailable, names/tags/receipts/original hashes, actual
post-preflight ENOSPC/retry on 192MiB NTFS, retained 96MiB original/import fault test,
actual 0.1.21 upgrade/failed-upgrade rollback/reinstall and all existing regressions.
Linux race/vet passed. Artifact retrieved and executable SHA256 verified:
`ca713b421b963c0ef645fd8479ce5999c32b5a70e04abd0110f9266779722828`.
ZIP SHA256 `d6982bcc3459c64123638aaad43a544c31409945fbbd566345aedc6e174b6c3b`.
Screenshots inspected; no new setup or reboot requested. Harness-only range-session
and undersized-volume failures before acceptance are recorded in backups.md.
Full-library/ZIP64/throughput/disconnect and clean target acceptance remain open.


## Bulk local analysis and Build Notes 0.1.25

No dependency/model/native-worker change. Source d558f7c35c66e3169c73af3b19b71167d16d1dd0,
Windows Server 2022 LocalService/Edge run 41406051070, job 125148201437 passed actual
batch inference/control/import-priority, review/JSON export, Build Notes/Home
navigation/layouts, model failure/retry, original hashes, independent restores,
NTFS ENOSPC and actual 0.1.23 installer upgrade/rollback/reinstall acceptance.
Linux race/vet and pending/cancelled/ready/failed JSON export regression passed.
Raw internal membership strings require JSON encoding in export; the failure was
fixed before accepted packaging. Test harness uses a new 64x48 synthetic PNG to
avoid reanalyzing a completed batch member or violating the provider minimum.
Retrieved installer 102605388 bytes SHA256
e88a3fee85da9ecc3efe574b844ab534c5254d9cb1ea455e6d78338e8763bf00;
artifact ZIP SHA256 cbbccb47d52ea91b9a4b259f76bd8fcdb67b69b79288e0cf19220d0b425684fe.
Both verified. Seven-day artifacts; unsigned prototype. Source remains in PR #1
pending main-merge authorization. No target Windows 10/broad quality/endurance
claim; no full-library import or reboot requested.


## Security candidate 0.1.27 — 2026-11-02

No installed dependency/model/native-provider change. govulncheck v1.9.0 was
acquired as a checksum-verified Go build/test tool, not added to app go.mod.
Windows-targeted source/symbol scan against cmd/immichclone returned zero findings
using database https://vuln.go.dev, timestamp 2026-10-27T21:16:18Z; actual Windows
runner scan passed as well. This excludes C/C++, installer tools and unknown
vulnerabilities. Selective current native advisory review and remaining zlib/
static-runtime servicing gates are in security-baseline.md; no blanket clearance.

[41469530044](https://github.com/aataat12/immich-clone/actions/runs/41469530044),
source 19b049ffeb7902fe4509a0c50a83b3c754901c8d, job 125359889950 passed actual
Windows socket-deadline regression and the full existing service/browser/native/
restore/ENOSPC/0.1.25-upgrade/failed-upgrade rollback/reinstall/uninstall chain.
Installer 102607252 bytes SHA256
cffb19c052ac17cac30c62f845225cc6e30868f8d14389dc9e7ffd75876312ff
retrieved and verified against manifest; package ZIP digest verified separately.
Build Notes screenshots inspected. Unsigned prototype candidate; broader security
milestone, target Windows 10 and extended reliability remain unaccepted.

## Native compression servicing candidate — zlib 1.3.2

Official source https://zlib.net/zlib-1.3.2.tar.gz, SHA256
bb329a0a2cd0274d05519d61c667c062e06990d72e125ee2dfa8de64f0119d16;
upstream zlib license retained unchanged with the archive. Static MSVC /MT build
replaces OpenCV's bundled 1.3.1 via explicit external include/library selection;
OpenCV 4.12.0, protobuf parser backport and the three model hashes stay unchanged.
No new end-user DLL, interpreter, GPU or SDK requirement. CRT remains statically
embedded and requires rebuilt/retested providers for servicing.

[41483644241](https://github.com/aataat12/immich-clone/actions/runs/41483644241),
source 30fc135bc0f6bd1d5af40ddaed17527caf91e219: build 125408169198 and
LocalService 125411689735 passed. Real zlib version/roundtrip/inflateCopy/malformed
operations passed on cloud Linux and Windows before downstream compilation.
Worker reports linked zlib 1.3.2; no fallback old zlib library and PE imports only
kernel32.dll. Native face/object inference, Python-reference minimum embedding
cosine 0.9999996401, four malformed frames, model/cancellation recovery and restart
under LocalService with Unicode/space paths passed. Input hashes unchanged;
peak commitment 271380480 within the 536870912-byte job cap.

Artifact 13161790873 ZIP SHA256 verified:
6bb2cbe69687501b0323f1d40fe1419109cf083e40bae219cb03f029f5986d3c.
7365120-byte ai-worker.exe SHA256 matched manifest:
cb35f0ccb3e38a6120a6f84fe91d17beaa719f9dcb9329c192b2c885bfb121ba.
Replacement source digest, notices, static import and version evidence inspected.
Native candidate archive credential-signature scan passed (4269 scan items,
72776961 bytes). This is isolated component acceptance, not app/installer
integration or Windows 10 certification. Existing installed provider ID/hash
remains unchanged until separately integrated and accepted in the broader release.


## zlib 1.3.2 worker integrated in 0.1.29

Use immutable provider opencv-4.12.0-zlib-1.3.2-cb35f0ccb3e3 with the previously
accepted cb35f0cc... binary and unchanged models. Packaging verifies replacement
source SHA256, source/license/version evidence and worker hash, without rebuilding
native code. [41487369408](https://github.com/aataat12/immich-clone/actions/runs/41487369408),
source f2242c2942caf1cc923a1ea7ad708900a8a3ad5a, job 125421140599 passed installed
native inference, old-ready-result/name/tag preservation, failed cross-provider
recovery to the actual 0.1.27 executable/provider, upgrade/rollback, independent
restore/reinstall, private-canary payload checks and real NTFS full-disk recovery.
Installed scan: 28454 items / 384104155 bytes, selected signatures/canaries only.
Installer 104117297 bytes SHA256
646a47acc8ac77751c8343202d12c09eff0a754fd0731f976b0b7e8fc9dcea66 verified;
package ZIP SHA256 458c02d6b316f8054d068565a5815159e20a197472d3e6161812f1e535416493.
Screenshots inspected. No new end-user prerequisite, model update or schema change.
Clean Windows 10, new-worker target inference, wider security and long endurance
remain distinct gates; this is an unsigned cloud-accepted prototype.


## NR-01 OpenCV servicing — 2026-11-03

Retain OpenCV 4.12.0 base, models and preprocessing; apply six hash-pinned official
production backports (28272/28446/28579/28581/28959/29447), plus the separately
recorded project single-plane mean correction required by unchanged numeric
oracles. Exact patches/source provenance, hash and execution are opencv-nr01.md.
Native run41551816976/build125641002773/service125643646910 passed 64 API cases,
core controls, actual model parity/malformed/model loss/cancellation/restart under
LocalService. Compiler MSVC19.44.35229/v14314.44.35207; static /MT, CPU SSE2,
only kernel32 import. Static runtime servicing remains a provider rebuild gate.

New worker SHA256 15d1c2ca7ef3f526b34ac4823f9337c51280ad171ddf43df4f3af5d8128a7c58,
7365120 bytes; provider opencv-4.12.0-nr01-zlib-1.3.2-15d1c2ca7ef3.
Artifact13192746922 ZIP SHA256
2d420cb78acd8d9d170eeac9952f0c669d34cac6a8e071228ee8af4ee3d7c3ea verified.
Actual signature scan:4291 inputs/members,61531095 bytes; not complete privacy
or unknown-vulnerability proof. Regression executable excluded from app packaging.

Application [41554459739](https://github.com/aataat12/immich-clone/actions/runs/41554459739), source
ae1fbc3011de33df9a1c25594b6f9864f535bdb8, job125649834966 passed actual selected-worker
service/browser/inference/model recovery and independent original/organization
restoration. No installer/version/main merge, no Windows10 or broad accuracy
certification. R-RESTORE-01 remains unexplained and original SEC-04 is reopened;
other full release/upgrade/rollback/input/isolation gates remain unchanged.


## Milestone6A existing-provider execution

No new product dependency. FFmpeg8.1.3 provider7eb77d88e2e9 and accepted patched
OpenCV provider15d1c2ca7ef3 supply bounded960px still/video analysis frames.
Actual LocalService/restart feasibility41559260341 (sourcee95958bc9f24a1e56cdd5e47cce2248f80440d2a,
job125665621103) passed single/grid12MP HEIC reference comparisons (minimum
embedding cosine0.9966728075891548), known video PTS/content, H264/HEVC,
rotation/VFR/short clips, cancellation, malformed/no-video refusal and immutable
fixture checks. Tolerances preceded execution; no empty-grid substitute.

Actual application/Edge acceptance41564920558, source9c15c56dba52481f98d8b255e98303f6ce29af7b,
job125684028344 passed HEIC inference, sampled H264 frame observations and browser
controls, service restart, original downloads and independent restored derived
crop/context regeneration. Exact scope and remaining delivery gates are
heic-video-ai.md. Server2022 evidence does not establish Windows10 compatibility.
Python/Pillow/pillow-heif and the fixture encoder are cloud test tooling only;
none is packaged as a new user dependency. Native models/binaries unchanged.
The earlier SEC04 reopening statement above is superseded by the original-eight
classification in security-priority-groups.md; no Group2/3 investigation here.


## SEC-03 installer file helper candidate — November 4

`cmd/installerfiles` reuses pinned Go1.27.3 and x/sys0.49.0, Windows x64,
CGO=0/GOAMD64=v1. It is a bundled fixed-action executable, not an end-user SDK,
new service, listener, user account, runtime download or native provider/model.
Existing Go/x/sys distribution notices apply. Inno verifies its compiled payload
hash and pins the staged bytes; the configured installed-payload canary inventory and Windows
Go scan now include this executable (their full release execution remains pending). Actual held-object descriptors and normalized
volume-GUID names use existing Windows APIs. Cross-compilation/vet and focused actual helper/privacy/alias plus production Setup
execution passed in run41842717775/source d74c6ee; see filesystem-boundary-milestone7.md
for exact operation evidence and remaining nonconverted paths/full release gates. These APIs do not establish Windows10 behavior from Server.


## Native installer diagnostic integration

No dependency/version/runtime change: the same staged Go/x/sys helper now writes
fixed-label installer diagnostics through validated handles. First focused actual
Windows Server2022 acceptance41846533710, source7f4b99b, job126631387368 passed
append/rotation/privacy/alias/oversize controls and real Inno failed-Setup recovery
when the optional failure note was a hard link. Outside bytes, original/code hashes,
health and completion order passed; existing native and SEC-01 suites passed.
Worker review/tests complete. Exact bounds and remaining root/ACL/init/Setup/full
release/Windows10 gates: filesystem-boundary-milestone7.md. No public artifact
or new end-user prerequisite; this does not close original SEC-03.


## Private storage root creation integration

No dependency/version/runtime change. Existing NtCreateFile now supplies a protected
security descriptor while creating each missing directory relative to held parent
handles; IO_STATUS_BLOCK distinguishes newly created objects for verification.
Focused Server2022 execution41849734950/source db37f6c/job126642511204 passed
exact root/child inheritance, protected creation under permissive parent, existing
ACL/byte preservation, real symlink refusal, held-ancestor and partial-progress
controls; separate ordinary-user reads/writes were denied by creation alone.
Actual Setup/rollback and existing SEC-01 compatibility passed. Scope/evidence:
filesystem-boundary-milestone7.md. Existing-tree ACL normalization is unconverted;
no new distribution/runtime prerequisite, artifact release or Windows10 certification.


## Held single-object ACL API proof — accepted after initial queue

Same pinned Go/x/sys/Windows APIs; no new dependency or product prerequisite.
Source8adf2fa adds an isolated MAXIMUM_ALLOWED directory-open + SetSecurityInfo
protected-DACL operation. Actual no-propagation/locked-child/inheritance and
outside-target behavior must be proven before installer use. Independent review,
Windows cross-compilation and vet passed. The initial queued observation was superseded by inspected PASS logs for
run41862213427/job126686818594 on Windows Server2022. Single-object suppression,
child preservation/future inheritance and writer/refusal controls passed; this
component alone did not establish installer integration or Windows10 acceptance.


## Existing ACL normalization integrated

Same Go/x/sys dependencies, no new service/runtime/provider/model. Actual held
tree enumeration uses Go names-only Readdirnames on duplicated handles backed by
GetFileInformationByHandleEx; no pathname enumeration fallback. Parent-first
individual MAXIMUM_ALLOWED SetSecurityInfo updates preserve explicit ACEs and
protected children, with re-enumeration/final actual-descriptor verification.
Windows Server2022 run41868231145/source989fb085/job126707927278 passed native
policy/refusal/late-name/alias controls, actual Setup sharing refusal with verified
safe service restart, normal/failed-upgrade recovery and closed SEC-01 matrix.
Independent worker review/tests complete. Full bounds, first fixture failure and
remaining integrated/target gates: filesystem-boundary-milestone7.md. No new public
installer or Windows10 certification; original SEC-03 remains open.


## Installer initialization through LocalService

No new dependency, service, account, launcher, schema or runtime download. Existing
LocalService startup performs store/access-code initialization before health;
installer no longer runs elevated offline init. Existing x/sys TokenElevation
query guards Windows elevated explicit init before any diagnostic/state access.
Windows Server2022 run41873094251/sourceb31f1aa/job126725226588 passed actual
elevated no-write direct/junction/hard-link controls, fresh Program Files install
with actual LocalService metadata/directory owners, unchanged commands log, usable
upload, post-initialization failure/rollback and SEC01 compatibility. Linux cmd
tests and Windows build/vet passed. Exact test-only status correction, broader
fixture scaffolding scope, other-offline-mode and final target/combined limitations
are filesystem-boundary-milestone7.md. No new artifact or Windows10 certification.

SEC-03 code-permission checkpoint adds no dependency. Existing x/sys native
held-tree DACL API replaces code-root icacls. Windows Server2022 run41882276092,
source802e09733e31caa1a272400dbca4477b8b99c7e5/job126757536563 completed success:
actual read-data file sharing/own-image, alias and unsafe ACL refusals, preserved
reader ACEs/private rollback, inherited root, custom/fresh Setup, recovery and
SEC01. Real outgoing same-object hard links can be created and are detected by
link-count validation; no atomic hard-link freeze claim. Exact scope, failed-test
corrections and remaining publication/race/full-provider gates are recorded in
filesystem-boundary-milestone7.md. Published0.1.39 and Windows10 limits unchanged.

## Setup native payload publication

No new runtime/dependency. Existing Go/x/sys NtCreateFile held leaves now include
DELETE for exact newlycreated-payload rollback, with SetFileInformationByHandle
FileDispositionInfo exercised by actual Windows recovery tests. Native inventory
uses held Readdirnames; source and all destination policies/aliases/sharing are
validated before old-byte truncation. Pinned Inno6.7.4 dontcopy temporary extraction
and explicit fixed UninstallDelete names were proven by actual compilation/install/
removal, not inferred from package availability. Official semantics:
https://jrsoftware.org/ishelp/topic_isxfunc_extracttemporaryfiles.htm and
https://jrsoftware.org/ishelp/topic_uninstalldeletesection.htm.
Windows Server2022 run41894239272/source30fa427021139265791f36a14409130adb13b4b9/
job126799357326 completed success: independent distinct-code full recovery, real
links/unsafe ACLs/writers, corrupted/missing recovery sources, late name refusal,
custom/fresh actual Setup, fixture rollback/removal/original/privacy and SEC01.
First passing run41893555355/source33a76c5 was superseded after Install ordering
and late-name controls. Full provider/prior-release, kill/powerloss, remaining
metadata/uninstall/offline paths and Windows10 acceptance remain unproven by this
bounded gate. See filesystem-boundary-milestone7.md for exact limits.

## Native uninstall cleanup

No new runtime/dependency. Existing Go/x/sys native read/DELETE handles and
SetFileInformationByHandle disposition now remove actual held trees bottomup,
with source/staged hash pins and Inno directory-chain pins. Pinned Inno6.7.4
InitializeUninstall/usUninstall/Deinitialize gates and overwrite logging were
compiled/executed; normal removal and refusal gating are actual evidence, not
documentation-only assumptions. Official event/log semantics:
https://jrsoftware.org/ishelp/topic_scriptevents.htm and
https://jrsoftware.org/ishelp/topic_setup_uninstalllogmode.htm.
Windows Server2022 run41898776310/sourcec6d384eca9d675a5053f66e60d0354821fff1f61/
job126815286190 completed success: native selected cleanup/refusal/late-name/
persistentpin and root Inno active-metadata writer/policy controls; actual
modified bootstrap/hardlink/real-user-write refusals; custom/fresh uninstall
removal/original and unrelated-file preservation; installation/rollback/SEC01.
Exact metadata-only audit permits Inno's own active root journal/image without
claiming data-sharing exclusion; payloads retain read/delete sharing pins and
all owner/DACL checks. Prior run42e011ad failed normal audit, diagnostic6daa7e14
located it before service stop; corrected source passed. Full metadata-leaf races,
offline/interruption/durable fullprovider recovery and real prior-release gates
remain original SEC03 scope. Server results do not certify Windows10.

## Entrypoint guards, existing-code read and stationary metadata

No new dependency. Existing x/sys token/SCM-context APIs and winfs read handles
are executed by actual Windows controls. Windows Server2022 run41905712864,
sourcef8f1c0a1fbd7582b6fc54d2baa7294ca42fe58c5/job126839683829 completed success:
elevated/direct/unknown early no-write aliases and missing paths, existing-code
read without catalog/log initialization, valid journal hardlink outside-byte/PID
preservation, explicit post-health rollback, LocalService installs/uninstalls
and SEC01. Windows build/vet/Linux cmd tests passed; worker reviewed/tested.
Initial run081a5f08 failed test-process exit capture; rerun changed only harness
and docs, retaining exact assertions/product bytes. Admin recovery, pre-main
worker, durable payload interruption and real metadata concurrency remain original
SEC03 implementation gates. Corrected best additional passes5 (range5–7), not
full integrated closure or Windows10 certification. See filesystem boundary record.

## Administrative read-only verification and backup

No new dependency. Existing bbolt1.4.3 ReadOnly+OpenFile consumes actual held
Windows read-only catalog handle; actual mapping/shared lock and default-zero
Tx.WriteTo reusing fd were executed in native tests and ordinary RX-user CLI.
ZIP catalog streams directly, no source temp; existing x/sys NtCreateFile
FILE_CREATE supplies final private DACL and exclusive-name create, not OPEN_IF.
Windows Server2022 run41913488167/source348dd18b790fcc47e6ccf5228a723be7f0fbd125/
job126866916248 completed success: complete source immutability, real input/output
aliases/writers, unsupported state, private/exclusive output, bounded integrity
failures/exact cleanup, independent ZIP/restore; actual admin installed source and
separate RX-only source-copy verify/backup, archive ACL/checksums, preserved
administrator recovery, remaining Setup/uninstall/rollback/SEC01. First run017b5130
passed product checks but failed fixture ACLrestore; accepted fixture avoids
installed ACLmutation without product changes or weaker checks. Existing automatic
catalog baseline41912052098 passed. Native tests/cross-build/vet and Linux baseline
backup checks passed; two worker reviews/tests done. Full untrusted restore,
durable installer interruption, metadata races and genuine release/providers
remain original SEC03 rows. Published0.1.39 unchanged; not Win10 certification.
