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

Current checkpoint: implementation and tests prepared; actual Windows acceptance
pending. Production automatic naming remains disabled. Separate independent FEI
quality evidence uses its frozen unchanged score policy and research-only ephemeral
inputs; no corpus images or individual predictions enter source/artifacts.
