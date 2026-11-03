# Milestone 3: Hidden Media management

Current candidate: hidden-item discovery, managed location, reversible restore and explicitly confirmed permanent managed-copy deletion. Not in a released installer.

The gallery has a Hidden Media entry point using the existing bounded hidden gallery. Each hidden card offers Managed file location with a read-only absolute path and Copy path (manual Ctrl+C fallback). This identifies the content-hash-named managed original, not an external source file. Full source paths are not known; relative source aliases remain catalog metadata. The path is returned only to an authenticated local session for a currently hidden item, is not included in diagnostics, and is served with no-store caching. The page does not launch Explorer or a shell, rename a file or alter filesystem permissions. Existing Download original saves with its original filename.

Restore to library uses the existing durable visibility update. No original, face, tag, analysis result, proposal or import history is deleted. Hidden originals remain included in backups.

## Acceptance for this slice

- HTTP authentication/Host guard coverage includes the new read-only location route; visible assets refuse location inspection through this feature.
- Browser acceptance uses an existing generated public video with completed sampled observations, a named face, tags and an applied reanalysis record. Hide, open/copy location, inspect a narrow dialog, restart while hidden, restore, compare full review metadata and original SHA256.
- Existing independent backup/restore, media recovery and reanalysis tests remain in the focused Windows workflow.

## Remaining milestone work

The deletion implementation below is a candidate. Remaining acceptance gates:

1. Focused Windows LocalService/browser deletion, permission-failure, crash/retry and real pre-delete archive restoration.
2. Deletion-specific full-disk/transaction failure checks on disposable NTFS storage.
3. Combined actual old-installer upgrade/failed-upgrade rollback, including pending intents, followed by installed deletion/restore/uninstall checks.
4. Final target Windows acceptance before marking milestone 3 delivered.

No security Group 2/3 investigation or mitigation, every-frame Full Scan, new dependency or user-PC action is included. Windows Server testing does not certify Windows 10 compatibility.

## Verified candidate evidence

[Windows run 41671321055](https://github.com/aataat12/immich-clone/actions/runs/41671321055), source `9ea771a8274f500b344b042440a5abc8f6e324b1`, job `126040140853`: PASS on Windows Server 2022, Go 1.27.3, LocalService, Edge/Playwright 1.62.1. Both hidden-media browser phases passed, including actual clipboard readback, 390px layout, restart while hidden, restore to library, exact original SHA256 and full review-record equality. HTTP/guard and hidden-state independent backup/restore unit tests passed. Existing HEIC/video inference, failure/crash recovery, reanalysis controls and independent restored-frame regeneration passed in the same run. Local full web suite and vet also passed. No installer/version publication or Windows 10 certification; milestone 3 is not complete.


## Permanent deletion candidate

Hidden cards offer Permanently delete managed copy. The dialog names the item, describes the managed-copy/review-data scope and historical-backup/source exclusions, and requires typing DELETE. The API requires the content ID and original import timestamp, preventing an old dialog from deleting a new incarnation with identical bytes. Cancel before submission has no effect; after acceptance there is no undo/cancel-cleanup operation.

The operation nonblockingly acquires analysis, import, catalog-copy and media-decoder coordination gates and rejects an active import lease. It verifies the original before committing intent. A single durable catalog transaction records a bounded version-1 cleanup intent, removes the asset and its face/label/tag/analysis/proposal/queue records, updates gallery/batch indexes, and marks historical import receipts deleted. It then removes only exact hash-derived preview, face-crop and original filenames (no recursive deletion), clearing intent only after cleanup succeeds. External source aliases are metadata, never deletion paths. Existing backups are untouched.

Failed/interrupted cleanup stays visible in Hidden Media with Retry cleanup. Startup makes a bounded three-second recovery attempt; it does not prevent unrelated library access if permissions still refuse cleanup. While any cleanup is pending, new backups are refused; a pending ID cannot be reimported. Historical receipts cannot silently recreate the item; a new explicit import is allowed after completion and starts without old corrections. Manual-face/crop and reanalysis Apply publication recheck identity to reject stale work.

Older executables ignore the additive intent bucket. If an older writer reintroduces an asset while cleanup is pending, the new app refuses further deletion, shows a conflict, and offers Keep current copy to clear intent without touching that copy. Actual old-installer upgrade/rollback validation remains a combined-release gate; tests simulate older-writer catalog publication and check that recovery preserves the reappeared original.

Local tests cover original-integrity refusal before intent, visible/wrong/stale confirmation, cancellation before acceptance, active inference/copy/import/media/actual backup refusal, queued batch removal, deleted import receipts, explicit reimport, partial-cleanup/restart/missing-file recovery, older-writer conflict, retained external source and pre-/post-deletion independent backup restores with corrections. Focused Windows acceptance adds actual LocalService native photo analysis, dialog cancel/type/busy checks, narrow layout, deleted-cache refusal, NTFS delete denial, forced service death with a committed pending intent, restart/retry cleanup and independent restoration of the real pre-delete archive. No power-loss durability or secure erasure claim.
