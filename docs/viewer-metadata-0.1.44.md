# 0.1.44 steps 3–4 acceptance

Accepted November 6, 2026 on disposable Windows Server 2022 runners. Steps 3
(viewer navigation) and 4 (metadata storage, compatibility and recovery) are
implemented. This extends the accepted steps 1–2 in
[gallery-foundation-0.1.44.md](gallery-foundation-0.1.44.md).

## Viewer behavior

Previous/Next follows the grid's normalized query, ordering and snapshot across
page boundaries without wrapping. Closing materializes the current item's grid
page and restores its card focus and scroll position. Reverse paging uses server
cursors, including after a direct page-anchor jump.

Visible controls provide image zoom, pointer pan, Fit, full screen and Close.
Shortcuts are Left/Right, +/−, 0 for Fit and F for full screen. Escape exits full
screen before closing. Editable fields, video controls and browser modifier
shortcuts retain their keys. Navigation resets image transforms; navigation,
close and browser Back stop and release video. Unsupported or unavailable
renditions retain information/download and navigation controls.

The viewer freezes its query/snapshot, suspends periodic grid refresh, and guards
late responses. Changed results require explicit refresh; return re-anchors an
eligible current item or explains its fallback. Hidden items do not return to
the ordinary grid. No full-catalog browser ID list or original-media sequence
prefetch was introduced.

`GET /api/gallery/anchor` shares matcher and snapshot validation. It returns at
most the requested page (default 60, maximum 100), page number and cursors.
Locating an anchor can traverse metadata index keys on the server. Filtered
reverse pages reuse a bounded number of cached query-key sequences; ordinary
unfiltered pages retain their indexed path. The 50,000-row fixture checks a
bounded first-item anchor; it does not establish worst-case last-item latency
or full-media ingestion performance.

## Durable organization contract

Catalog 3 stores authoritative asset favorite/title/caption/notes records,
albums (including empty albums), covers and ordered many-to-many membership in
separate bbolt buckets. These records do not alter original files or replace
recognition provenance. Managed-copy deletion cleans affected organization and
membership in its existing deletion transaction. Album deletion preserves media.

The candidate reads catalogs 1/2/3. Browsing and no-op edits do not promote old
catalogs. The first actual new organization write atomically creates the new
buckets and promotes to 3. Automatic naming/correction helpers preserve 3;
clearing organization never downgrades it. Invalid versions and inconsistent
relationships are rejected before writable initialization or restore publication.

Authenticated routes are `GET /api/metadata/asset/{id}`, `GET /api/metadata/albums`,
`GET /api/metadata/albums/{id}` and `POST /api/metadata/edit`. Album pages contain
at most 100 members. Edits require the expected record token, incorporating
revision, open epoch and asset incarnation. Restart, restore and delete/reimport
invalidate stale drafts. Album creation accepts a stable client ID; adding an
existing member with a fresh token is a no-op. `set_asset` replaces
favorite/title/caption/notes together: clients must fetch, merge their intended
change and send the expected token. Conflicts must be reviewed before retrying.

| Limit | Bound |
|---|---:|
| Asset/album title | 256 UTF-8 bytes |
| Album description | 4096 UTF-8 bytes |
| Caption / notes | 16 KiB / 64 KiB |
| Albums / members per album | 1000 / 5000 |
| Total memberships | 100000 |
| Serialized organization values | 64 MiB |
| Individual record / edit request | 1 MiB / 1 MiB |

Metadata mutations require a 16 MiB free-space reserve before database growth,
matching recoverable deletion. Existing backup/restore bounds remain in force.
Full selection/batch operations, favorites/album interfaces, caption editing and
integration of new fields into gallery search remain later work. This checkpoint
establishes their durable foundation and authenticated access.

## Export, compatibility and recovery

Organization export retains existing fields and adds a versioned `curation`
section with asset records, empty albums, descriptions, covers, membership and
manual order. Transient edit tokens are omitted. JSON export is interchange;
this does not add JSON import. Whole-catalog backup independently preserves the
authoritative graph and originals. Restore validates the graph within existing
bounds and uses the existing index/rendition recovery path.

