# 0.1.44 captions, recorded sources and combined calendar browsing

Steps 7–8 extend the accepted [selection/album checkpoint](selection-albums-0.1.44.md).
This record tracks development acceptance, not a published release.

## User workflows

Titles, captions and notes can be viewed and edited from a gallery card or the
viewer. The catalog keeps them separately from originals and recognition data.
All three fields participate in case-insensitive gallery text search. Grid cards
show the bounded title; full text appears in the viewer. The editor reads current
revision/incarnation tokens, rejects a stale write and keeps the unsaved draft.
Cancel or Escape with unsaved text keeps the draft and offers an explicit Discard changes and close action. Text fields are disabled while initial reads and saves are in flight. Reload saved text deliberately replaces that draft. Favorite edits retain text;
text edits retain the favorite state they read and refuse concurrent changes.
Existing catalog-3 text limits and the backup format remain in force. JSON export adds retained import history as described below.

People, tags, dates & sources provides exact-name Any, All and Exclude groups.
Groups combine with each other and with text, filename, media, favorites, album,
visibility, relative folder, recorded batch and dates. Apply filters commits a
single browsing query. Active-filter buttons remove a group; Clear all filters
retains visibility and an applicable ordinary sort. Query state survives address
reload and Back/Forward. Changing it clears temporary selections. Pages, counts,
selection review, viewer neighbors and return anchors use the same query.

Recorded sources is a virtual browser of catalog paths. A folder includes its
subfolders by default, with a slash boundary so Trip excludes TripTwo. This folder only uses exact recorded parent directories; breadcrumbs and Parent folder navigate the virtual tree. Direct root scope shows files recorded at the virtual root. No recorded folder selects items without any relative alias in the chosen batch or across all batches. All folders includes
assets without a recorded source; Any recorded folder selects recorded relative
paths, including a file recorded at the virtual root. Folder lists show immediate
children and at most 100 per browser request (API maximum 200). More folders pages
through them. Parent folder and active-filter removal provide direct return.
Paths are case-sensitive recorded aliases. Equal relative paths from different source roots merge; those records cannot reconstruct a drive or distinguish legacy roots. No external source filesystem is scanned, opened, changed or hydrated by this feature.

Recorded import batches use existing durable receipts, including duplicates,
counting each extant original once. A combined folder/batch query uses that batch's
own recorded paths. An alias from another batch cannot satisfy it. Pending,
failed or deleted entries, and asset records no longer present in the catalog, do not acquire successful membership.
Canceled/partial batches retain accepted members; status, unique matching counts and separate total/accepted/new/duplicate/failed/deleted/pending file outcomes are shown. Accepted/new/duplicate counts retain historical successful receipt outcomes, including later deletion; failed and pending describe never-accepted entries. A syntactically valid unknown batch ID yields an empty gallery. Receipts
have the existing 128-session retention/capacity bound, not an unlimited new
import-history promise.

Calendar offers direct year, month and day choices with counts under the other
active filters. Opening another date replaces the current date restriction.
Date taken uses the recorded calendar value without inventing a capture zone;
missing or malformed dates have an explicit Missing date taken choice. Date
imported uses the browser's IANA time zone. Server-generated bucket boundaries
handle leap days, daylight-saving days, missing midnights and skipped dates.
The standard Go time-zone database is embedded for Windows IANA lookup; there is
no new downloaded runtime/provider/model. From/Through custom dates use inclusive
calendar-day input and the shared inclusive-from/exclusive-until query contract.

## Service interfaces and preservation

`GET /api/gallery/sources` accepts the shared query plus `source_after` and
`limit`; it returns folder counts, paged children and bounded batch choices.
`GET /api/gallery/calendar` accepts the shared query plus `calendar_basis`,
`calendar_zone`, optional `year` and optional `month` with a year. Date restrictions
are removed for its counts; all other filters remain. It returns year/month/day
counts and exact `from`/`until` boundaries. Unknown, ambiguous or invalid fields,
absolute/traversing paths and unsupported methods are refused. Source queries also accept `source_mode=direct` and `source_missing=true`; missing-source scope cannot be combined with a nonempty folder.

Derived matching state includes source aliases and accepted receipt provenance,
so source-only edits invalidate stale page/viewer snapshots. Read caches are
published only from committed read transactions. No new authoritative catalog
bucket, migration or original mutation is introduced by source/calendar browsing.
JSON export adds a versioned `import_history` section with exact recorded session metadata and index-keyed receipt entries. Export refuses history beyond its 64 MiB aggregate raw-receipt/key budget without returning a partial success; existing source/backup state remains unchanged. This is an interchange record, not a new JSON-import workflow.

