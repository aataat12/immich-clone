# 0.1.42 prototype release acceptance

PASS: corrected combined Windows Server 2022 acceptance run42147333212/job127661889239.
Tested source: f37f61f2fe79deb92eeb3ff614ea6b245180a391.
Installer: 104649228 bytes, SHA256 18dbc670ce4d46809e2a90d5831d17b31f633050907290082f1e793294df8b87.
Artifact13471895175, ZIP SHA256 64338a54ac2654b0f5052f9f7070a65c2b0e141e8fa7680142bcf5b657748330.
All four extracted files were independently checked; checksum list matches installer and notes match frozen source after line-ending normalization.
Publication pending the separate immutable-artifact publisher.

## Workstream reconciliation

All five requested owners were contacted and their current reports read.
SEC-03 concurrency is checked in at dcc54c1f8ba2846626c1e58c7d9671d9dba9561e
and is already integrated in the published 0.1.41 baseline; no old concurrency
branch was copied over the newer corrected installer. The prior primary's
release and UI records were checked in. SEC-06's 25 local files matched its
public accepted source 8d3fa47fc5d2b2ff4fd5726da6a43e4f35eb852b; its scoped
code/tests were integrated without replacing later restore boundary controls.
SEC-05's restricted-token diagnostics remain experimental and are excluded.
SEC-08's previously uncommitted sanitized target record is included; private
archives, support reports, exports, manifests and raw receipts are excluded.

## Exact source and evidence

The release starts from published 0.1.41 source
e1212539511ee3413a7049e318dc0c2699e8ee0c. Every baseline file was checked against
the published tree, including line-ending identity, before applying the scoped
release changes. Every uploaded blob SHA matched the local Git blob hash.

Initial candidate d6f3f1bfe94e0a938a0b0619df6218f2722e013c passed focused
installer trust run 42145872199/job127657085863 and catalog run42145872202.
Combined run42145872113 passed unit/vet/security/disk-full/service/browser,
integrated hostile-input/media checks, actual upgrade, rollback and preservation,
then stopped at a stale test launching the previous executable with elevated
direct serve. Published 0.1.41 deliberately refuses that launch mode.

Corrected source f37f61f2fe79deb92eeb3ff614ea6b245180a391 changes only the test
fixture and workflow trigger. The real published prior executable runs under
the existing LocalService registration, with exact executable hashes checked
before and after the bounded disposable-host fixture. Runtime and installer
controls are unchanged. Corrected full acceptance run42147333212 is required.
No failed-run installer is accepted or released.

## Status and limits

After the corrected combined gate passes, original security status is six closed
(SEC-01/02/03/04/06/07) and two open (SEC-05 and SEC-08). All eight retain Group 2
classification. Group 1 has no demonstrated open original item; Group 3 remains
record-only/deferred. Existing Job Object limits are not a filesystem/token
sandbox. Signing is deferred and the package remains unsigned.

Build Notes and roadmap retain milestones 7 and 8 as in progress, with no delivered
release assigned to those incomplete milestones. Existing-installation 0.1.41
target backup/restore/hash/reinstall evidence is distinct from final-candidate
Windows 10 clean-machine, reboot, endurance, upgrade and restore acceptance.
No live installation, reboot, media change or private-data upload is part of this
release operation. Dependencies, providers, models and schema are unchanged.
