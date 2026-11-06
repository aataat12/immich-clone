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
Execution results are recorded separately after the gate completes. Windows
Server results do not certify Windows 10 or installer/release acceptance.