Existing captions already use catalog 3, including export, independent restoration
and older-writer refusal established by steps 3–6.

## Validation

Accepted November 6, 2026 on disposable Windows Server 2022 runners.

| Gate | Evidence | Result |
|---|---|---|
| All runnable internal/CLI Go regressions, vet, module verification, date helpers and actual NTFS capacity refusal/retry | [Run 42274949273](https://github.com/aataat12/immich-clone/actions/runs/42274949273), foundation job 128088765041 | Passed |
| Actual LocalService/Edge caption, folder/batch, combined filter/calendar, history, selection and existing viewer/album journeys | Same foundation job | Passed |
| Actual service restart; HTTP backup followed by independent CLI restore with source state unavailable; original SHA256 and restored query/receipt graph checks | Same foundation job | Passed |
| Genuine released 0.1.43 reader/writer refusal; real Setup upgrade/failed-upgrade rollback before and after catalog-3 writes, for starting catalogs 1 and 2; installed independent restored enrollment | Same run, installed-metadata job 128091167845 | Passed |
| Existing bounded synthetic catalog/PNG worker probe | [Run 42274949254](https://github.com/aataat12/immich-clone/actions/runs/42274949254) | Passed |

The tested checkout was `8d296abcf78cedc2a2ed30d9d13ae0c0cf6d6579`,
tree `d36ac4574e81bfb3b892a12fdd7b669cee6ad0dd`. Both jobs ran the same
explicit Go formatter. Their downloaded canonical patches are byte-identical;
that exact formatting was applied and saved locally. This record commit adds
only the focused acceptance record. Public implementation source remains the
tested checkout; the documented formatter supplies the accepted build input.

The browser checks demonstrate Unicode/markup-like text, stale-write refusal
with draft retained, explicit reload/discard, editable-field locking during
reads/saves, text search updates, title display and viewer text, duplicate aliases
in two sessions, same-batch source provenance, direct scope and empty intermediate
navigation, breadcrumbs, no-alias originals, import outcome labels and receipt JSON.
They combine assigned people, All/Exclude tags, favorites, album/manual order,
folder, batch and an import-day bucket, then exercise address reload/Back/Forward,
selection clearing and delayed dialog responses. Desktop 1280px and narrow 390px
caption-editor, source-browser and combined-filter screenshots were inspected;
no horizontal overflow was observed.

Library/HTTP fixtures additionally cover Any/All/Exclude semantics inherited
from the shared-query foundation, matching counts/pages/anchors/neighbors,
missing aliases globally versus within a batch, deleted/failed/pending receipts,
folder-prefix boundaries, hidden/visibility semantics, source-only snapshot
invalidation, leap day, Chicago 23/25-hour days, missing midnights/skipped dates,
historical second-resolution zone offsets and earliest-year query validity.
Export tests compare exact persisted JSON before/after transaction closure and
independent restore, exercise aggregate-budget refusal with a smaller injected
budget, and reject an over-1-MiB raw record through the real production guard.
This is not an RSS measurement or a 64-MiB history endurance test.

Verified downloaded identities:

- Service/browser synthetic evidence artifact `13532322802`, ZIP SHA256
  `bffb05a0a2b1533b2f123e54d4e468e6d90b8ba4b944a9c2fc38efb603d32599`.
- Installed identity evidence artifact `13532418664`, ZIP SHA256
  `3dc32b4a110823bf31c3a70e0834d6f2ab9792b1e3910ee409c5d2579571cb65`.
- Identical canonical patches: raw SHA256
  `edcc2490be1200b38301bd6dcaf3539ceea5322bd9b4cb98e337d193bc35a310`.
- Internally executed test installer SHA256
  `ac542fb62d665d666ed6d4d62452eb616c4a3010079eab12e6bd9240566d147e`.
  It was exercised on disposable runners; no public installer release was made.

Earlier attempts exposed two browser-fixture errors (a variable used across
process boundaries and cleanup of a still-running delayed route). They were
corrected without weakening the source/filter/draft assertions. Review also fixed
stale viewer text, dialog responses after history changes, URL ordering, typing
while metadata loads/saves, exact historical zone boundaries and complete
source-scope/outcome/export coverage before this accepted run.

Next is step 9's broader combined-experience acceptance, including larger
source/history-heavy catalogs, 100k combined calendar/query measurements and
extended concurrent/interruption/endurance scenarios, followed by step 10's exact
final installer/release delivery. Environment-gated native inference/quality,
broader format/large-container studies, final Windows 10/clean-machine/reboot,
signing and full-library endurance were not certified here. No user library or
private machine report was used. Existing pinned providers/models and product
module versions remain unchanged.
