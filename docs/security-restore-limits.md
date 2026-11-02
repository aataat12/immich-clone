# SEC-04: hostile backup and catalog resource limits

Original item: **Hostile backup limits**. This implementation bounds the whole
untrusted restore operation, including ZIP directory parsing and bbolt structural
checking/index rebuild, rather than only limiting individual archive entries.
It preserves the schema-1 ZIP/manifest/original layout and independent recovery.
Windows execution and independent verification are required before closure; source
readiness and cross-compilation alone are not acceptance.

## Implemented operation

`Restore` now calls `RestoreWithLimits(context, archive, target, RestoreLimits{})`.
The parent creates private staging beside a new destination and a bounded nonce
request. It invokes the same executable in one narrowly guarded internal worker
mode; no external interpreter, install prerequisite or arbitrary-command option
is added. The internal entry accepts only the exact request filename, matching
nonce and generated staging path. Unit-test executables use the same dispatch,
not a separate unbounded validation path.

The worker performs **all** ZIP opening/directory parsing, metadata/entry checks,
manifest parsing, extraction/checksums, read-only bbolt Open/Check/Close, normal
catalog Open/reindex/list/original verification, preview retry transactions and
final stage traversal/Close. A panic, malformed/cyclic catalog or ignored in-process
cancellation cannot trap the parent in bbolt work: the parent supervises a separate
process. It never opens the catalog or traverses untrusted ZIP metadata itself.
The existing strict names/modes, duplicate rejection, manifest EOF/CRC, SHA256 and
catalog/original relationship checks remain.

Before extraction, checked arithmetic sums every declared uncompressed entry,
including manifest bytes, and central-directory header/name/extra/comment costs
(including the archive comment). There is no wrapping addition. An independent
cumulative writer enforces actual copied bytes even if sizes or decompression are
hostile. It never writes beyond the selected remaining extraction budget.
The original-blob 4GiB ceiling is unchanged. Catalog and manifest limits are
separate configurable recovery budgets, not permission to upload larger originals.

## Finite defaults and explicit recovery overrides

Zero fields select these defaults independently; negative timeout and overflowing
or invalid settings fail before child execution. All numeric CLI values are bytes
or counts, never inferred GiB units.

| RestoreLimits field | CLI flag | Default |
|---|---|---|
| MaxBytes | `-restore-max-bytes` | Current destination available bytes minus 64MiB reserve |
| MaxEntries | `-restore-max-entries` | 100000, retaining the previous restore default |
| MaxManifestBytes | `-restore-max-manifest-bytes` | 16MiB, retaining the previous default |
| MaxCatalogBytes | `-restore-max-catalog-bytes` | 4GiB, retaining the previous per-catalog default |
| MaxMetadataBytes | `-restore-max-metadata-bytes` | 64MiB of central ZIP metadata |
| MemoryBytes | `-restore-memory-bytes` | 512MiB worker committed/private-memory budget on Windows |
| Timeout | `-restore-timeout` | 30 minutes for the whole worker |

Aggregate capacity is tied to the destination, not a fixed total-library size.
An explicit aggregate budget must fit the current free-space-minus-reserve check;
operators can raise other finite budgets for a larger legitimate recovery. The
existing backup-creation and original-upload bounds are not silently widened.
Larger full-library recovery/throughput still needs its own actual acceptance;
these controls do not establish that a particular large library fits the defaults.
The Windows memory setting must be at least 32MiB; raising it changes the explicit
resource allowance, not the immutable backup format.

CLI examples for an operator-selected larger recovery can add, for example,
`-restore-timeout 2h -restore-memory-bytes 1073741824` to the normal restore command.
If a valid catalog itself needs a larger input/rebuild allowance, explicitly raise
`-restore-max-catalog-bytes` too, within available aggregate capacity. Do not
increase all budgets automatically after a hostile or corrupt archive is rejected.
Fixed CLI categories distinguish configured byte/count/capacity budget refusal,
hard wall timeout, explicit cancellation, archive validation failure, worker
memory/process/output failure and cleanup requiring attention. They reveal no
raw catalog errors, private paths, access codes or child stderr.

