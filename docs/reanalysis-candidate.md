# Review-first additive reanalysis candidate

Status: released in 0.1.33-prototype after combined cloud installer acceptance.

For an already completed supported photo or sampled MP4/MOV, **Prepare reanalysis proposal** runs the currently bundled models into a separate durable job. The completed analysis and manual review remain available. Preparation does not change existing face records, labels, custom tags, AI-tag dismissals or originals.

The proposal shows new-face, retained-face and skipped-overlap counts, plus AI object suggestions. **Apply** commits additional detections and refreshed AI object suggestions atomically. **Discard** deletes the pending proposal results. There is no automatic application or name assignment.

## Conservative first mode

This is additive reanalysis, not replacement of prior face geometry or embeddings. Any positive box overlap with an existing observation at the same actual video timestamp is skipped, including dismissed observations and manual faces. Existing IDs, names, dismissal history, embeddings and geometry remain intact. An overlapping legitimate new face may therefore be skipped; existing manual face marking remains available for supported stills. Counts are recalculated against current manual edits at Apply time.

Videos retain the existing up-to-12-position sampling policy, actual decoded timestamps, deduplication and per-frame checkpoints. Pause takes effect after the in-flight frame; resume survives restart. A fresh generation token prevents discarded or superseded work from publishing. No every-frame Full Scan, new model, native dependency or automatic background rescan is included.

## Persistence and limits

- Active proposals are limited to 128; each uses existing inference limits (at most 64 faces per frame, 12 video positions, 80 object categories).
- Import and ordinary requested analysis take priority over proposal work.
- Each preparation step has a 75-second bound. Failure preserves the old completed analysis and the prior successful proposal checkpoint. Retry is explicit.
- Apply verifies the original hash before its atomic catalog transaction; staged vectors are never exposed through browser JSON or catalog JSON export.
- Application-consistent backups retain proposals and vectors for independent restore; discard clears staged inference data. Existing face crops are unchanged; additional crops are derived lazily.

## Acceptance evidence

Local tests cover preview/apply preservation, repeated-run deduplication, failure, pause/reopen, discard during inference with a new generation, video checkpointing, independent backup restore and edits made after proposal creation. HTTP tests cover sanitized proposal responses and the existing authenticated route/origin guard matrix.

The focused Windows workflow includes actual bundled inference under LocalService and Edge browser controls using generated public fixtures. Passed on Windows Server 2022 with Go 1.27.3, actual bundled providers under LocalService, and Edge/Playwright 1.62.1: missing-model failure retention, HEIC proposal discard/apply, video pause/restart/discard/apply, no duplicate faces after same-model rerun, preserved names/custom tags, narrow layout, decoder/storage/model failures, service termination during native inference, and independent restored-frame regeneration. Windows Server 2022 execution does not establish Windows 10 compatibility. Combined installer upgrade/rollback validation passed in the release run below.

## Public Windows run history

- [41650178334](https://github.com/aataat12/immich-clone/actions/runs/41650178334), commit `c88d297cbc14352ca537c8c41336aebacda5d22f`: focused Windows tests passed; actual service passed missing-model retention and HEIC discard/apply plus video pause/restart setup. The browser finish phase timed out because its next click raced the preceding Resume action. This run did not establish final video Apply or independent restore acceptance.
- Corrected test synchronization waits for the POST response and completion of the existing UI action; an explicit cancelled-state assertion was added. No application behavior or assertion was weakened. [41651264618](https://github.com/aataat12/immich-clone/actions/runs/41651264618), commit `46705cba0f60f9ada6842937dd33370838bf8440`: PASS. All focused Windows checks, three reanalysis browser phases and the complete existing media recovery/independent restore harness passed (job `125969738235`). No installer was built or published by this focused workflow.

## Combined 0.1.33 installer acceptance

[41652777887](https://github.com/aataat12/immich-clone/actions/runs/41652777887), source `73b8e9b910633b8525097d18e2eae37a2ecc1d1c`, job `125975035550`: PASS on the first attempt, Windows Server 2022/Edge. Full suite/vet/module and vulnerability gates, bounded real NTFS full-disk recovery, application/browser/backup restoration, actual hash-verified 0.1.32 installer upgrade, both injected failed-upgrade rollback paths, installed HEIC/video reanalysis controls, narrow layout, payload canary scan, uninstall preservation and independent restored reinstall passed. An unapplied proposal survived rollback and independent restore, then applied while retaining face IDs with zero duplicates. The known restore progress-read:32 intermittence did not recur in this run; its cause remains unresolved. No claim of Windows 10 certification or a security-backlog closeout.
