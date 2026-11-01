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

Inventory received privately; no raw machine report or user path retained here.
No qualifying cloud entitlement is known. Merely installing Visual Studio is not
proof of an active subscription: [Microsoft licensing guidance](https://www.microsoft.com/licensing/guidance/Visual-Studio).

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
