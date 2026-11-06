# 0.1.43 prototype release acceptance

PASS: combined Windows Server 2022 run [42170003998](https://github.com/aataat12/immich-clone/actions/runs/42170003998), job 127735349942.
Exact tested and tagged source: `90d86bd932234d703c4166e4c47585cfbfd047d5`.
Published [v0.1.43-prototype](https://github.com/aataat12/immich-clone/releases/tag/v0.1.43-prototype)
from the accepted artifact without rebuilding. Publication run
[42171398610](https://github.com/aataat12/immich-clone/actions/runs/42171398610), job 127739815798, passed.

## Deliverable identity

Installer: `ImmichClone-0.1.43-prototype-x64-setup.exe`, 104671997 bytes.
SHA256: `96670e90c4abc55cdc07f480267c6f896b784a2a6c11de9f9f5333edb4158415`.
Accepted artifact: 13481410728, `prototype-0.1.43-server-tested`.
Artifact ZIP SHA256: `d9fb2946b42490ce32786938c5b397050ffed1bc19bfe006b86726409a8e6975`.

The downloaded ZIP contained exactly the installer, checksum list, release notes
and validation scope. Every extracted file was hashed; the checksum list matched
the installer and both notes matched the frozen source after line-ending normalization.
Publication rechecked the successful source/run/job identities, artifact identity,
ZIP and all four deliverables. The draft assets were independently checked before
publication. The version tag was created and verified at the accepted source before
the public transition. No existing tag, release or asset was overwritten.
All four public release downloads were then independently downloaded and matched
the accepted hashes. The verified installer was saved in the designated local
release output folder; the preceding 0.1.42 installer remains intact.

## Accepted scope and evidence

The three agreed changes are clearer date provenance, search within Photos & videos,
and automatic identification of strongly matched known people. Broader research
features remain outside 0.1.43. No dependency, native provider or model was added.

- Enabled feature/browser, actual native assignment/recovery and genuine old-writer
  compatibility passed [42166864007](https://github.com/aataat12/immich-clone/actions/runs/42166864007)
  at source `204c5a5abc0237b54f71f3f9242dbdbcf70d4c77`.
  Final combined acceptance checks ancestry and the canonical Git blobs of all
  fourteen production feature files; later changes are canonical formatting,
  packaging, evidence and corrected acceptance fixtures.
- Frozen-policy independent Nottingham evaluation passed
  [42164599554](https://github.com/aataat12/immich-clone/actions/runs/42164599554)
  at source `d6cfd0968d5c5b8d9671520fcabeb71dd1f7254a`: 129/181 known queries correctly
  assigned, no wrong identity, 52 unresolved; 239 detected unknown queries rejected,
  no false assignment. Six scheduled unknown queries were unavailable/unresolved.
  This controlled research corpus does not establish general-library recognition
  accuracy. See [the quality decision](automatic-naming-quality-0.1.43.md).
- Focused actual-Setup trust gate passed
  [42169084769](https://github.com/aataat12/immich-clone/actions/runs/42169084769)
  at source `160aa35e116f36d8fe98ca274db61be7496a6153`.
- Final combined acceptance passed all Go tests/vet/module and vulnerability checks,
  bounded NTFS full-disk recovery, full browser/service/native-provider/media/input
  regressions, package checks, actual published 0.1.42 upgrade, injected failure
  recovery, uninstall preservation and independent restored reinstall.
- The additional installed catalog-2 case passed actual Setup enrollment, automatic
  provenance plus manual correction/undo history, injected 0.1.44 fixture rollback,
  exact prior executable restoration, access-code and original-file preservation,
  HTTP backup, uninstall and independent restore/reinstall with the old source
  unavailable. The 0.1.44 package exists only as a failed-upgrade fixture.
- Genuine 0.1.42 reader/writer refusal on catalog 2 preserves source bytes.
  Manual-only catalog-1 behavior and existing prior-writer controls also pass.
  No database is forced back to version 1. See [catalog compatibility](release-0.1.43-catalog-compatibility.md).

Earlier combined runs stopped on acceptance fixtures that still expected the old
notes location or gallery search to navigate to People. Those fixtures were corrected;
no failed-run installer was released. The first publisher created and verified the
unpublished draft, then stopped because drafts have no version tag yet. The successful
publisher revalidated that same draft and assets without replacing/uploading them,
explicitly created and checked the accepted-source tag, then published it.

## Remaining limits

Cloud results use disposable Windows Server 2022 runners; they do not certify
Windows 10. No live installation, media changes, reboot or private-data upload was
performed. Target checks are described in [validation scope](validation-0.1.43.md).
The package remains an unsigned prototype. Original security status remains six
closed and two open: SEC-05 paused/open and SEC-08 open. Milestones 7 and 8 remain
incomplete. Final Windows 10 clean-machine, reboot, endurance and final-candidate
recovery acceptance remain open. Older releases cannot open a catalog promoted to
version 2; retain the verified pre-upgrade backup and prior installer.
