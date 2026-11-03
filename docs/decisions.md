# Decisions

- 2026-11-01: Project source requirements copied verbatim to repository root and
  authoritative. The prior diagnostic's proposed immediate feature build is
  superseded by compatibility/dependency acceptance gates.
- No production .NET/PostgreSQL version selected by the old diagnostic.
  LocalSystem and trust authentication remain disposable diagnostic shortcuts.
- [Historical; superseded by the later Server-first user amendment] Windows 10 remains the target. Do not substitute Server/11. Continue only
  portable experiments and preparatory work while access is unresolved.
- No Azure provisioning or license purchase without established eligibility,
  bounded cost and user authorization. Prefer an existing eligible entitlement.
- AI remains optional; no model selected and no inference success claimed.
- PC report stays in conversation, never public commits or CI logs. Inventory
  script prints selected technical fields only and makes no configuration changes.
- 2026-11-01: Treat cloud-synced media as a
  read-only external source initially: no writes/deletes, no deletion propagation,
  no database/jobs/cache inside the sync root. Missing/offline sources are an
  availability failure, not authorization to delete catalog entries. Detect
  cloud placeholders before opening; do not silently hydrate an entire library.
  Reject reparse-point traversal and recheck identity/hash around reads before
  accepting derived results. These are acceptance requirements, not implemented claims.
- Keep application runtime undecided: current .NET 10 supported-OS documentation
  omits Windows 10 Pro 22H2. ESU for Windows alone is not assumed to establish .NET
  vendor support. Resolve support policy before a substantial .NET implementation.
- FFmpeg LGPL shared build is an isolated candidate executable, invoked out of
  process. Pin monthly retained build and checksum. This is not redistribution
  approval; corresponding source and transitive license review remain release gates.

- 2026-11-01 user override: cloud Server 2022 validation now authorizes development;
  final Windows 10 validation moves to the user's PC after a packaged working
  build. No paid cloud Windows 10 environment needed; earlier cloud-access gates
  are superseded. Do not request routine milestone approvals.
- Evaluate Go 1.27.3 with CGO disabled and GOAMD64=v1, bbolt 1.4.3, and x/sys
  0.49.0 through real LocalService execution before building downstream features.
  Embedded catalog avoids PostgreSQL installation and preserves a standard file
  layout plus JSON export migration path. DB is single-writer local state, never
  inside Dropbox; originals remain independent. Candidate until execution passes.

- Basic runtime/embedded database/LocalService probe passed. Proceed with the
  originals slice using these components. No production reliability claim.
- Prototype metadata schema 1 lives in bbolt; originals are independent immutable
  SHA256-named files. A blob is published exclusively before catalog/job commit.
  Interrupted ingestion can leave an unreferenced blob, never a successful record
  referencing an uncommitted upload. Such blobs are retained; no automatic deletion.
- Native Go JPEG/PNG preview path only for this slice, 40MP preview limit and 4GiB
  streaming upload limit. Other files remain downloadable with explicit unsupported
  preview errors. EXIF orientation and camera metadata are pending. No FFmpeg
  redistribution is required for this initial build.
- Authentication is local single-user access-code login with ephemeral sessions,
  loopback binding/Host restriction, same-origin writes and CSRF request header.
  LAN/multi-user/public exposure remain disabled pending security design.

- Inno Setup 6.7.4 is pinned for prototype packaging (official source license
  permits use without a purchase). No external redistributable runtime for the
  CGO-disabled Go build; license notices are packaged from exact dependencies.
- Installer data is outside Program Files and outside sync folders, protected by
  ACLs for system, administrators, LocalService and the installing account.
  Nonempty unrelated data folders are refused. Upgrades preserve the data path.
- Schema-1 upgrades stop the service and retain protected copies of the previous
  executable, catalog and access code. On startup failure restore those copies;
  no original is modified or deleted. Future migrations require separate rollback
  tests. Uninstall removes service/program files and retains all application data.
