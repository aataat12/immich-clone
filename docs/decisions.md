# Decisions

- 2026-11-01: Project source requirements copied verbatim to repository root and
  authoritative. The prior diagnostic's proposed immediate feature build is
  superseded by compatibility/dependency acceptance gates.
- No production .NET/PostgreSQL version selected by the old diagnostic.
  LocalSystem and trust authentication remain disposable diagnostic shortcuts.
- Windows 10 remains the target. Do not substitute Server/11. Continue only
  portable experiments and preparatory work while access is unresolved.
- No Azure provisioning or license purchase without established eligibility,
  bounded cost and user authorization. Prefer an existing eligible entitlement.
- AI remains optional; no model selected and no inference success claimed.
- PC report stays in conversation, never public commits or CI logs. Inventory
  script prints selected technical fields only and makes no configuration changes.
- 2026-11-01: Target inventory received privately. Treat cloud-synced media as a
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
