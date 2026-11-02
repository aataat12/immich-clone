# Remaining milestones and Build Notes acceptance

Baseline: 0.1.23. This table tracks remaining work, not the already delivered
baseline. No future release number is assigned before delivery.

| Order | Milestone | Completion criteria | Status | Released In |
|---|---|---|---|---|
| 1 | Bulk local AI analysis | Analyze all unprocessed eligible photos; counts/progress, pause/cancel, restart recovery, no duplicate jobs, preserve names/corrections. Include the Build Notes page and navigation specified below. | Implemented — acceptance pending | — |
| 2 | People & Tags improvements | False-face dismissal, safe reanalysis preserving corrections, larger-library review/search performance, broader recognition-quality evaluation. | Planned | — |
| 3 | Hidden Media management | Managed-file location, restore, confirmed permanent managed-copy deletion, durable recovery and job/backup coordination; external sources untouched. | Planned | — |
| 4 | Large-library import reliability | Longer mixed-media imports, interruption/retry/duplicate/disconnected-storage tests, safe resume improvements and resolution of large-file limits. | Planned | — |
| 5 | Large-library backup and recovery | Large ZIP/ZIP64, capacity and throughput, interruption handling, independent restoration and practical full-library backup workflow. | Planned | — |
| 6 | Media compatibility and resource limits | Representative camera/video and metadata checks, malformed media and CPU/memory limits; prove actual operations/inference before adding providers or formats. | Planned | — |
| 7 | Installer, security, and update hardening | Dependency/security and diagnostic-privacy review, upgrade/rollback coverage and signing arrangements; obtain approval before new charges. | Planned | — |
| 8 | Final Windows 10 acceptance and release candidate | Clean install without development prerequisites, final reboot/service startup, upgrade/uninstall preservation, extended operation, final restore drill and documented limits/recovery. | Planned | — |

## Required in milestone 1: Build Notes

- Add a **Build Notes** button immediately next to **People & tags** on the
  Photos & videos page. It opens an application page containing the table above,
  including completion criteria, **Status**, and **Released In** columns.
- Put the common **← Photos & videos Home** button at the top right of Build
  Notes. Both navigation controls must work on desktop and narrow layouts.
- Maintain the table as milestones progress. Mark a milestone complete only when
  its acceptance criteria are met; identify outstanding validation explicitly.
  Populate Released In with the actual release delivering the completed milestone,
  never a forecast or a version merely containing partial work. Leave undelivered
  entries as an em dash. Link evidence in repository records.
- Keep the shipped page and this roadmap aligned at release preparation. The page
  describes its installed build and does not require network access or expose
  private machine information.
- Verify navigation, headings, all eight rows, status/release rendering and narrow
  layout in browser acceptance. Confirm the delivered milestone-1 row identifies
  its actual release and the remaining rows do not claim completion.

## Boundaries

Small local 0.1.23 backup validation is pending independently of milestone 1.
Previously passed local checks need no routine repetition. Reboot remains deferred
to final acceptance. Windows Server evidence does not certify Windows 10.
HEIC/video AI, RAW/HDR expansion and incremental/scheduled backups remain deferred
possibilities, not promised additions. Mobile is excluded.
