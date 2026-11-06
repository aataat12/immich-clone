# SEC-08 target validation — November 5, 2026

Status: **open; final combined release acceptance blocked**. This record
distinguishes actual Windows 10 checks from historical Server evidence.

## Baseline and candidate dependencies

The installed baseline is published 0.1.41-prototype. Its exact installer SHA256
is `2f61cca971e9d4bcbdd390f3809e9fcc55c7c1988ce40ce2e399dda799e2740a`.
Both retained installer copies matched that identity in this session.
[Published acceptance](release-0.1.41-acceptance.md) includes the genuine prior
0.1.39 upgrade, injected rollback, full providers, independent restored reinstall
and browser checks on **Windows Server 2022**, not Windows 10.

The SEC-06 chat's latest correction says its accepted separate candidate still
requires integration and validation against the current 0.1.41 baseline.
Its final recorded source is `d9f4146a1e0138e80d0e21a50f7b83cad2691439`,
[Server run42076364254](https://github.com/aataat12/immich-clone/actions/runs/42076364254).
Do not apply the older checkout's SEC-03-open/0.1.39-current summary to this
baseline: SEC-03 is closed. SEC-05 remains paused/open. No frozen combined
SEC-05/06 release candidate or new-version installer was established here.
These are prerequisites to final combined acceptance, independent of target access.

## Authorized target testing and executed checks

The user reported no disposable Windows 10 machine, then explicitly authorized
testing the current installation provided its media is backed up and the backup
retained. This supersedes earlier instructions against local test actions.
No clean-machine status is inferred from an existing installation.

| Check | Actual Windows 10 result |
|---|---|
| Consistent backup | PASS: authenticated application backup downloaded outside live state; archive checksum saved; exact application executable retained |
| Independent-folder restore | PASS: application restore completed; offline verify passed; every live original matched restored SHA256; access-code hash matched |
| Restored application | PASS: separate loopback instance authenticated with restored code and exported matching organization; asset identity/name/hidden checks matched; no new inference claim |
| Service stop/start | PASS: administrator-controlled stop, offline verification, start; every original hash unchanged; automatic LocalService configuration retained |
| Clean installation without development prerequisites | NOT TESTED: current machine is an existing installation |
| Full OS reboot/automatic startup | NOT TESTED at this checkpoint; service restart is distinct |
| New combined version upgrade and injected rollback | BLOCKED: no integrated accepted candidate; older Server results cannot clear target gate |
| Uninstall/reinstall preservation | PASS: after user-requested elevation retry, actual 0.1.41 uninstall/reinstall both exited 0; service/code removal observed, catalog/access-code/original hashes preserved, authenticated access and complete organization/stable asset metadata matched after reinstall; automatic LocalService retained. This is same-version recovery, not new-version upgrade behavior |
| Extended operation | NOT TESTED: bounded checks do not establish endurance |
| Final candidate restore drill | BLOCKED: baseline drill above is not final candidate acceptance |

Private archives, manifests, exports, logs and detailed receipts stay outside
the source repository and must not be uploaded to CI/public issues. The retained
archive and restored copy are on the same physical disk as the live installation:
they protect against installation mistakes, not loss of that disk. Retain all
copies until final validation and user-directed cleanup; no automatic deletion.

The first non-administrator service-control attempt was refused; the service
remained running. A subsequent Windows-elevated controlled stop/start succeeded.
The first uninstall/reinstall elevation returned "The operation was canceled by
the user". The user then explained they were away and explicitly requested another
prompt. That retry succeeded: actual uninstall/reinstall and preservation checks
passed. Private receipt records both installer exits as zero and all preservation
assertions passing. The service is running. No OS reboot, scheduled test, or
endurance process was started. The separate restored test process was stopped
after its authenticated export comparison. All recovery copies remain retained.

## Remaining acceptance protocol

1. Integrate SEC-06 into the current baseline, resolve SEC-05 before final security
   release acceptance, and freeze one exact source/provider/model/installer identity.
   Run combined genuine-prior-version upgrade, injected rollback, provider,
   browser, privacy/integrity scan and independent recovery gates on that identity.
   Do not rebuild or overwrite published 0.1.41 assets.
2. On a legitimate clean Windows 10 target without SDKs or assumed dependencies,
   install that exact package and prove LocalService startup, authentication,
   image/video operations and actual pinned inference. Record prerequisite
   behavior and unsigned-installer blocking honestly; never disable protections.
3. Reboot the actual target and prove a changed OS boot session, automatic service
   startup under LocalService, authenticated access and original/organization
   preservation. A manually started service or service-only restart cannot pass.
4. Upgrade from the genuine accepted prior package on Windows 10; compare all
   original hashes, access code and logical organization. Exercise injected
   failure/rollback on a disposable fixture, not by corrupting a live library.
   Uninstall must remove service/code while retaining state; reinstall must recover
   authentication and organization. Keep the known-good package and backup.
5. Define and record the intended workload, dataset scale and duration before
   endurance acceptance. A proposed minimum is a measured 24-hour session with
   periodic authenticated health checks, representative real media/AI work,
   backup completion, job recovery and before/after hashes; it is not an existing
   project-agreed closure threshold or proof of natural full-library reliability.
   Record failures and resource trends privately; no arbitrary waiting-only pass.
6. Take a fresh final-candidate backup, retain a verified separate-storage copy,
   restore to a new destination, verify all originals and organization, and prove
   restored service enrollment with old state unavailable in a reversible fixture.
   Preserve the live library and all recovery copies throughout.

## Recovery and limitations

- Follow [manual update recovery](update-guide.md) and the backup's matching
  instructions. Check exact installer/archive hashes before use. Checksums establish
  byte identity, not publisher authenticity or malicious-replacement resistance.
- Restore only to a nonexistent destination outside sync folders. A nonzero exit
  or partial staging directory is not success. Preserve the archive on refusal;
  never merge partial catalog/original files into live state or bypass budgets.
- If installation fails, identify the running version/service state first. Retain
  the original state and backup; avoid manual binary/catalog swaps or ACL changes.
  Use private diagnostics. If uninstall succeeded but reinstall failed, retain the
  unchanged data and retry only the verified known-good installer against that
  same data location after diagnosing the refusal.
- Separate-folder CLI restore does not enroll the restored library as the Windows
  service. A tested activation procedure remains required. The historical
  [recovery guide](recovery-guide.md) has an older example version; use the actual
  backup-producing version or explicitly proven compatibility, not that example
  as an instruction to downgrade.
- Unsigned prototype publisher identity/SmartScreen acceptance is unverified;
  paid signing remains deferred. Native worker isolation remains open. No physical
  power-loss, disk-failure, broad format/recognition quality or long-term servicing
  claim is established by these bounded tests.

SEC-08 may close only against the final frozen candidate with the required target
evidence. Documenting a limitation does not waive a required gate.