- Prototype artifacts are unsigned and expire after seven days. They are test
  builds, not approved for bulk ingestion or a claim of Windows 10 compatibility.

- Critical checkpoint after the Server-tested native prototype: require actual
  local Windows 10 installation/helper/reboot results before claiming desktop
  compatibility or clearing real-library release acceptance. No paid cloud OS
  provisioning is needed. The application is still an early prototype; broader
  formats, AI and endurance remain unaccepted.

- 2026-11-01: User requested extensive support logging and a new release. Use
  structured allowlisted events and classified errors/native codes, not raw
  transcripts or exception messages. Logs have a separate ACL-protected directory
  so an application-data permission failure can still be diagnosed. No inventory
  collection or automatic upload. Browser and offline Start menu exports combine
  bounded files; originals/catalog/backup semantics remain unchanged.

- 2026-11-01 user-reported local functional acceptance passed, including PNG
  preview/full-image view, service restart, original hashes and independent restore.
  Accept this limited Windows 10 evidence without publishing the transcript or
  machine information. User deferred reboot validation to final acceptance; it
  does not block further development. Current setup requires no OS reboot.

- 2026-11-01 metadata slice: retain schema 1 with additive photo metadata,
  relative source aliases and a preview revision. Requeue older JPEG/PNG derived
  previews once; never rewrite original blobs. Failed upgrade rollback still
  restores the pre-upgrade catalog. JSON export/backup preserve new fields.
- Bounded standard-library JPEG APP1/TIFF parsing supports orientation 1–8,
  capture date (original date preferred), camera make/model only. No native
  runtime or new third-party dependency. Camera wall-clock times have no invented
  timezone. Unsupported/malformed metadata warns without sacrificing originals.
- Relative folder paths are catalog metadata, never executable filesystem paths.
  Reject absolute, drive-qualified, traversal and control-character input before
  writing. One blob may retain multiple relative aliases; diagnostic logs omit all.

- BMP/TIFF/GIF prototype gate passed before integration. Pin x/image 0.47.0
  for pure-Go BMP/TIFF decode; GIF uses standard library. Keep first frame/page
  only with 64 MiB encoded/40MP guards. TIFF viewer shows the derived preview,
  not a falsely promised full-resolution image. Explicitly defer TIFF metadata,
  HDR/color management, HEIC and RAW. Old unsupported assets reconsider once.
  Version 0.1.6 was a deliberate failed-upgrade fixture; deliver 0.1.7 next.

- User reports 0.1.7 BMP/GIF/TIFF acceptance passed and earlier JPEGs remain good.
  Keep this as user-reported evidence without private media/logs or inventory.
- Next provider gate: time-box a minimal FFmpeg 8.1.3 build from pinned official
  source instead of integrating the broad BtbN binary before its source/native
  audit is complete. Decoder-only H264/HEVC plus HEIC container experiment; no
  network, external codec libraries or GPU. Keep exact corresponding source and
  compiler/configuration evidence together. No app dependency or new installer
  until actual service operations and remaining distribution/containment gates pass.

- Native worker gate passed: Windows Job List assigns a 512 MiB commitment/one-
  process/kill-on-close job before execution. Real control allocation/spawn tests,
  crash, timeout, output flood and parent-death cleanup passed under LocalService.
  This bounds resources; it does not claim a filesystem/token security sandbox.
- HEIC grid reconstruction uses FFmpeg's grouped output and built-in xstack/crop,
  verified with 6-tile edge cropping and a 12MP/48-tile four-color fixture. Keep
  HDR/high bit depth, RAW, bursts and broad camera metadata unsupported.
- Integrate an optional checksum-pinned subprocess provider. Move the tested job
  launcher into internal/media; cloud probe imports this exact implementation.
  One provider failure is a visible retryable preview failure, never a storage or
  startup failure. Service cancellation preserves the durable queued job.
