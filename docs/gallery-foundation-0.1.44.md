# Frameout 0.1.44: baseline and shared gallery foundation

Scope: implementation steps 1 and 2 only. Viewer controls/zoom/full screen and
organization persistence are subsequent checkpoints. This is not a release.

## Released baseline

The public `v0.1.43-prototype` tag was independently resolved to
`90d86bd932234d703c4166e4c47585cfbfd047d5`. The accepted installer SHA256 is
`96670e90c4abc55cdc07f480267c6f896b784a2a6c11de9f9f5333edb4158415`,
104671997 bytes; the retained accepted artifact was hashed again and matches.
The 448 released tracked files were compared against their Git blob hashes.
All production, test, packaging and dependency files matched; seven historical
document differences in the local snapshot were accounted for. The separate
working copy uses the 450-file release-record tree at
`09a836e4e8794fe730d386cbca2e4cae46f01a52`. The three intervening commits change
only the publisher and release records, not application behavior.
The older development checkout and retained installers are preserved.

Regression requirements include date provenance/unknown capture timezone,
whole-catalog assigned-person/filename/active-tag substring search, automatic
name provenance/correction protections, visibility, downloads, bounded paging,
backup/restore and catalog 1/2 compatibility. No schema, dependency, provider,
model or installer change is needed by this browsing foundation.

## Query contract

`library.GalleryQuery.Normalize` is the only validation/normalization contract.
`QueryGalleryPage` and `GalleryNeighbors` share `galleryResultsFor`, its matching
set, ordering index and snapshot. The old GalleryPage/SearchGalleryPage entry
points delegate here. Total/hidden remain whole-library counters; count is the
matching count for the active query/visibility before paging.

- Sort: capture (existing import fallback), upload, case-insensitive filename;
  asset ID resolves ties. Visible and hidden results remain separate.
- Text: the released case-insensitive substring behavior across original name,
  active custom/AI tags and assigned non-dismissed names. Suggestions are excluded.
- Filename: a separate substring; MIME-based photo/video, with unknown files
  accessible through All files regardless of misleading suffixes.
- People/tags: exact case-insensitive full labels, Any/All/Exclude; fields AND
  together. Exclude rejects any hit; dismissed faces/AI tags do not contribute.
- Capture ranges use recorded calendar values, with no timezone invented and no
  import fallback. No date taken is separate. Import boundaries are explicit
  RFC3339 instants normalized to UTC. From is inclusive; Until is exclusive.
  The calendar UI will resolve inclusive display dates in its declared timezone.

HTTP: GET `/api/gallery`, GET `/api/gallery/neighbors`, both authenticated.
Shared fields are `sort`, `visibility`, `q`, `filename`, `media` (photo/video),
repeated `person_any/person_all/person_exclude`, repeated
`tag_any/tag_all/tag_exclude`, `date_basis` (capture/upload), `from`, `until`,
and `no_capture` (true/false). Pages add `after` and `limit` (1–100, default 60).
Neighbors require `asset` and the page's `snapshot`; at most current/previous/next
asset records are returned, without original bytes. Unknown fields, repeated
scalar fields and cross-query tokens return 400. Stale revisions/restarts return
409. Missing or ineligible current items cannot reveal hidden/out-of-query rows.

Cursors bind normalized query digest, sort, visibility, index epoch and ordering
revision. Filtered snapshots also bind the committed transaction, conservatively
invalidating on metadata writes. No unbounded browser list or media prefetch is
introduced. Filtered metadata cache retains no image bytes or embeddings; at
most eight cached membership sets are retained per transaction. Unfiltered pages
keep the existing index-only path and tolerate preview-only writes.

## Browser context and staged UI

One `galleryContext` supplies grid requests, filter address, page history and
the viewer's frozen query/snapshot/current item. Filename and media controls
exercise the contract now; the complete people/tag/date/calendar controls remain
step 8. Supported bookmarked fields are preserved in the address and browser
back/forward restores query/page context (stale pages recover explicitly).
Selection clears on query/sort/visibility changes with a message. Polling pauses
in the viewer, and sequence guards reject late grid/neighbor responses. Close
preserves the page/scroll and returns focus to the current card. The neighbor
contract is wired into viewer context; visible previous/next, page materialization
after navigation, keyboard/zoom/pan/full screen remain step 3.

Favorites, albums, captions, folders and import sessions will extend this same
query as their authoritative records and recovery support land. Unsupported
future fields fail explicitly rather than being silently ignored.

## Representative catalogs and validation

Reusable deterministic `seedGalleryFoundation` creates empty, single-item and
187-row mixed catalogs with tied timestamps/names, hidden items, unknown MIME,
missing/leap-day capture dates, Unicode/markup-like names, multiple relative
aliases, manual/automatic labels and dismissed tags/faces. A 50,000-row metadata
catalog checks the fast path and warm filtered paging allocations. These are
synthetic catalogs; no user media or machine inventory is used.

