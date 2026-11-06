# 0.1.43 validation scope

Validation scope for the0.1.43 prototype. Independent native quality and catalog
compatibility evidence are recorded in automatic-naming-quality-0.1.43.md and
release-0.1.43-catalog-compatibility.md. Exact final installer acceptance must be
recorded separately; target Windows10 acceptance remains distinct.

## Required cloud release evidence

- Genuine permanent published 0.1.42 installer, SHA256
  `18dbc670ce4d46809e2a90d5831d17b31f633050907290082f1e793294df8b87`,
  installs and upgrades to the exact 0.1.43 candidate. Injected initial failure
  restores the captured 0.1.42 executable/catalog and preserved originals.
- A separate real-Setup case installs an existing catalog-2 library, preserves
  automatic provenance plus manual correction/undo history through injected
  0.1.44 rollback, then backs up and restores/reinstalls independently with its
  old source unavailable. Catalog version, original hashes and access code persist.
- The synthetic 0.1.44 failed-upgrade fixture restores the captured prior
  0.1.43 executable/catalog, organization, credentials, original bytes and healthy
  service. It is a rollback test fixture, not a 0.1.44 feature release.
- Existing actual LocalService old-writer deletion controls remain exercised on
  an explicitly unpromoted catalog. Schema-2 old-writer refusal is tested
  separately; no catalog is forced back to version 1.
- Date provenance and unread-time-zone wording, whole-gallery search, assigned
  multi-person names, filenames/custom/active object tags, hidden rules, filtered
  paging, clear, empty results, viewer return and desktop/narrow layouts pass.
- Enabled automatic naming passes its independent quality gate and actual
  analysis-publication checks. Uncertainty, manual corrections, undo, multiple
  people, restart, reanalysis, provenance export and independent restore pass.
  Automatic assignments never train subsequent assignments without confirmation.
- Catalog-1 manual-only compatibility, atomic catalog-2 promotion, failed
  transaction rollback, read-only export/verify/backup, restart, independent
  restore, unsupported-version refusal and genuine 0.1.42 old-writer refusal
  pass without source-state mutation on refusal/read-only paths.
- Full storage/HTTP tests, pinned module verification, vet, existing security and
  input regressions, bounded full-disk tests, complete media/AI provider execution,
  package privacy, genuine installed authentication/browser checks and independent
  restore/reinstall/uninstall preservation pass on the final candidate source.

Exact source, run/job identifiers, installer size and checksums must be recorded
after acceptance. An uploaded test artifact alone is not a published release.
The installer build refuses automatic naming that has not been explicitly enabled
after its independent quality acceptance.

## Focused target check after publication

After the tested installer is issued, use the ordinary same-folder update and
confirm the running version is0.1.43.
Check existing photos, names, tags and analysis remain available.

Use ordinary existing media to check **Date taken** versus **Date imported**,
then search in Photos & videos for a known assigned person, a filename and a tag.
Check the visible result count, paging, opening/closing a result and Clear. A
photo of two assigned people should be findable under either name. Hidden items
should remain within Hidden Media.

When a normal new import provides a familiar person with enough manually named
reference originals, check any automatic name and its displayed provenance.
Uncertain faces should remain in review. Correct or undo an assignment only when
there is an ordinary reason to do so. No extra import or forced reanalysis is
requested just to produce a test result. If suitable media is unavailable, record
that portion as untested rather than enlarging the library.

No routine reboot, destructive restore drill, fresh installation or dependency
installation is requested here. Retain a verified backup and prior installer.
Existing target results do not certify this candidate. Windows 10 clean-machine,
reboot, endurance and final-candidate recovery gates remain open. SEC-05 remains
paused/open and SEC-08 remains open; security is six closed/two open, and
milestones 7 and 8 remain incomplete.
