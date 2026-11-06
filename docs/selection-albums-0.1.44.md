# 0.1.44 selection, favorites and manual albums

Steps 5–6 accepted November 6, 2026 on disposable Windows Server 2022 runners.
This is a development checkpoint, not a released installer. It extends the
[steps 3–4 foundation](viewer-metadata-0.1.44.md).

## Selection and reviewed batches

Select items and Select this page collect up to 500 explicit items across pages
of the current results. The selected count persists through paging and viewer
open/close. Query, sort, visibility or album changes clear temporary selection;
reload also clears it. There is no implicit all-results action.

The action toolbar supports set/unset favorite, add/remove a custom tag,
reversible hide, and add/remove membership in an explicitly chosen album.
Review exact targets freezes IDs, incarnations, names and relevant organization
state. A stale browsing snapshot may be refreshed for review only when every
explicit target still belongs to the same query and incarnation; an invalid or
changed query token is rejected. The preview identifies this recheck and lists
every target and action. Individual targets, including missing items, can be
removed through the Selected items list before another preview. Existing
hidden-media restore and permanent-delete controls and confirmations remain
separate.

Each explicit continuation commits at most 25 targets and their outcomes in the
same bbolt transaction. Outcomes are saved, unchanged, conflict, missing and not
attempted; canceled targets remain explicitly not attempted. Conflicting targets
are not overwritten. A transaction/storage error
rolls back the current chunk; earlier committed chunks remain saved. Stop waits
for the current chunk. Cancel remaining targets marks pending work terminal
without changing media or undoing saved successes, so canceled receipts can retire.
No background worker resumes operations after browser closure, service restart
or backup restoration.

Recent operations / recovery opens a retained receipt. Continue processes pending
targets only; Review unresolved targets again requires a fresh preview before
retrying conflicts or missing items. Missing items must be removed from the new
selection. Continuations carry the processed-target count: repeating a request
whose response was lost returns the recorded state without advancing again.
Unknown or expired operation IDs return 410 and cannot start new operations.

Receipts are available to reopen for 30 days, maximum 64. Expired records are
pruned when preparing a new operation; they may remain physically stored until
then, and existing backups retain their saved history. At capacity, the oldest
completed receipt may retire sooner; unexpired unfinished work is not evicted
to accept another operation. History lists summaries; exact targets are fetched
for one receipt. The optional `organization_batches_v3` bucket is part of catalog
3 validation, export and the existing 64 MiB organization budget. Preparing a
durable reviewed operation is itself a real catalog-3 write, even before media
organization changes. No receipt restoration auto-replays mutations.

`GET/POST /api/organization/batches` lists summaries or prepares a reviewed
operation. `GET /api/organization/batches/{id}` reads its exact receipt. `POST`
to that ID accepts `{action:"continue",step:<processed count>}` or the same
shape with `action:"cancel"` to abandon pending work without undoing successes.
These routes use the existing authentication, request-header and bounded-JSON
protections.

## Favorites and albums

Grid and viewer buttons explicitly set the displayed intended favorite state;
they preserve other user text and reject revision conflicts. Favorites and Not
favorites are shortcuts into the same gallery query. Album and favorite fields
therefore govern matching counts, pages, anchors and viewer neighbors together.
The viewer waits for current favorite state before enabling its button, guards
late reads and offers Refresh results after a failed metadata read.
Grid responses include only the bounded favorite flag needed for display.
Filtered snapshots fingerprint the shared matching/order inputs, including
organization state, instead of unrelated database transaction IDs. Preview
progress and receipt bookkeeping do not reset paging; changed organization,
labels, tags, membership and order invalidate it. The derived cache is published
only from committed read transactions.

Flat manual albums support stable IDs, title, description, empty albums,
many-to-many membership and covers. Album summaries show visible and hidden
counts. Hidden members remain stored and return when restored to the library;
ordinary galleries and cover thumbnails omit them. A hidden/removed cover uses
a visible fallback. Removing membership or an album changes organization only:
media and membership in other albums remain intact.

Album manual order is available alongside the existing date/filename sorts.
Normal sorting never rewrites saved manual positions. Move earlier/later uses
the neighboring item in the shared manual result order and checks the album's
revision and incarnation. A recreated album cannot receive a draft or batch
prepared for its earlier incarnation. These controls are available only in an
unfiltered visible manual album view; clear search and other filters first.
Gallery pages, reverse cursors,
viewer neighbors and return anchors all follow the selected order.

Album create/edit/removal is available through Create album, Albums and Edit
current album. Removal requires an explicit organization-only confirmation.
Existing metadata API bounds and conflict tokens remain in effect. Caption/source
browsing, calendar navigation, broader filter interfaces and final combined
release acceptance remain later steps.

## Validation record