Tests check independent expected counts, date endpoints/DST offset boundaries,
exact Alice versus Alicia, normalized queries, all sort/visibility combinations,
page/count/neighbor agreement, endpoints, stale edits/hiding/reopen, invalid
queries and API authentication. Existing backup/rebuild/provenance tests remain.
The Windows workflow first executes the exact released baseline, then candidate
regressions/vet and actual LocalService/Edge journeys with synthetic originals.
Execution results are recorded below. Windows Server results do not certify
Windows 10 or installer/release acceptance.

## Accepted checkpoint — November 6, 2026

Steps 1 and 2 are complete on `work/release-0.1.44-foundation`. Final tested
application source is `3bc29f2292b52a4a1a52dd0af340776d53fbcefe`. The following
evidence is sequential and cumulative; no untested production changes followed.

| Gate | Exact source | Windows Server 2022 run / job | Result |
|---|---|---|---|
| Exact released 0.1.43 baseline, candidate regressions and first browser journeys | `7d42fba0389e6d099cc354c1eb51aabca0a69350` (baseline checkout separately pinned to `90d86bd932234d703c4166e4c47585cfbfd047d5`) | [42178779852](https://github.com/aataat12/immich-clone/actions/runs/42178779852) / 127763273123 | PASS |
| Final server query/API, all runnable retained internal/CLI tests, module verification, vet, date helpers and LocalService/Edge journeys | `3d5085232eafa751e9a88c3c66e865314b2f63ee` | [42179929183](https://github.com/aataat12/immich-clone/actions/runs/42179929183) / 127766930506 | PASS; no formatting diff |
| Final layout and browser journeys, real application rebuild and temporary LocalService execution | `3bc29f2292b52a4a1a52dd0af340776d53fbcefe` | [42180516257](https://github.com/aataat12/immich-clone/actions/runs/42180516257) / 127768792771 | PASS |

The final run reuses the proven baseline/server checks; its only production
change is search-field CSS. Browser tests additionally enforce readable field
widths (at least 140px), not just absence of horizontal overflow. Screenshots
were inspected at 1280px and 390px. A narrow search-field collapse found in the
first visual review was corrected before acceptance.

Malformed percent-encoded filters now return 400 instead of silently dropping
the filter. Import-boundary tests include an exact inclusive/exclusive minute,
and capture tests reject malformed values without an import-date substitution.
The final server run's synthetic 50,000-row metadata case measured a cold
filtered query at 207.0503ms and warm 60-row page allocation count at 1568,
under its 5000-allocation guard. This is one disposable-runner observation,
not peak-memory, media-ingestion, endurance or a general speed guarantee.

The real browser fixture uses 65 synthetic PNG originals. It proves released
search/date behavior, filtered page boundaries, viewer query/snapshot/focus/scroll
context, no polling while open, history/back/forward, reload, filename/media
combinations, stale edits, selection reset, errors, narrow layout and all original
hashes. The mixed-format representative catalogs are metadata fixtures; the full
actual decoder/provider matrix is separate evidence.

Accepted synthetic artifacts:

- First formatting/screenshots ZIP, artifact 13484168620: SHA256
  `d3ddf192238e871a71e71808c258fbb5b2766449fdce35c1c9902fc5083d2e2f`.
- Final server screenshots/empty formatting patch, artifact 13485072592: GitHub
  digest `457fb69d712c134d1fcd9bd83703a46d8a2c51994820f59900a77a8907065c0a`.
- Final browser screenshots, artifact 13485540602: SHA256
  `a73c4f499837ac43b5d48c6512fd32fb6df9cafbb4a8ab16cf7ef7ff10d0b088`.

First/final browser ZIPs were downloaded and independently hashed; the first
artifact's formatting patch matches the job-log patch after removing Git's
line-ending notices. The canonical patch was applied locally. Every uploaded
source blob was verified against its local Git object hash before branch updates.
The final server run proves that the resulting Go files need no further formatting.
Local date/script checks passed; 18 privacy-tool tests completed with one existing
platform-gated skip (17 passed).

Existing environment-gated tests remained skipped where they require additional
fixtures: actual native naming/quality and media providers, genuine older-writer
fixture, full-disk/mounted-volume cases, installed/large-volume/ZIP64 restores,
larger People-index acceptance, exact 4GiB HTTP boundary and separate-account
retirement actor. This foundation gate does not replace the 0.1.43 release's
broader acceptance or claim those special scenarios were rerun. Native naming,
providers/models, catalog 1/2 formats, installer and recovery implementation files
are unchanged. General catalog/rebuild/backup/restore/provenance tests passed.

Next checkpoint is step 3: visible previous/next across pages, keyboard behavior,
zoom/pan/full screen and materializing the current item's page on viewer return.
It should consume GalleryNeighbors and the frozen browser context rather than
reimplement matching. Metadata schema/recovery work remains step 4. No installer,
release publication, live PC installation or user-media operation occurred here.
