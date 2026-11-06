# Automatic name provenance and older app versions

Catalog version2 is required when automatic assignment provenance exists. The
first successful automatic label and the version change commit in one bbolt
transaction. Aborted publication leaves both unchanged. Empty/manual-only
libraries remain version1; correction, undo or later deletion never downgrade2.

New readers/writers accept versions1/2; the genuine0.1.42 writer and verifier
accept only1 and must refuse2 before committing changes. The directory marker,
backup ZIP manifest and JSON export container retain format1 because their layouts
are unchanged. JSON export adds catalog_schema for the actual catalog version.
Backup retains the exact private catalog including assignment provenance.

New writable open also promotes intact current/previous automatic provenance in
development-era version1 catalogs. Read-only commands refuse such inconsistent
state without repair or mutation. No previously published version assigned names
automatically. Unknown catalog versions remain unsupported and are never reset.

Installer failed-upgrade rollback restores its saved prior catalog along with
the old executable. After successful automatic naming, an older executable
cannot operate on the newer catalog; an intentional downgrade requires a
compatible pre-upgrade backup. New imports/originals are never deleted to make a
catalog compatible. Combined upgrade/rollback acceptance remains a release gate.

Tests cover atomicity, no-assignment compatibility, preservation across restart,
read-only export/verify/backup, independent restore, undo/correction and unknown
version refusal. The disposable Windows fixture downloads the checksum-pinned
published0.1.42 installer; positively opens the same synthetic library as version1,
promotes it, verifies old reader/writer refusal with exact source byte/namespace
preservation, then checks the candidate service and verifier on that library.

Accepted Windows checkpoint: source `bad3a94b3b3a3d323c558cc62750d6e3d7942b38`,
[run42163237480](https://github.com/aataat12/immich-clone/actions/runs/42163237480).
Job127713610818 passed genuine0.1.42 schema-one positive control, old read-only and
LocalService writer schema-two refusal, exact namespace/all-file hash preservation,
new service automatic-provenance retention and new read-only verification.
Job127713610583 passed full Go tests/vet, date helpers and LocalService/Edge search
regressions. Atomic migration, restart, offline export/verify/backup and independent
restore tests passed as part of that suite. Runner canonical Go formatting is
copied back without behavior changes.

Automatic naming is enabled for the next validation pass after the independently
measured [quality decision](automatic-naming-quality-0.1.43.md). Enabled publication
and combined release gates are still pending. FEI acquisition failed before
inference from the cloud runner: direct archive request403 in run42163237506 and
normal official public-page request403 in run42163644644. The normal public
page/session/download flow succeeded locally with unchanged archive checksum,
but cloud access did not. No recognition result exists for FEI, no access bypass
was attempted, and the policy/split/gates remain frozen. Further FEI retries are
stopped. The independent Nottingham source was subsequently accessible and passed its
frozen gates. No portraits, embeddings or individual predictions enter source/artifacts.
