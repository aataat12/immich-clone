# Milestone 3: Hidden Media management

Current slice: hidden-item discovery, managed location and reversible restore. Candidate only; no new installer published.

The gallery has a Hidden Media entry point using the existing bounded hidden gallery. Each hidden card offers Managed file location with a read-only absolute path and Copy path (manual Ctrl+C fallback). This identifies the content-hash-named managed original, not an external source file. Full source paths are not known; relative source aliases remain catalog metadata. The path is returned only to an authenticated local session for a currently hidden item, is not included in diagnostics, and is served with no-store caching. The page does not launch Explorer or a shell, rename a file or alter filesystem permissions. Existing Download original saves with its original filename.

Restore to library uses the existing durable visibility update. No original, face, tag, analysis result, proposal or import history is deleted. Hidden originals remain included in backups.

## Acceptance for this slice

- HTTP authentication/Host guard coverage includes the new read-only location route; visible assets refuse location inspection through this feature.
- Browser acceptance uses an existing generated public video with completed sampled observations, a named face, tags and an applied reanalysis record. Hide, open/copy location, inspect a narrow dialog, restart while hidden, restore, compare full review metadata and original SHA256.
- Existing independent backup/restore, media recovery and reanalysis tests remain in the focused Windows workflow.

## Remaining milestone work

Permanent deletion is not implemented by this slice. Before adding its user action:

1. Define the explicit hidden-only confirmation and exact managed-copy scope; existing backups remain historical snapshots and external sources are untouched.
2. Serialize deletion against original import, preview, inference, proposals and backup snapshots. Refuse or drain in-flight work; no dangling catalog references or stale worker writeback.
3. Persist a versioned deletion intent and recovery state before removing managed files. Define failures and interruption recovery so restart completes or clearly reports the operation; never claim deleted bytes while retained files remain.
4. Test cancellation/permission/storage failures, restart at each boundary, duplicate reimport, export consistency, old-installer rollback behavior and independent restore from a pre-deletion backup.
5. Finish combined installer and Windows target acceptance before marking milestone 3 delivered.

No security Group 2/3 investigation or mitigation, every-frame Full Scan, new dependency or user-PC action is included. Windows Server testing does not certify Windows 10 compatibility.

## Verified candidate evidence

[Windows run 41671321055](https://github.com/aataat12/immich-clone/actions/runs/41671321055), source `9ea771a8274f500b344b042440a5abc8f6e324b1`, job `126040140853`: PASS on Windows Server 2022, Go 1.27.3, LocalService, Edge/Playwright 1.62.1. Both hidden-media browser phases passed, including actual clipboard readback, 390px layout, restart while hidden, restore to library, exact original SHA256 and full review-record equality. HTTP/guard and hidden-state independent backup/restore unit tests passed. Existing HEIC/video inference, failure/crash recovery, reanalysis controls and independent restored-frame regeneration passed in the same run. Local full web suite and vet also passed. No installer/version publication or Windows 10 certification; milestone 3 is not complete.
