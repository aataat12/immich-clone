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