- Package corresponding source/license/compiler/MinGW notices with the provider.
  Immutable binary-hash provider directories survive executable rollback without
  replacing an older provider's files. Deliver 0.1.9 next (0.1.8 was failure fixture).

- User requested an in-page running version and removing cards from the carousel.
  Show the executable's actual build in the top-right (including narrow layout).
  Implement removal as reversible hide/unhide metadata in schema 1; originals,
  jobs and backup/export retain hidden records. Permanent deletion was not
  requested and is not implied. Preserve hidden state on duplicate uploads.
- Render a complete detached grid and ignore superseded refresh responses. This
  prevents an older snapshot from replacing a newer grid; it does not establish
  the cause of the separately reported local MOV disappearance.

- User reports MOVs now show; close the diagnostic-input gate unless it recurs.
  Treat this as user-reported visibility, not established root-cause/codec proof.
- Measure catalog/worker operations with bounded metadata-only synthetic rows,
  plus an actual PNG job. Do not present row-count benchmarks as bulk import or
  camera-corpus acceptance. Use observed bottlenecks to scope queue/paging changes.

- Baseline Linux and Windows measurements show full-catalog idle polling grows
  linearly (about 88 MB Go allocations per poll at 50k synthetic rows). Add a
  derived preview_queue_v1 bucket; index and asset status commit in one bbolt TX.
  Rebuild the derived index from authoritative asset states on every open, so old
  catalogs, restores and older-writer rollback remain recoverable without changing
  schema 1 or original layout. No new dependency or parallel worker is introduced.
- Limit this milestone to worker lookup; API listing/JSON and browser rendering
  remain unpaged and are separate measured bottlenecks. Do not claim a 50k-media
  import from catalog-only fixtures. VM timer-zero readings are below measurement
  resolution, not proof of zero execution time.

- [Historical; superseded by the managed-copy decision below] User clarified that the primary workflow is indexing an existing mostly-static
  library without duplicating originals. Existing Upload/Choose folder copies
  bytes into managed storage, so this is a material distinction in user behavior.
  Prioritize read-only external-library feasibility, preserve existing managed
  copies, keep source inventory private, and make copy-import explicit. Avoid
  automatic permission changes/hydration or claims that a catalog backs up source
  originals. Queue indexing remains useful but is not released before acceptance.

- User explicitly retained managed-copy import after considering duplicate storage.
  External-primary planning is superseded; no external implementation to migrate.
  Preserve source files and immutable managed blobs. Build bulk import separately
  from the gallery, with durable per-file acceptance, byte/file progress and
  import-first scheduling, then drain persistent previews. Do not imply browser
  file selection survives closing the page. No personal library inventory in CI.

- User moved people/object identification and organization ahead of further import
  development/testing because AI success determines project usefulness. Preserve
  verified managed-copy work and defer bulk-import milestones. First prove real
  candidate model inference, quality/limitations and Windows service/packaging,
  then build reviewable people groups, manual naming/merge/split and object
  tags/search. AI remains replaceable and unable to damage originals. No user photos
  sent to cloud inference; private media and personal machine information never
  enter public CI. Public licensed fixtures may be used in cloud CPU experiments.

- Review controls confirmed: unnamed/uncertain queue, name confirmed groups,
  manual missed-face assignment, split/merge/move, batch custom tags and dismissal.
  Keep original bytes immutable and corrections independent of model versions.
- The initial cross-photo screen supports candidate named-person assignment but
  also finds different-person pairs above the basic threshold. Do not automatically
  merge people using single-link/pair similarity or propagate names from unreviewed
  suggestions. Continue conservative reviewed grouping and broader quality gates.

- User authorized a Hidden Media UI with leave-hidden, restore and explicit
  permanent-delete controls. This is an intentional exception to retaining a
  managed original after the user confirms deletion, never permission to delete
  external sources. Show actual local managed locations and recorded relative
  aliases; do not invent unavailable absolute source paths or log them. Durable
  deletion/recovery and synthetic preservation tests gate delivery. See hidden-media.md.

