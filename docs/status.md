# Active milestone — metadata / timeline 0.1.5

Implementation and focused cloud Linux race/vet checks passed. Windows Server
acceptance and installer build pending. Adds bounded JPEG capture-date/camera
metadata, all eight EXIF orientation transforms for previews, date grouping/sort,
a details panel, relative folder paths and duplicate source aliases. Existing
JPEG/PNG previews are queued for a derived-data rebuild; originals unchanged.
Malformed metadata, big/little endian input, recursive IFDs, path traversal,
legacy rebuild and original checksums tested with synthetic JPEGs. No new native
or third-party dependency. Windows 10 prior user-reported evidence remains valid
only for the previous build. Reboot remains deferred to final acceptance.

---

# Current checkpoint update — 2026-11-01

User-reported local Windows 10 acceptance passed: restricted service account,
application response/authentication, original upload SHA256, native PNG decoding
and persistent preview job, original retrieval across service restart, independent
backup restoration and checksum verification. User also confirmed full-image view.
These are user-reported results, not directly observed cloud execution. No raw
transcript, machine specifications, personal paths or logs are committed.

The user explicitly deferred post-reboot validation until final acceptance.
The current installer does not require a setup reboot; service restart completed.
Reboot is no longer a gate on continued development. Nothing further is needed
from the user for this checkpoint. Broader formats/metadata, optional AI, extended
reliability, and final OS reboot acceptance remain separate unfinished work.
This supersedes earlier stopped/awaiting-local-validation notes below.

---

# Status — 2026-11-01

Authoritative requirements: `../Immich_Clone_Project_Requirements.md`, including
the later user-authorized Server-first amendment. No paid cloud client OS needed.
No personal hardware, exact OS build, drive inventory, source paths or enrollment
state belongs in this public repository, logs or artifacts.

## Current product

A working native Go prototype provides loopback access-code login, streaming
upload, confirmed browser-folder copying, immutable SHA256 originals, duplicate
detection, persisted JPEG/PNG preview jobs, browsing, download, JSON export and
consistent backup/verified independent restoration. No original deletion API.
External folders are never server-scanned or changed. App state is outside sync
folders. Browser-folder copying does not retain relative directory structure yet.

Full-image viewing and actual H264/AAC MP4 playback have passed in Edge on Server.
No AI features are implemented.
Format limits are explicit in `prototype-notes.txt` and the UI.

## Milestones

| Milestone | State / acceptance remaining |
|---|---|
| M0 target/testing path | Server-first path authorized; final local Windows 10 validation pending |
| M1 basic dependencies | Go 1.27.3, bbolt 1.4.3, x/sys 0.49.0 executed under LocalService; core accepted for Server prototype. Optional media/AI remain separate gates |
| M2 originals slice | Server service and Edge browser tests passed, including confirmed folder copy and independent restore |
| M3 recovery | Abrupt exit, duplicate concurrency, corrupt backup, missing originals path, denied permission and actual NTFS full-disk recovery passed. Physical disconnect, extended endurance and OS reboot pending |
| M4 installer | Server fresh install, actual version upgrade, injected failure rollback to prior healthy version and data-preserving uninstall passed. Clean Windows 10 and reboot validation pending |

## Executed evidence

