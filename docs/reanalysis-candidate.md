# Review-first additive reanalysis candidate

Status: implementation candidate; not yet in a released installer.

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

The focused Windows workflow includes actual bundled inference under LocalService and Edge browser controls using generated public fixtures. Pending execution: missing-model failure retention, HEIC proposal discard/apply, video pause/restart/discard/apply, narrow layout and the existing media recovery/independent restore checks. Windows Server 2022 execution does not establish Windows 10 compatibility. Combined installer upgrade/rollback validation remains a later release gate.