The genuine published 0.1.43 executable positively opened the starting catalog-1
and catalog-2 fixtures. On catalog 3, its offline verify/backup and LocalService
writer refused without source-byte or namespace changes. Intentional downgrade
therefore requires a compatible earlier backup and prior installer.

Actual candidate Setup upgrades were exercised from both starting formats. For
each, an injected failure before new metadata and another after an observed
authenticated catalog-3 commit restored the exact saved prior executable and
catalog bytes, checked before old-service index rebuilding. The prior service
then became healthy; access code, existing originals and an original imported
after the snapshot were preserved. Successful upgrades, backup, independent
restore with the source state unavailable, actual restored Setup enrollment and
uninstall also passed. Manual/automatic face provenance was compared alongside
the complete new organization graph.

Installer observation windows activate only when both existing failure injection
and the additional curation test flag are enabled. They are bounded test delays;
ordinary Setup has no additional delay or validation bypass.

## Exact evidence

- Tested checkout: `0084da11470c8aef791526369b2b72b7981a6683`, tree
  `88bf01e1a1a932ed6d3b72ab95a599585de6976e`. Both jobs apply the same explicit
  Go formatter. Their identical canonical patches were downloaded and applied
  locally; the final record commit contains only that formatting and documents.
- [Accepted run 42226330751](https://github.com/aataat12/immich-clone/actions/runs/42226330751):
  foundation job `127920654634` and installed-metadata job `127922562228` succeeded.
- Foundation: all runnable internal/CLI Go regressions, vet, module verification,
  date helpers, process exits before/after metadata commit, actual bounded NTFS
  capacity refusal/retry, actual LocalService/Edge viewer and H264 playback,
  HTTP metadata/export/backup and independent CLI restore passed. Desktop 1280px
  and narrow 390px synthetic viewer screenshots were inspected.
- Browser coverage includes three pages, reverse/direct anchors, current-card
  return, focus, editable/video-key exclusions, zoom/pan/Fit, full-screen Escape,
  unsupported media, hidden-current invalidation and delayed responses after
  close. Library/API fixtures add existing sort orders and query contracts.
- Synthetic evidence artifact `13505712501`: ZIP SHA256
  `6747f6e06bd2d76c487f2d13a85932e399754ba7156888c625a13900bc657620`.
- Installed identity evidence artifact `13506012985`: ZIP SHA256
  `a6cd7940959df3ccff39de0560088aed90b74e3e6e6c73d75f983471002e9c17`.
  Both downloaded archive hashes matched GitHub's artifact digests.
- Internal test installer SHA256:
  `7ce56f9d23d2ab641031449129dfd1c8869f4dad43903fe29dbc3aae60906b86`.
  The installer was executed on disposable runners and was not published.
- Genuine released 0.1.43 installer: 104671997 bytes, SHA256
  `96670e90c4abc55cdc07f480267c6f896b784a2a6c11de9f9f5333edb4158415`.
  Released source remains `90d86bd932234d703c4166e4c47585cfbfd047d5`.
  The exact baseline gate from steps 1–2 was reused, not rerun in this run.

Earlier attempts exposed excessive sparse reverse-page allocation, unsupported
full-screen targeting of a dialog, a browser assertion racing re-anchoring, and
the need to refuse database growth on already exhausted Windows storage. Those
were corrected before this passing run. The disk test now also checks export
errors and catalog version when proving an unchanged failed mutation.

No new product dependency, model or provider was introduced. Accepted bundled
provider artifacts were reused for the installed test. Environment-gated native
quality, broader format/endurance, large-container/ZIP64 and large People-index
studies were not rerun. Server 2022/Edge evidence does not certify Windows 10,
reboot, physical power-loss durability, signing or final release acceptance.
There was no installation or migration of the user's library and no new release.
