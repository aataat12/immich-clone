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
| Media tool/library | None selected | Exact binary/hash/license, CPU/native prerequisites, actual image/video/metadata operations, HEIC/RAW/HDR corpus scope, Windows 10 service execution |
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
