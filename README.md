# Immich Clone

Windows-native photo/video manager under development. A working originals prototype
is Server-tested; installer and browser lifecycle tests have passed on Server 2022.

- [Authoritative requirements](Immich_Clone_Project_Requirements.md)
- [Status and milestone acceptance](docs/status.md)
- [Dependency evidence and Windows 10 cloud blocker](docs/dependency-evidence.md)
- [Decisions](docs/decisions.md)
- [Read-only PC inventory](scripts/Get-CompatibilityInfo.ps1): paste its contents
  into Windows PowerShell. No elevation, installation, saved report or transmission.
  Review output and share privately in conversation, never in repository issues.

`storage-probe.yml` runs bounded synthetic file checks on Server 2022 and validates
inventory execution in Windows PowerShell 5.1. It does not certify Windows 10,
media codecs, AI inference, application backups or an installer.

The current prototype supports local access-code login, immutable managed-copy
upload/folder import, deduplication, image metadata/previews, optional CPU
HEIC/video previews, browsing, reversible hide/restore, exact original downloads,
JSON export and verified independent backup restoration. It includes private
bounded diagnostic export with no automatic upload. See
[prototype limitations](docs/prototype-notes.txt) and [bulk-import milestones](docs/bulk-import.md).
Current delivery baseline is [0.1.38-prototype](https://github.com/aataat12/immich-clone/releases/tag/v0.1.38-prototype). Supported HEIC AI, sampled video AI, additive reanalysis and batch Hidden Media actions are implemented. Folder import, durable resume and pagination are available; server-side source-folder scanning and mobile features remain excluded or unavailable. See the [current roadmap](docs/roadmap.md), [backup evidence](docs/backups.md) and [independent recovery guide](docs/recovery-guide.md). The guide restores to a new folder; it does not switch the live library.

Milestone2's active People review/search/native-quality scope and Milestone6's declared media-format/metadata/resource scope shipped in0.1.38 after [combined installed acceptance](docs/release-0.1.38-acceptance.md). Deferred security work is not closed by that functional delivery. Remaining large-library import/backup recovery work follows. Every-frame Full Scan and incremental/scheduled backups remain deferred. No Windows 10 certification, final clean-machine/reboot acceptance or full-library endurance claim is made. Keep personal machine reports, media and paths out of public issues and artifacts.

The earlier [local checkpoint instructions](docs/local-validation.md) print pass/fail only and send no machine report. They are historical bounded checks, not final Windows 10 acceptance.
