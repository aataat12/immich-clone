# Immich Clone

Windows-native photo/video manager in feasibility validation; no installable app yet.

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
