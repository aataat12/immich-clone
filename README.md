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
Current priority is [people/object AI feasibility](docs/ai-feasibility.md); further
import work is deferred. Server-side source scanning, bulk-import recovery/paging, AI and mobile features
are not implemented. Never put personal machine reports, media or paths in
public issues or artifacts.

[Server-tested 0.1.13 prototype](https://github.com/aataat12/immich-clone/actions/runs/41357496504)
and [local Windows 10 checkpoint instructions](docs/local-validation.md).
The user-facing local check prints pass/fail only and sends no machine report.
See status for later candidates and the limits of user-reported local evidence.