- Native AI candidate: source-built static OpenCV 4.12.0 preserves model parity
  while testing a self-contained Windows worker, SSE2 baseline, no GPU/Python or
  compressed-media parsers. Bounded BGR working frames separate inference from
  original decoding. Per-task model loading reduces combined memory pressure.
  Actual service operations/PE audit, exact third-party notices, current security
  review and target packaging gate integration; no shipping dependency selected.

- Native worker uses /MT static runtime linkage for the experiment. Runtime/CRT
  fixes require a newly built/tested worker; Windows redistributable servicing does
  not update embedded static code. Keep provider replacement independent of
  originals/catalog. Retain build-only checkpoints separately from service
  acceptance to avoid full recompiles for harness fixes. Parser backport source
  and SHA256 remain explicit, not a silent modification of the upstream archive.

- First People/tagging pilot: explicit per-photo CPU analysis of native still
  image formats, 64MiB/40MP input guard, original-derived oriented 960px BGR frame,
  fixed worker/model checksums and existing bounded Windows launcher. HEIC/video
  AI remains deferred. AI jobs stay durable across cancellation and use a separate
  processing mutex so native inference does not hold catalog/backup writes.
- Model results, face geometry and embeddings live separately from manual names,
  one-level name undo, custom tags and dismissed AI tags. No automatic naming or
  merging; nearest named-reference suggestions require explicit confirmation.
  Ready analysis is not silently rerun. Manual face marking uses the canonical
  preview's orientation; normalized boxes survive independent crop regeneration.
- Add People & tags navigation and bounded pilot review pages. Names/tags/face
  crops remain usable without the provider; full private catalog backups retain
  organization. JSON export excludes raw embeddings and uses one snapshot.
  This is a small reviewed pilot, not broad family recognition or bulk acceptance.

- User resumed postponed import work while privately validating 0.1.15 AI. Keep
  those gates independent. Add a dedicated managed-copy page and additive import
  receipt buckets using the accepted blob/preview transaction, without changing
  original storage or introducing dependencies. Copy progress is durable only
  after catalog+receipt commit; network byte progress is explicitly provisional.
- Copy-first scheduling uses one renewable in-memory 60-second lease, cleared by
  process restart and released by expiry/pause/cancel/completion. Start no new
  previews/AI during the lease; do not forcibly cancel already running inference.
  Durable session history survives while runtime pauses cannot strand work.
- Initial resume reselects the full manifest and retransfers accepted bytes for
  SHA256 receipt comparison. Avoid metadata-only skips, false receipt counts and
  claims of partial-file resume. This conservative checkpoint precedes optimization
  and full-library scale acceptance. Test upgrade from the actual accepted 0.1.15
  installer, not a relabeled build of the new source.

- Gallery uses additive derived bbolt ordering/filter indexes, rebuilt from assets
  at open for older-writer/restore recovery. Update indexes/counts/order revision
  in the asset transaction, including import receipts. Reads seek directly to
  bounded pages; no full List+sort hidden behind the paginated response.
- Opaque validated cursors bind sort/filter, runtime index epoch and order revision.
  Ordering changes/restart invalidate old cursors; UI resets to page one with a
  notice. Preview-only status changes retain cursors. Later pages refresh on user
  action rather than automatic polling. Date ties use content ID, filename order
  is UTF-8 case folded (first 1024 bytes for nonstandard overlong API names), then
  content ID. No timezone is invented for camera capture dates.

- Startup rebuild sorts each derived index's keys before bulk insertion. A local
  50k metadata experiment exposed random-insertion startup cost; sorted insertion
  reduced the observed rebuild while retaining the same authoritative source and
  bounded runtime page reads. One-time rebuild memory/work is separate from page
  allocation measurements. No claim of a 50k media import or full-library safety.

- Backup checkpoint retains the existing schema-1 restorable ZIP. Managed web
  jobs preflight application-storage free bytes against a conservative archive
  bound plus catalog snapshot and 64MiB reserve under the same consistency lock.
  Actual failures still clean partials; no free-space promise despite other apps.
