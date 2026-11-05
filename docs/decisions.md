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

- November 2, 2026 America/Chicago user priority amendment: mitigate Group 1
  Internet-reachable app/component attack surfaces alongside 6A; pause other
  Group 2 mitigation. Retain controls/findings without falsely closing them.
  This supersedes all-backlog-before-6A ordering. Normal immutable originals,
  recovery and honest compatibility/dependency execution gates remain.

- User clarification: Groups 1/2 classify only the original eight; newly found
  issues go in passive Group 3 with no investigation time. Current evidence
  establishes no open Group 1 exploit among the eight; all eight Group 2.
  Continue 6A functional implementation. Separate R-RESTORE-01 reliability from
  proven SEC-04 bounds; no claim to have explained or fixed the refusal.

- Milestone6A candidate freezes initial sampled coverage at up to12 evenly spaced
  source-relative targets, excludes the end, stores actual decoded timestamps and
  skips duplicate timestamps. Each frame commits one checkpoint; pause/cancel
  preserves completed observations and finishes an in-flight frame. Existing
  model identity remains shared with supported still photos; sampling policy
  has its own versioned field, allowing compatible review-only name suggestions
  without automatic naming/tracking. Full-frame coverage remains out of scope.


- November 4, 2026 (America/Chicago): user reported the combined 0.1.39 regression
  passed every exercised check, then instructed proceeding with milestone 7.
  Preserve local NOT TESTED items; see validation-0.1.39.md. Reuse actual 0.1.39
  installer/rollback/payload/restore acceptance instead of repeating green CI.
  Milestone 7 preparation and manual update procedure are in milestone7-hardening.md
  and update-guide.md. Group 2/3 security deferrals are not silently revoked by the
  generic milestone instruction. Before resuming original SEC-02/03/05/06/08
  mitigation or extra security experiments, obtain the explicit scope decision.
  Closed SEC-01/04/07 remain closed. Signing remains unconfigured and prototypes
  unsigned; no certificate/account purchase, new workers or new costs authorized.

- November 4: user explicitly approved resuming original SEC-02/03/05/06/08.
  Group 3 investigations and paid signing remain deferred; no workers or new costs.
  Begin with code-tree trust refusal and pinned new Setup management scripts.
  Focused real Inno/standard-user execution must precede any closure or downstream
  packaging claim. Keep all original security IDs and closure criteria unchanged.

- November 4: milestone 7 installer code-trust implementation passed focused
  production Inno/LocalService/standard-user/custom and Program Files acceptance
  in 41801504714. Preserve exact limits in installer-trust-milestone7.md. Do not
  mark SEC-02 complete before candidate combined prior-release/rollback gates,
  or infer directory race/native isolation/input/final target closure. Continue
  using 0.1.39 as the accepted published prototype; no new release produced.

- November 4: genuine 0.1.39 to unpublished 0.1.41 combined installer/provider/
  initial failure and later 0.1.42 rollback/independent restore acceptance passed
  in 41802642179, source1c9d5e8. Candidate artifact ZIP and installer hashes were
  independently verified against GitHub digest and packaged manifest. Together
  with focused token/Setup controls this closes original SEC-02 in the candidate,
  not SEC-03 races or final SEC-08 target/release acceptance. Status4 closed/4 open.
  0.1.39 stays published; no new installer offered. Next gate is SEC-03.

- November4: SEC-03 component gate passed Windows run41836459908, source0a08a79.
  Open each component relative to a held directory, use real FILE_LIST_DIRECTORY
  access for sharing pins, reject links/unsafe names on opened files before writes.
  Real NTFS copy and adversarial namespace controls passed; initial metadata-only
  handle and unexercised-racer failures remain recorded. No installer integration
  or original-item closure is inferred. Current count stays4 closed/4 open.
  User permits checkpoint handoffs awaiting a response; next work is installer
  ACL/snapshot/rollback integration, without new charges, workers or local tests.

- November4: user permits one worker for SEC-03 integration review and independent
  privacy/alias controls. Integrate the verified winfs primitive through one fixed
  three-file installer helper operation, preserving preflight ACL policy and
  staged exact-hash/no-write/delete-sharing trust. Require completion receipts,
  actual-file private descriptors, physical-root overlap checks and fail-closed
  partial recovery. No whole SEC-03 closure or new product release from this slice.