## Hard worker boundary and bounded supervision

Windows uses the same already-executed Job Object creation pattern as the media
launcher: assignment at process creation, active-process limit one, kill-on-close,
and configurable process/job committed-memory limits. A context deadline or
cancellation terminates the whole Job and confirms process exit before cleanup.
Worker stdout is capped at 4096 bytes; output flooding terminates the Job. Stderr
is discarded. The memory budget is a committed/private allocation limit, **not**
a 512MiB RSS/working-set or filesystem-cache guarantee. File-backed catalog mappings
also have the independent catalog-byte budget below. A soft Go memory target is
supplementary, never described as the Windows hard enforcement.

Linux is a cloud development/test host: separate-process timeout, process-group
termination/reaping, output and restore-budget tests execute there. Its Go memory
target is soft; Linux results are not claimed as proof of the Windows Job memory
boundary. The product has no Linux prerequisite. Actual Windows memory-exhaustion
controls must prove the hard Job limit independently of compilation.

The parent checks worker liveness at a 50ms wait interval on Windows, kills on
cancellation/deadline and waits up to five seconds for confirmed exit. That is not
a promise that arbitrary kernel I/O cannot delay termination. If death cannot be
confirmed, it fails closed without publication and retains private staging rather
than deleting files beneath a potentially live writer. Normal confirmed failures
remove staging; a cleanup failure is surfaced instead of silently hidden.

## Catalog growth and final output

Catalog Open/reindex/retry can grow a file after extraction. Before these operations,
the worker atomically writes fixed-size nonce-bound progress containing the known
non-catalog/manifest byte cost. The parent reads at most 4097 progress bytes and
stats only `catalog.db`; it never calls catalog Open or performs an unbounded final
walk. During processing it checks:

- current catalog size <= MaxCatalogBytes;
- known non-catalog/manifest cost + current catalog size <= MaxBytes.

The polling period is pinned at **20ms**, plus OS scheduling/I/O and termination
latency. Exceeding a bound cancels the supervised worker, then requires kill/wait
before cleanup. This is bounded-latency detection, **not an atomic disk quota**;
a write can cross the chosen threshold between observations. Final catalog/progress
checks run again before parent publication, while the worker performs the complete
final output-size walk within its hard wall/memory boundary. Trusted request and
progress files are removed before rename and are not published in restored state.

The extraction budget counts manifest read bytes plus payload writes; catalog
processing additionally accounts for the current catalog size replacing its input
size. Final actual stage files must fit the selected aggregate budget. Temporary
protocol files and filesystem metadata have small separate overhead within the
reserve; free-space checks cannot reserve capacity against competing applications.
ENOSPC/permission failures still reject and clean unpublished staging. No free-space
or maximum transient overshoot guarantee is made.

## Required acceptance and closure

The independent verifier owns `restore_limits_test.go`, an actual Windows CLI
acceptance script and verification notes. Required observations include:

- valid schema-1 independent restore after source isolation, preserving original
  hashes, ready results, names/undo/dismissal, tags and access code;
- high-compression valid backup accepted within a sufficient explicit budget,
  rejected under a small aggregate budget without publication or outside mutation;
- entry/manifest/metadata/catalog limits, ZIP64/header arithmetic overflow,
  malformed/cyclic bbolt with a valid outer SHA256 manifest;
- actual cumulative writer exhaustion and live catalog-growth cancellation;
- pre-start and after-staging cancellation with no target/staging retention;
- exact supervisor success, noncooperative timeout/cancel, output flood, process
  death confirmation, and actual Windows small-vs-excess committed-memory control;
- actual installed executable CLI default/override success and low-budget refusal,
  followed by metadata/original verification, plus existing normal regressions.

Initial focused Linux tests and race checks passed, along with vet and Windows
cross-compilation. Those do not close SEC-04 until actual Windows Job/CLI execution
and root review confirm the complete matrix. Parent-folder swap races remain
SEC-03; same-token filesystem authority remains SEC-05; broader input review,
combined accepted-old-installer/restore, clean Windows 10 and long endurance remain
their separate gates. No product installer or broad security certification is
created merely by passing these resource controls.