| Gate | Exact evidence | Result |
|---|---|---|
| Full regressions, vet, module verification, date helpers, actual bounded NTFS capacity failure/recovery | [Run 42254361898](https://github.com/aataat12/immich-clone/actions/runs/42254361898), foundation job `128017487491` | Passed |
| Actual LocalService/Edge, metadata/export, independent HTTP backup → CLI restore with original state unavailable | Same foundation job | Passed |
| Genuine 0.1.43 positive controls/refusal, real Setup upgrade and rollback before/after version-3 commit, independent restored enrollment | Same run, installed-metadata job `128019993984` | Passed for starting catalogs 1 and 2 |
| Final mixed photo/H264 album, UI visibility, favorite-read retry, stale displayed intent, selection and recovery journeys | [Run 42259007001](https://github.com/aataat12/immich-clone/actions/runs/42259007001), foundation job `128033719795` | Passed |

The full backend/recovery checkout was
`a491783bbe86c08b9f1b4ff3e7b18aa84bdbf865`, tree
`a7299638a8efedc703cb4d46e33af8beea254130`. Both jobs applied the same formatter;
their downloaded canonical patches were identical and applied locally. The
final browser checkout was `6e51ef06b41869136f21a9b9d9b1f8c96828508e`, tree
`55c4f550a81907b9b40934a8639fa2f25f9f966f`. It asserted no Go formatting diff
before compiling the real service. Backend source is the same accepted canonical
Go; its full regression/NTFS/installed checks were reused. The final browser
adds verified frontend loading/visibility behavior and stronger journeys.

The final record adds documents and a display-only target-album placeholder
correction after album deletion. The empty request value remains unchanged;
bundled browser script syntax was checked. No additional backend behavior changed.

Demonstrated cases include 131 explicit targets across three pages, off-page
individual removal, viewer selection retention, query/scope clearing, exact
preview identities, a committed first chunk with its HTTP response deliberately
discarded, actual service restart with work paused, Stop/Continue, terminal
cancellation, repeated requests, stale targets, receipt capacity/retirement,
whole-chunk budget rollback and actual NTFS reserve refusal/retry. Custom-tag
results were checked before/after removal, alongside unchanged original tags
and recognition/correction provenance. A pending favorite-removal operation was
included in the independently restored graph and remained paused.

Album checks include empty albums, one item in two albums, duplicate add,
cross-page manual moves and viewer return, ordinary sorting without order loss,
actual H264 playback/release within a manual album, explicit covers and hidden
fallback, hide/restore, membership removal, album removal, and SHA256 verification
of every synthetic original. Grid/viewer favorites preserved another client's
new caption and honored the displayed set intent. Desktop 1280px and narrow
390px organization and viewer screenshots were inspected.

The 50,000-row metadata fixture retained its bounded pages and warm allocation
guard: one observed cold query was 232.7473 ms and warm page allocation count
1886. This is synthetic metadata evidence, not media ingestion, worst-case album
performance or a latency guarantee. The older anchor-scale fixture still targets
the first matching item, not a worst-case final-item seek.

Verified downloaded artifact identities:

- Full synthetic evidence `13521453925`, ZIP SHA256
  `e8e183c3377adbbc11433c1c0639ae7878af5a2ead45f8f3fcc91f62a1732e7b`.
- Installed recovery identity `13523271254`, ZIP SHA256
  `a59d5f83cc923b6253a8a23babf5f934f2da46505e15dc67d4b95048ab6afa65`.
- Final browser evidence `13524710584`, ZIP SHA256
  `4f0605e9b9534d9939b4d41ef17305464a7f41d991a73a727bd5600bca848429`.
- Identical full-job canonical patches: raw SHA256
  `2e5bd2624087d0375c118e6c640b5d88c8680123b3afcdfd13926f443a6b02d8`;
  LF-normalized applied patch SHA256
  `f037cdf61e40afe72e1ebf40a965f138f2665b106bf4f15193a1842ec0957d23`.
- Internal recovery installer at the full checkpoint: SHA256
  `05a2c2882a69ab7a1cf89de8e8f650f7ab021c97fce03549acf88d1559bb7ed9`.
  This is the tested internal backend/recovery package, not a published final UI
  installer. The later UI refinements were exercised by the final service/browser
  gate. The genuine old installer and released baseline identities remain recorded
  in the steps 3–4 acceptance record.

Earlier attempts caught an obsolete favorite-query rejection fixture,
transaction-wide snapshot invalidation by unrelated writes, an uncommitted cache
publication risk, asynchronous return/move timing, and native hidden attributes
overridden by button styling. Those were corrected before acceptance. The final
concurrent-client browser assertion freezes the displayed state with selection,
so automatic refresh cannot change the intended test action before the click.

No new product dependency/provider/model was introduced. Existing pinned bundled
providers were reused for Setup recovery. Environment-gated native quality,
broader formats, large-container/ZIP64, long endurance, larger People-index and
full release/Windows 10 studies were not rerun for these slices. Durable batch
receipts are historical organization records; original deletion does not rewrite
past receipts or existing backups. Their availability and retirement follow the
policy above.

All fixtures use synthetic media and disposable runners. No user-library
installation, public installer release, new product dependency/provider/model,
Windows 10, reboot, endurance or signing acceptance is implied.