- November4: source d74c6ee passed first focused Windows installer integration
  run41842717775/job126618212032. Actual fixed-set snapshot/verified restore,
  partial-copy refusal, private actual-handle policies, real DOS alias controls,
  three staged payload mutation denials, injected Setup rollback and existing
  SEC-01 compatibility passed. The authorized worker reviewed and contributed
  independent controls. Retain source/runtime/placeholder limits in the evidence;
  no original SEC-03 closure, public release or Win10 certification. Next checkpoint
  is remaining privileged root/ACL/diagnostic operations. No workers remain active.

- November4: native installer diagnostic integration passed run41846533710,
  source7f4b99b/job126631387368 on Windows Server2022. Keep fixed label/numeric
  schema and existing private access policy, audit actual held descriptors and
  both rotation leaves before mutation, copy/sync previous before current reset.
  Optional failure-note refusal must not suppress recovery; actual hard-linked
  note/Setup failure proved outside-byte preservation and healthy verified rollback.
  Worker review/independent tests complete. No new dependency/release/Win10
  claim or SEC-03 closure; remaining root/ACL/init/Setup gates stay open.

- November4: user requested continued development with scope/drift prevention and
  allowed one worker. Bound this checkpoint to native private-root creation only:
  existing-tree ACL normalization is a separate gate because inheritable
  SetSecurityInfo can propagate to descendants. Do not use a root-only handle as
  proof of safe recursive ACL mutation or introduce protected-child policy drift.
  Missing components receive final protected policy in relative NtCreateFile;
  verify while held, leave existing components untouched, retain safe progress
  after errors without pathname cleanup/fallback. First focused actual Windows
  run41849734950/source db37f6c/job126642511204 passed native, separate ordinary-user
  creation-only, actual Setup/rollback and existing SEC-01 controls. Worker completed
  review/tests. SEC-03 remains open; no new release, cost or Windows10 claim.

- November4: for existing-tree ACL work, prove the documented MAXIMUM_ALLOWED
  SetSecurityInfo no-child-propagation contract by actual execution before any
  installer dependency. Parent chain stays pinned; leaf opens once at final access
  to avoid self-conflicting sharing locks. Independent unlocked/locked-child,
  protected-child, outside-symlink, future-inheritance and writer-refusal controls
  are required. Source8adf2fa is isolated and not called by production helper.
  Run41862213427/job126686818594 remained queued awaiting a Windows runner at the
  last observation; cross-compilation/vet/review do not establish behavior.
  No completed ACL subtask or reduction in security closure count is claimed.

- November4: prior queued ACL proof passed run41862213427/source8adf2fa. Integrate
  existing state/log/rollback permissions through held MAX trees, preserve exact
  supported safe explicit ACEs and protected descendants, verify owner/protection
  and inherited policy, reject reparses/hardlinks/late names before accepting.
  Keep stationary all-root read-only audits before service stop; stop verified
  service for native catalog-sharing compatibility. ACL-phase failure only
  restarts after final policy audits, otherwise stopped/recovery required.
  Run41868231145/source989fb085/job126707927278 passed native and real Setup/
  sharing-recovery/rollback/SEC01 controls. First run's deny-fixture read failure
  was fixed with preauthorized observer plus explicit fresh-read refusal proof;
  no production policy/assertion weakening. Worker review/tests complete.
  User defines passes as completion/pause sessions; latest7-pass forecast is now6
  after this bounded ACL conversion. Remaining init1/Setup2/uninstall1/races1/
  combined closure1; code-tree permission grant remains Setup/init scope. No
  feature/Group3 expansion, new release/cost, whole SEC-03 or Windows10 claim.

- November4: eliminate elevated installer initialization by using already-proven
  LocalService startup before health; no new store/bootstrap implementation.
  Explicit Windows elevated -mode init (or failed elevation query) refuses before
  diagnostics/state writes. Move injected failure after Health so rollback still
  follows real initialization. Six disposable fixtures bootstrap empty roots via
  their existing services; five explicitly mkdir diagnostics before grants.
  Run41873094251/sourceb31f1aa/job126725226588 passed independent token/alias
  refusal controls, fresh LocalService-owned metadata/directories, unchanged
  command log, actual Setup/rollback/original preservation and SEC01 suite.
  First run's test-only missing success exit was corrected without changing
  candidate exit1/refusal assertions. Worker review/tests done. Broad fixture
  scenarios and other offline mode paths remain in combined/integrated review,
  not falsely certified or moved to Group3. Forecast6 to5 remaining passes. No
  new dependency, feature, release, cost or Windows10 claim.

