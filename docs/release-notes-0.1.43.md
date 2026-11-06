# Immich Clone 0.1.43 prototype

This prototype includes the three agreed improvements below. It remains subject
to the compatibility and validation limits described here. Use only the installer
issued with the accepted release; development artifacts are not release downloads.

Included in this build:

- Gallery cards show **Date taken** when a usable capture date is available and
  **Date imported** otherwise. The information panel explains that capture time
  zones have not been read from original metadata. Import timestamps use UTC.
  Original metadata remains unchanged.
- Search in **Photos & videos** filters the gallery by assigned person names,
  filenames, custom tags and active object tags. The query, matching count and
  Clear control remain visible. Paging and closing a photo retain the search.
  A photo containing multiple named people can be found under each name.
  Unconfirmed suggestions do not count as assigned names. Hidden Media has its
  own search and remains excluded from the ordinary gallery.
- New supported photo/video analysis can assign strong, unambiguous matches for
  known people using manually named faces from at least two independent originals. Uncertain matches stay
  in review. Automatic names are identified, can be corrected or undone, and
  never become trusted recognition references without explicit confirmation.
  Existing corrections and cleared names are preserved. Recognition can still
  make mistakes; this prototype does not promise general recognition accuracy.

For a normal update, retain a verified backup and use the existing application
and data folders. Do not uninstall or manually replace program files. Setup
requires administrator permission; the application runs as LocalService. No
separate SDK, Python, .NET, PostgreSQL, Docker, WSL or GPU installation is needed.
Original files and external source folders remain untouched. Uninstall preserves
application data and originals. Keep access codes, exports and backups private.

The first automatic assignment promotes the catalog to version 2 to preserve
its provenance. Manual-only catalogs remain version 1. This build supports both
catalog versions; the state directory layout and backup container remain version
1. Older releases cannot open a version-2 catalog. Failed-upgrade recovery restores
the captured pre-upgrade executable and catalog together; manually downgrading a
newer catalog is unsupported. Keep the verified pre-upgrade backup.

Pinned media providers, AI models, supported formats and original-file size limits
are unchanged. AI failure must leave ordinary browsing, original downloads and
backups usable. Access remains local, single-user and loopback-only; no mobile,
LAN, public sharing or cloud synchronization is added.

This is an unsigned prototype. Windows Server 2022 testing does not certify
Windows 10. Final Windows 10 clean-install, reboot, endurance and final-candidate
recovery acceptance remain open. Six original security items are closed; SEC-05
native worker isolation is paused/open and SEC-08 final target validation is open.
Milestones 7 and 8 remain incomplete. No completed native sandbox, physical
power-loss reliability or full-library acceptance is claimed. Do not expand a
real library solely to test this candidate.

Installer: Inno Setup, by Jordan Russell and Martijn Laan. Exact Go, bbolt,
x/sys, x/image, native media and AI license notices accompany the application.