- Persist one service-owned backup job outside catalog snapshots. Status/plan
  values stay local; logs omit paths, free-space inventories, IDs and checksums.
  Context cancellation covers lock wait, snapshot/archive/read-back/hash steps.
  Catalog writes wait during archive creation; browsing and job controls do not.
- Read back all archive checksums and compute archive SHA256 before publishing a
  download. Keep one verified temporary archive until explicit discard, supporting
  repeated/range download without regeneration. Never auto-delete completed user
  downloads or managed originals. Restart exposes an interrupted job and removes
  generated incomplete files; unknown/corrupt job state fails closed for backups.
  No external destination, incremental/deduplicated backup or automatic retention.

- Include an offline Build Notes page in the bulk local AI milestone. Place its
  entry beside People & tags and Home at the page top right. Track all eight
  remaining milestones with status and actual delivered release; keep shipped
  content aligned with docs/roadmap.md without forecasting release numbers.

- Bulk analysis owns only its atomically selected not_requested visible native
  still-image jobs, using additive batch buckets and the existing inference queue.
  Persistent pause affects batch members only; cancel returns pending members to
  not_requested, allowing active inference to finish. Skip failed/ready/queued work
  and preserve manual corrections; no automatic retry or naming. Missing provider
  leaves batch work waiting. Build Notes ships offline with the actual release.


- 2026-11-02 user authorized two bounded security workers while the primary agent
  resumes People functionality coding. Installer/ACL and native-isolation reviews
  have separate owned artifacts; no unrestricted delegation or release bypass.
- False-face dismissal is an additive face-label correction, not detection/original
  deletion. Exclude dismissed faces from active review and name suggestions, retain
  names/undo history, and provide a separate restore view. Existing full-catalog
  backups/JSON export retain the correction. No provider rerun or new dependency.

- 2026-11-02 continued user-authorized parallel work: separate security workers
  own staged-upgrade preparation and a native-child access canary experiment.
  Primary People work applies unnamed/search filters before pagination; reference
  candidates remain independent of the displayed face filter. No release bypass.

- 2026-11-02 user requires complete original security items, not partial probes.
  Keep the original eight stable IDs in security-worklist.md. Two authorized
  workers jointly own SEC-01 implementation and independent Windows negative/
  normal controls; only reviewed passing completion evidence closes the item.

- 2026-11-02 user authorized four workers for two complete original items:
  SEC-04 and SEC-07, each with implementation plus independent verification.
  No partial probe or reassigned unfinished review counts as closure. Main agent
  handles People previous-page navigation and integration. No release bypass.

- SEC-04/07 closure follows full combined Windows execution 41547447887 and
  independent review, not source readiness. Original tracker is 3 closed / 5 open.
  NR-01 from the completed review blocks release until the native provider is
  patched/replaced and actual parity/hostile/integration checks pass; no risk
  acceptance or silent codec/model removal. People Previous navigation remains
  held for the combined release, with no installer/version/main change.

- 2026-11-03: prioritize NR-01 with hash-pinned upstream backports to the exact
  retained 4.12.0 source; six reviewed production fixes apply cleanly locally.
  Require actual API/model/LocalService execution before provider selection.
  Add HEIC/video AI as milestone 6A without renumbering original milestones.
  Initial video scope is explicitly bounded sampled frames, not all-frame or
  automatic identity merging; feasibility must establish policy before shipping.

- NR-01 native and actual application acceptance passed with new immutable
  provider 15d1c2ca7ef3. Preserve prior ready provenance and user corrections;
  no automatic rerun or installer/version/main publication.
- Reopen original SEC-04 for R-RESTORE-01: one valid restore refused in a later
  run, cause unconfirmed despite 50 Windows repetitions/full app restores then
  passing. Keep fixed IDs and explicit evidence; no policy/test weakening or
  unsupported claim of a root-cause fix. No user PC action needed.
