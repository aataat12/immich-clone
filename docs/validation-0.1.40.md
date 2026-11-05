# 0.1.40 validation handoff

Use this document as the focused validation assignment for Immich Clone 0.1.40.
The prior exercised 0.1.39 checks were accepted. Validate this update and its new
navigation; do not repeat historical tests solely for chronology.

## Exact package

Release: https://github.com/aataat12/immich-clone/releases/tag/v0.1.40-prototype

Download these assets from that exact release:

- ImmichClone-0.1.40-prototype-x64-setup.exe
- SHA256SUMS.txt
- validation-0.1.40.md

Verify the installer's SHA256 against SHA256SUMS.txt and the release body before
executing it. The displayed application version and /health build must be
0.1.40-prototype. Do not use an older internally numbered 0.1.41/0.1.42 test
installer. The accepted package is unsigned; report a Windows protection block
privately rather than disabling protections.

## Preparation and update

1. Obtain any user permission required by your tools before installation. Use
   the project's existing service/application controls and the exact installer.
2. Retain an independently downloaded Ready backup and its matching recovery
   instructions. Keep that archive private; a temporary server copy alone is not
   an independent backup. Do not discard a Ready copy merely to enable Create.
3. Upgrade the existing 0.1.39 installation using its existing application/data
   folders. Setup manages the service. Do not move the live library, manually
   replace binaries/catalog/provider folders, edit permissions or run elevated
   direct serve commands. No SDK/Python/Node/database/GPU toolkit setup is needed.
4. If Setup refuses a location or fails, stop and report the exact visible
   message, actual running version and whether the old library is accessible.
   Do not bypass its checks or repeatedly retry an uncertain installation.

## Focused checks

Record PASS / FAIL / NOT TESTED for each row. Use existing media and organization
data; do not import or reanalyze files just to manufacture a test condition.

| Check | Expected result |
|---|---|
| Version and preservation |0.1.40-prototype is displayed; existing photos/videos, names, tags, AI results, hidden items, import history and retained backup remain available. Compare a previously known original hash if available; otherwise record visual preservation separately from hash proof. |
| Five-page navigation |Library, People & tags, Bulk import, Backups and Build Notes open and retain working controls. Hidden Media opens the correct view; returning to Library restores the normal view. No blocking page/script/CSP error. |
| Desktop and available narrow layout |Navigation, cards, controls, dialogs and scrolling are readable and operable. Test narrow width only if your available tool can do so; otherwise NOT TESTED. No page-wide horizontal overflow. |
| Search and task panels |Search an existing person/tag/filename using the existing matching scope. Library search hands off to People & tags; Local analysis and Review faces open/focus the relevant existing panel. Do not infer library-wide server search or global pending counts. |
| Existing media and organization |Open existing image/video previews and an original download; existing named faces/tags/results remain. If legitimate preview or AI work was already queued, observe status/responsiveness; otherwise processing is NOT TESTED. |
| Import history |Prior completed import history remains accessible. A new bulk import or storage-disconnect test is not required. |
| Backup controls |Retained Ready archive, download and recovery instructions remain usable. Create remains disabled while a Ready copy is retained. Do not discard the independent copy or run a live restore drill. |
| Authentication |Sign out deliberately, confirm protected pages require login, then re-enter the existing private code and confirm normal navigation. Do not reveal the code in the report. |
| Build Notes and limits |Milestone7 remains In progress with partial 0.1.40 delivery. Curation is unavailable; Cinema/global review counts/Full Scan/RAW/HDR are not presented as completed features. |

## Report format

Report the exact version, installer checksum match, upgrade outcome and each
check's PASS/FAIL/NOT TESTED result. For a failure, give the page/action, expected
versus observed behavior and whether original/library access is preserved.
Separate newly observed 0.1.40 results from reused 0.1.39 results. Do not convert a
missing viewport, absent queued work or unavailable hash into PASS.

Keep screenshots, diagnostic exports and logs private and review them before
sharing. Do not include access codes, personal media/names/paths, raw machine
inventory or backups in public issues/CI. A sanitized summary is sufficient first.

## Scope boundaries

Cloud acceptance covers actual Windows Server 2022 LocalService/Edge, genuine
prior-release upgrade, initial/late injected rollback, pinned installed media/AI,
original/organization preservation, independent offline restore/reinstall and
uninstall. The release body links its exact tested source/run. Focused separate-
account concurrency and interrupted-Setup/retry evidence is recorded in
sec03-combined-acceptance.md. Do not repeat adversarial races, forced crashes,
permanent deletion, a real restore drill or a reboot on the user's working library
for this update. Final Windows 10 clean-install/reboot/endurance and SEC-05/06/08
remain separate; this validation does not certify them or complete milestone 7.
