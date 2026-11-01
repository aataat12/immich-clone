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

The current slice supports local access-code login, preserved original upload,
duplicates, JPEG/PNG previews, persistent jobs, download/export and verified backup
restoration. See [prototype limitations](docs/prototype-notes.txt). Browser-selected folder copy is supported; server-side source scanning, generated
video thumbnails and AI remain unfinished. Never put personal machine
reports, media or paths in public issues or artifacts.