- [Infrastructure diagnostic 41320566461](https://github.com/aataat12/immich-clone/actions/runs/41320566461): reused without rerun; diagnostic-only limits retained.
- [Synthetic file primitives 41322200662](https://github.com/aataat12/immich-clone/actions/runs/41322200662): six assertions passed on Server NTFS.
- [Pinned FFmpeg experiment 41322933829](https://github.com/aataat12/immich-clone/actions/runs/41322933829): actual synthetic media operations passed; no app redistribution.
- [Go/database/LocalService 41325465670](https://github.com/aataat12/immich-clone/actions/runs/41325465670): rollback/reopen, JPEG/PNG encode/decode, service HTTP and database persistence across restart.
- [Originals application 41326182923](https://github.com/aataat12/immich-clone/actions/runs/41326182923), commit `1ee7fa2`: full native service, upload/download/hash, jobs, independent restoration and denied permissions passed.
- [Installer/browser/full disk 41327909119](https://github.com/aataat12/immich-clone/actions/runs/41327909119), commit `b0fb334`, job `124911644966`: all assertions passed. Actual Edge sign-in/upload/confirmed folder-copy/preview/backup download and narrow layout passed. Installer versions 0.1.0→0.1.1 and rejected 0.1.2 injection restored healthy 0.1.1; uninstall preserved catalog/code/originals. Actual 96MiB disposable NTFS volume filled, failed upload not catalogued, earlier original intact, and subsequent upload succeeded after space freed. Logs inspected.

Passing artifact 13092226402: installer ZIP retrieved and verified SHA256
`ffbfce3134316024f27bd53195e3139eb7ca089833ca4ae3543c7b363b4de81f`.
Installer 0.1.1 SHA256 at that commit:
`5310e7883040d89076893c2d5d0e874b6f23418380da5cfa26b5e5f9268c3e0e`.
Desktop/narrow screenshots inspected; no overflow or private data. Artifact expires
2026-11-08; this is historical evidence, not the latest media-enabled package.

Earlier installer failures exposed PowerShell service command discovery, strict
CIM parameter widths and false-success exit propagation; fixed and accepted only
after end-to-end lifecycle passed. No failed build is a validated release.

## Next concrete work — stopped at local compatibility checkpoint

Cloud prototype acceptance and the local helper are verified. The next critical
step requires the user to install the unsigned test build into a new empty local
NTFS data folder outside sync, run the local-only helper, and report its results
and behavior after a normal reboot. This workspace cannot perform that desktop
test. No background continuation is scheduled. No bulk
real-library ingestion, production release, Windows 10 reliability, AI inference
or clean-machine compatibility claim yet.

## Media-enabled acceptance

[Run 41328317537](https://github.com/aataat12/immich-clone/actions/runs/41328317537),
commit `19be51c`, job `124912848751`: all checks passed, logs inspected. Pinned
FFmpeg actually generated an H264/AAC MP4; application upload recognized it and
Edge decoded/played it (video dimensions and playback time checked). Full-image
view and HTTP byte-range retrieval also passed. Installer lifecycle and actual
full-disk recovery passed again for the changed build. Generated video thumbnails
remain unsupported; no FFmpeg binary is bundled.

The local Windows PowerShell validation helper is being exercised under the
installed package before the Windows 10 handoff. It uses an empty test library,
two synthetic originals, service restart, PNG decode and independent restoration;
no hardware inventory, personal-source reads or remote report upload.

## Local helper acceptance

[Run 41328990649](https://github.com/aataat12/immich-clone/actions/runs/41328990649),
commit `683fd8d`, job `124914893938`: every step passed, including the local
acceptance helper executed in built-in Windows PowerShell 5.1 against the actual
installed service. Native PNG decode, restart, original retrieval, independent
restore and checksum verification passed; no machine inventory is emitted.
Final service-level recovery test now adds reinstalling against restored state
while the original directory is unavailable, before the local Windows 10 gate.

## Final Server prototype acceptance / Windows 10 handoff

[Run 41329698834](https://github.com/aataat12/immich-clone/actions/runs/41329698834),
commit `cad325b672ea115907036ccd4032b03c8755428f`, job `124917002110`: all
checks passed and logs inspected. Complete recovery also passed: backup,
uninstall, old state moved unavailable, restore into a different location,
reinstall against restored state, authenticate with restored secret and retrieve
the original with matching SHA256. Restored installation uninstalled cleanly.

Validated installer artifact:
[13092787659](https://github.com/aataat12/immich-clone/actions/runs/41329698834/artifacts/13092787659).
Archive SHA256 `bc5e1afb65ece79a9099989c9c81e9cc892a383c6e4fee275b30499f9a2a70c7`.
Installer `ImmichClone-0.1.1-prototype-x64-setup.exe` SHA256
`2a34dd55c8bee11dbddde6d3c7de36d2b0b6ab2fee44c570ba9e229e5e28465b`.
Artifact retrieved, archive digest and extracted installer hash verified. Previous
media-enabled desktop/narrow screenshots inspected; final browser suite passed.
Artifacts expire 2026-11-08. This is an unsigned prototype, not a production release.

Additional focused cloud check on this source: govulncheck v1.9.0 with Go 1.27.3,
GOOS=windows, GOARCH=amd64, CGO=0, package ./cmd/immichclone reported
"No vulnerabilities found." This is a known reachable Go-vulnerability check,
not a general security audit. No production dependency versions changed.

Full reachable Git history audited for private inventory values and paths: zero
matches. No user source folder or personal media was accessed, and no machine
report was sent to CI. Only synthetic media is in browser evidence.

See `local-validation.md`. Remaining beyond this checkpoint: actual Windows 10
install/reboot behavior, clean-desktop prerequisite confirmation, extended
endurance, physical disconnect, camera metadata/EXIF orientation, broader media
previews and optional AI. No bulk personal-library ingestion authorized by a
claim of completed validation.

## Diagnostic build 0.1.3 — verified Server test build

User authorized a new diagnostic release. Adds bounded, private structured logs
for native service startup/shutdown, operation stages, HTTP outcomes, uploads,
preview jobs, backup/restore/verify and installer/rollback events. Error categories
and native error numbers are retained; raw exception text, URLs, queries, headers,
credentials, filenames, content hashes, filesystem paths and machine inventory
are excluded. Each of three channels retains two 4MiB files (24MiB total).

Authenticated browser download or standalone Start menu Export diagnostic log
produces one .log file for private return to this conversation; no automatic
upload. Export remains available with the service stopped. No log files or user
diagnostic reports belong in GitHub. Cloud installer/privacy tests, local race tests and vet passed.

Diagnostic acceptance: [run 41334317634](https://github.com/aataat12/immich-clone/actions/runs/41334317634),
commit `c09bf4d98d62d243c542b6ab9c700d9954bdc1e6`, job `124930788161`, all
steps passed. Logs inspected: authenticated HTTP diagnostic export, offline export
under Windows PowerShell with service stopped, actual startup permission failure
capture, absence of state path/media name/access code, installer failure/rollback
logging, rotation/privacy tests and actual NTFS disk-full error captured. Existing
media, backup/restore/reinstall and installer lifecycle regressions passed.

Artifact [13095085457](https://github.com/aataat12/immich-clone/actions/runs/41334317634/artifacts/13095085457).
Archive SHA256 `a31b91398916f4e7a1fb32157d554011ccf61ab52feb2cd193e241f26acc830e`.
Installer `ImmichClone-0.1.3-prototype-x64-setup.exe` SHA256
`000c73dac2b6d57d00843fe27cbbbf6eba0b1fc6d1b1fc1ce2a5eaff9e09658c`.
Archive downloaded and installer hash verified against manifest.

The initial diagnostic CI attempt caught source transport truncation before any
installer was produced. Recovered affected source from the prior passing revision,
reapplied instrumentation, and uploaded individual blobs verified against local
Git object hashes; local/remote diff was empty before final cloud execution.

Next local step: install this diagnostic prototype, reproduce any issue, and use
Download diagnostics or Start menu Export diagnostic log to return one .log file
privately. The log is never uploaded automatically; machine specifications and
personal paths remain excluded. Actual Windows 10 validation remains pending.