- SEC-03 code grant: replace path icacls with held native code-tree auditing and
  DACL updates; preserve trusted readers/owners/protection and private protected
  rollback subtrees. Real read-data sharing authority excludes data writers/rename
  while permitting own running image. Preserve exact ALLOW grant multiset despite
  Windows inherited-entry ordering; never admit deny/unsupported mutation policy.
  Run41882276092/source802e097/job126757536563 passed native controls, actual
  custom/fresh Setup, recovery and SEC01. Outgoing same-object hard-link names are
  detected by link-count checks, not falsely claimed impossible or atomic. Full
  interleavings/offline modes remain integrated scope. Payload publication remains
  next; forecast5 to4 remaining passes. No release, dependency or Win10 claim.

- SEC03 Setup payload checkpoint: use bundled temporary extraction and native
  fixed-root/media/ai publication during Install, not Prepare, with all-target
  preflight, disk-backed complete-code snapshots, verified copies and receipt
  after data closes. Checked copy failure restores held prior bytes; completed
  Install-failure recovery restores payload before old3file/state and Health.
  Unverifiable close/receipt/recovery leaves service stopped. Explicit fixed seven
  uninstall names preserve native-created file removal; Inno metadata/hostile
  cleanup remain next. Run41894239272/source30fa427/job126799357326 passed native
  distinct-old/new fullcode recovery/alias/late-name/writer controls and actual
  minimal custom/fresh Setup, rollback, removal and SEC01. Worker review found
  three pre-CI defects, receipt/close ordering and an avoidable Prepare window;
  corrected and tested without weakening assertions. Temp fullcode snapshots are
  not durable powerloss recovery; original3file remains, integrated interruptions
  and real-provider combined closure stay SEC03. Forecast4 to3; no release/cost,
  dependency/feature/Group3 expansion or Windows10 certification.

- SEC03 uninstall: compiler-hash-verified staging of native helper/two scripts,
  actual nofollow app directory-chain pins until Deinitialize, initialize-only
  audit and post-confirmation cleanup. Remove path UninstallRun/Delete and
  overwrite same-app logs to prevent legacy path-action retention. Native all-
  target preflight then read/delete-only bottomup cleanup, preserve state and
  unknown files, retain registration until deletion succeeds, truthful partial
  failure. Actual active Inno journal sharing needed a root-only exact metadata
  integrity audit after sharing refusal; never broaden payload sharing/policy.
  Run41898776310/sourcec6d384e/job126815286190 passed native controls, actual
  hostile refusals (samePID/journal/registration/original/outside), custom/fresh
  removals, privacy/rollback and SEC01. Initial refusal/trace retained. Trusted
  loaded-uninstaller integrity, actual metadata-leaf races/offline/interruption
  limits and genuine prior-release/fullprovider recovery stay explicit original
  SEC03 work. Forecast3 to2; no release, cost, Group3 or Win10 certification.

- SEC03 bounded entrypoints/stationary metadata: refuse elevated serve and
  direct/non-LocalService service before diagnostics, unknown modes before IO;
  read existing code through held nofollow single-link file without Store or
  generation/logs. Native Prepare audits hardlink counts before metadata writes.
  Run41905712864/sourcef8f1c0a/job126839683829 passed exact real no-write/read
  controls, valid journal alias refusal before PID change, explicit post-health
  rollback, real installs/removals/privacy/SEC01. Test-only status capture fixed
  without product or assertion weakening. Preserve admin recovery workflow:
  verify/backup/restore and pre-mainworker require proper held read/write designs,
  not blanket elevated refusal. Persistent prewrite fullcode intent/mixed recovery
  and concurrent metadata tests remain implementation, not merely validation.
  Correct prior2-pass estimate to best5 (range5–7) original SEC03 passes; no
  feature/Group3 expansion, new release/dependency/cost or Win10 certification.

- SEC03 administrative verify/backup: preserve admin and RX-user recovery use
  with existing-only ReadOnly store, native held source leaves/catalog fd hook,
  direct Tx.WriteTo ZIP instead of source staging and no diagnostics/repair.
  Preserve source/count/size/pending-delete/config bounds; compare actual blob
  size, serialize metadata reads. New archive is FILE_CREATE private at birth,
  held-parent/physical source exclusion, handle-only failed-output cleanup and
  truthful incomplete-close reporting. Run41913488167/source348dd18/job126866916248
  passed native tests, actual admin installed source and ordinary RX copy commands,
  immutable source/ACL/namespace/log checks, ZIPpermissions/checksums, independent
  restore/verify and Setup/uninstall/rollback/SEC01. Two workers complete. Initial
  product checks passed but fixture ACLreapply cleanup failed; isolate RXclone
  instead of mutating installed ACLs, retain+expand exact assertions. Same row
  completes; forecast5 to4, unchanged remaining restore/durable/races/combined
  rows. No new dependency/provider/schema/release/cost, Group3 or Win10 claim.


- SEC03 restore: preserve administrative recovery through held nofollow archive,
  exclusive private stage/derived paths and inherited pipe/handle worker protocol;
  refuse legacy self-authorized path requests before file access. Keep strict
  parent pin until protected DELETE-held stage guarantees nonempty parent, then
  identity-check limited final-parent sharing needed for native no-replace rename.
  Actual FSCTL empty-positive/nonempty-refusal proves reparse protection; archive
  ancestor pins may close after strict selected file is held. Use private fixed
  progress FILE_CREATE + sync/close and strict read, never live partial acceptance.
  Retain catalog/ZIP/original checks and existing hard resource/reaping controls;
  held inventory cleanup refuses unsafe late aliases with truthful private residue.
  Run41922662936/source39d7a2e/job126898869323 passes native/boundary/resource tests,
  actual admin recovery, Setup/rollback/uninstall and permission compatibility.
  Correct native sharing conflicts and test-only fast-process/junction snapshot
  defects without reducing assertions; two workers reviewed/tested. Same restore
  row closes, forecast4 to3, remaining durable/races/combined rows unchanged.
  No new release/dependency/provider/cost/Group3 or Windows10 certification.


- SEC03 interrupted publication component: write a private exclusive version2
  prewrite intent after all prior-file backups are synced/verified, before the
  first old-byte truncation. Bind recovery to the application and each exact
  destination object; partial bytes cannot match a completed-payload digest.
  Keep version1 successful-publication receipt and its strict new-byte check
  separate. Explicit interrupted recovery validates all saved old copies and
  destination identities/policies before restoring; repeat recovery tolerates
  already removed new leaves, never missing old leaves or replacements.
  Source5e17f2c/run41944521442/job126973302428 passes actual process-exit and
  partial-content recovery, repeat recovery, corruption/replacement refusal and
  existing native installerfiles/build/vet checks on Windows Server2022.
  No Setup orchestration change yet: temporary backups are still not durable
  transaction storage. Persistent roots, pending-transaction/commit lifecycle,
  service-start gating and actual killed-Setup/retry acceptance remain in the
  SAME first outstanding row. Do not reduce the three-workstream forecast or
  claim power-loss/Windows10 acceptance. Separate candidate branch only; no
  security-baseline update, release, dependency/provider/cost/Group3 expansion.


- SEC03 interrupted-upgrade integration accepted: persistent private sibling
  transaction/payload roots; exclusive synced pending metadata after the prior
  snapshot, resolve pending recovery before the next snapshot, and identity-bound
  prewrite full-payload recovery. Verify/hold all state snapshot sources and live
  state leaves before code rollback. Disable service during mixed copies, start
  complete candidate manually for health, commit before automatic startup. Write
  synced terminal metadata before held fixed-leaf retirement; delete terminal
  last so interrupted cleanup cannot roll back a committed candidate. Unknown or
  unsafe objects retain recovery data and refuse, never guessed success.
  Source649c145/run41947440346/job126983031492 passes native controls plus actual
  production Setup process-tree kill during synthetic1GiB publication, recovery
  before resnapshot/retry, exact old catalog/executable and access-code/original
  preservation, private intent denied to a real standard user, commit/retirement,
  auto-start return, normal/fresh/explicit rollback/uninstall/SEC01 controls.
  Final canonical Go formatting from the tested runner is saved without behavior
  changes. Initial integrated8604f4/run41946697550 also passed; subsequent state
  preflight refinement and its corruption/empty-root/fresh tests passed final gate.
  Same outstanding row closes, forecast3 to2; remaining concurrency and genuine
  release/fullprovider/recovery rows unchanged. No release/provider/dependency/
  catalog-schema/cost/Group3 work or Windows10/physical-powerloss certification.
- Existing-function UI acceptance uses fresh actual LocalService fixtures for
  hidden-navigation and People tests, preserving the elevated/direct-service
  entrypoint guard. Dedicated Server/Edge gate source d1a47efe/run41958781493/
  job127019911895 passed full application/provider/restore regression plus five
  desktop/narrow shells. Keep the default-off UIReview switch and attach-mode
  fixture support for final integration. No new backend/auth/CSP policy/provider;
  Curation/Cinema/durable review decisions/counts/global filters remain deferred.
  Final visible-text cleanup preserves identical markup/scripts/styles. See
  ui-redesign-acceptance.md; no release, SEC03 closure or Windows10 certification.
