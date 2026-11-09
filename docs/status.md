## November 8, 2026 — Suggested event and trip albums

C18 is implemented on the local suggested-albums branch. Suggestions use the
visible library across folders. The first pass sorts by embedded JPEG capture
time or a recognized full date and time in the filename. Video uses the filename
fallback. Unknown capture times stay out of suggestions. Comparisons use the
recorded clock values without timezone conversion.

Events require at least three items and split after a gap over three hours,
with a 24-hour span limit. Possible trips require at least six items across dates
and a span of at least 18 hours. They split after gaps over 18 hours and are
bounded to seven days. A suggestion shows up to six neighbors on each side and
supports up to 500 reviewed items, including the neighbors. These are time-based
estimates. Landmark matching and embedded video timestamp extraction are later
work.

Saving creates a normal album and its ordered members in a single existing
catalog transaction. A matching retry returns the same album. Hidden, deleted
or replaced targets cause the entire new save to be refused. Focused Go tests
and local Edge checks passed, including a mixed photo/MP4 strip, adjustments at
and inside its boundaries, lost save response, and photo comparison regression.
Combined service and installer acceptance remains pending for the next release.

## November 8, 2026 — People directory and photo comparison

The development branch starts from the validated 0.1.44 prototype. C15 adds a
People directory based on existing named-face labels, with portraits and distinct
visible media counts. C38 adds side-by-side comparison of two selected photos,
with linked zoom and pan and per-photo favorites. Selected-original export (C27)
is deferred at the user's request.

The focused directory test and disposable Edge checks passed. The gallery
browser gate now exercises cross-page comparison, linked pan and zoom, closing
during a favorite save, filtered selection recovery, and stale gallery results.
Broader local library and web tests had identical failures and skips on the
candidate and 0.1.44 baseline under the same permissions. The candidate added
one passing directory test. Combined service and installer acceptance remains
pending for the next release.

## November 6, 2026: 0.1.44 steps 1–6 accepted development checkpoint

The released baseline, shared gallery and full viewer remain accepted. Selection
now spans pages with exact target review, durable 25-item chunks, stop/cancel/
resume and bounded receipts. Favorites work in grid, viewer and shared queries;
custom tags, reversible hiding and album membership support batch actions.
Flat manual albums support creation/edit/removal, descriptions, covers, hidden
fallback, empty/multiple albums and accessible manual order. Album and membership
removal preserve media. Shared semantic snapshots avoid resets from preview or
receipt bookkeeping while still invalidating changed result metadata.

Windows Server 2022 full regressions/vet/NTFS, LocalService/Edge, lost-response
and restart recovery, independent restore including paused work, genuine 0.1.43
refusal and actual Setup rollback passed. Full run 42254361898 and final browser
run 42259007001 provide the evidence. See
[selection-albums-0.1.44.md](selection-albums-0.1.44.md) for exact source, hashes,
contracts, retention and limits; earlier steps 3–4 remain in
[viewer-metadata-0.1.44.md](viewer-metadata-0.1.44.md).

Next: step 7 searchable captions/title/notes and source browsing, then step 8
combined filter/calendar interfaces and final combined/release gates. No release
or user-library migration occurred. Windows 10, reboot, endurance, signing and
final release acceptance remain separate. Older checkpoint text is historical.

## November 5, 2026: 0.1.42 release candidate

Based on exact published 0.1.41 source e1212539511ee3413a7049e318dc0c2699e8ee0c. SEC-06 source/tests integrated from 8d3fa47fc5d2b2ff4fd5726da6a43e4f35eb852b without replacing later SEC-03 recovery controls. SEC-05 experiments are excluded. All five workstream owners confirmed the handoff; SEC-08 sanitized target notes are included, private receipts excluded.

Security closure becomes 6 closed / 2 open only after combined candidate acceptance: SEC-01/02/03/04/06/07 closed; SEC-05 paused/open and SEC-08 open. All original items remain Group 2; Group 1 has no demonstrated open original item; Group 3 stays record-only/deferred. Milestones 7 and 8 remain in progress. Signing remains deferred. Genuine 0.1.41 upgrade, injected rollback, full providers, browser, input regression and independent recovery must pass before publishing this prototype. Clean-machine Windows 10, reboot, endurance and final candidate target acceptance remain open.

> Corrective release in progress: 0.1.40 target preparation refused normal
> ProgramData creation rights before payload replacement and left the prior
> owned service stopped/disabled. 0.1.41 fixes that bounded compatibility case;
> focused/full default-folder and service-recovery acceptance is pending.
> Private support logs remain local. See release-0.1.41-acceptance.md.

# Current status

## 0.1.40 published; target validation pending

User-requested [0.1.40 prototype](https://github.com/aataat12/immich-clone/releases/tag/v0.1.40-prototype)
is published from exact tested source5fcab695e086403c8e9bb1fdb339a7dc697b9eb6:
[run41979194020](https://github.com/aataat12/immich-clone/actions/runs/41979194020),
job127085130561, Windows Server 2022/LocalService/Edge. Genuine 0.1.39 upgrade,
injected rollback, full native providers, redesigned desktop/narrow pages,
backup and independent restored reinstall/uninstall passed. All four published
assets match the downloaded tested artifact. [Evidence/checksum](release-0.1.40-acceptance.md).

[Validation-AI assignment](validation-0.1.40.md): focused same-folder update and
existing-data/navigation regression. User 0.1.40 target results are pending;
accepted 0.1.39 results remain recorded. No routine reboot, destructive restore
drill or extra import is requested. Milestone 7 remains in progress, 5 closed /
3 open (SEC-05/06/08). No new provider/model/schema/cost/Group 3 work or final
Windows 10 certification. Prior keep-0.1.39 notes below describe earlier
checkpoints; 0.1.40 is now released for this authorized validation. Retain the
prior accepted 0.1.39 installer.

## SEC-03 final combined checkpoint accepted

Source `c8873e9cdbb197bc9d06af19c8985e69b132c8a0` passed the full focused
Windows gate41975733374/job127074014301 and the genuine0.1.39/full-provider/
independent-recovery [combined run41976076284](https://github.com/aataat12/immich-clone/actions/runs/41976076284),
job127075119754. Restore-to-service enrollment now preserves a protected private
root and private inheriting children; the unchanged held installer normalizer
grants service access. Actual restored reinstall/authentication/media/organization/
reanalysis/uninstall passed with the old state unavailable. Accepted concurrency
and existing-function UI are integrated. [Exact source, checksums and limits](sec03-combined-acceptance.md).

**Original SEC-03 is closed in the unpublished Server-validated candidate.**
Security is **5 closed / 3 open**: SEC-05/06/08 remain. SEC-03 has **0 passes left**;
this latest record supersedes older partial-checkpoint forecasts below. Next
bounded security work is SEC-05; it has not started in this checkpoint. Milestone7
remains in progress. No release publication/Windows10/reboot/endurance or physical
power-loss certification; keep published0.1.39, no PC action. Group3 stays deferred.

## Existing-function UI redesign: Server/browser acceptance

The five-page redesign passed actual LocalService/native-provider/Edge validation
on Windows Server 2022: source d1a47efe66cdec69aa340bbf3adda897280611f5,
[run41958781493](https://github.com/aataat12/immich-clone/actions/runs/41958781493),
job127019911895. Desktop/narrow navigation, original hashes, inference, import,
backup/range downloads, independent restore and failure reporting passed.
[Exact scope, screenshots and final text-only cleanup](ui-redesign-acceptance.md).
The combined-validation chat owns final integration with full concurrency evidence.
SEC-03 subsequently passed final combined acceptance above; security5 closed/3 open.
Curation/Cinema/durable review counts and library-wide filters remain deferred.
No release or Windows10 certification; keep0.1.39, no PC action.

## November 4, 2026: 0.1.39 local regression accepted; milestone 7 started

Current published prototype is **0.1.39**. The user reported all exercised combined
0.1.38/0.1.39 checks passed, with no failures. Existing sampled-video results show
12/12 timestamped positions; this is not a fresh inference execution or an
explanation of an older video failure. [Exact tested/untested scope](validation-0.1.39.md#combined-user-result).
Accepted cloud source/run/checksum: [0.1.39 acceptance](release-0.1.39-acceptance.md).

The user instructed proceeding with milestone 7 after the handoff pause.
[Milestone 7 gates](milestone7-hardening.md) and the [manual update guide](update-guide.md)
record permitted preparation, reused evidence and the remaining scope decision.
The user subsequently explicitly approved resuming original SEC-02/03/05/06/08
work. Group 3 investigations and paid signing remain deferred. Original count is now **4 closed / 4 open**. SEC-02 passed focused actual
Setup/token controls and combined genuine prior-release/full-provider/rollback
acceptance in an unpublished 0.1.41 candidate: [evidence](installer-trust-milestone7.md).
SEC-03/05/06/08 remain open; milestone 7 is in progress. 0.1.39 is still the
published prototype; no candidate installation is requested. Signing is
unconfigured; no new costs or published product installer were introduced. A
dedicated focused gate and the combined Windows Server candidate gate passed.
Candidate publication is disabled; this does not certify final Windows 10 acceptance. Final Windows 10 clean-install/reboot/endurance acceptance remains open.

### SEC-03 installer copy checkpoint

The fixed three-file snapshot/rollback helper now uses held directories/files,
checks actual private descriptors and physical role overlap, and publishes readiness
only after byte verification/Sync/Close. Focused Windows run41842717775 passed native
failure/privacy/alias controls, real production Setup injected failure/recovery,
protected custom/Program Files and existing SEC-01 compatibility. [Exact evidence](filesystem-boundary-milestone7.md#accepted-focused-installer-copy-integration).
One user-authorized worker independently reviewed the integration and added tests.
SEC-03 remains open for root/ACL, logging, initialization, Inno publication/uninstall
and full combined release gates. Count stays4 closed/4 open. No new installer or
PC action is requested. This checkpoint is complete; no workers/CI are left running.

### SEC-03 diagnostic checkpoint

Installer log append/rotation and sanitized failure-note writes now use the staged
native helper with held actual directories/files and descriptor policy checks.
Focused Windows Server2022 run41846533710/source7f4b99b passed independent native
controls, real Setup failure with hard-linked note refusal and healthy rollback,
existing snapshots and SEC-01 compatibility. [Exact evidence](filesystem-boundary-milestone7.md#accepted-focused-diagnostic-integration).
The worker completed review/tests; no workers or CI remain active. SEC-03 stays
open for root/ACL, initialization, remaining Setup paths and full combined gates.
Count remains4 closed/4 open; keep0.1.39, no PC action or new release.

### SEC-03 private-root creation checkpoint

Missing installer storage components now receive protected private permissions
at native creation through held parents. Focused Server2022 run41849734950,
source db37f6c, passed creation/inheritance/link/partial-failure controls, real
ordinary-user denial before ACL normalization, actual Setup rollback and existing
SEC-01 compatibility. [Exact evidence](filesystem-boundary-milestone7.md#accepted-focused-private-root-creation).
Worker review/tests complete; no workers/CI active. Existing-tree ACL normalization
remains explicitly unconverted; next is that bounded change, followed by remaining
init/Setup/full acceptance gates. No feature scope expansion or Group3 work.
SEC-03 remains open,4 closed/4 open; keep0.1.39 with no local action/new release.

### SEC-03 existing-tree ACL integration accepted

The prior queued proof has passed: run41862213427/source8adf2fa/job126686818594.
State/log/rollback ACL normalization is now integrated through audited held trees
with preserved explicit ACEs, protected descendants, owners and data; fresh
namespace checks detect late additions. Native and production Setup/storage
acceptance passed [run41868231145](https://github.com/aataat12/immich-clone/actions/runs/41868231145),
source989fb085d0999da2493463e7323f77feaf78aeca, job126707927278.
Actual unrelated-writer refusal restarted the safe prior service after verification;
failed-upgrade rollback, ordinary-user denial and the full SEC-01 suite passed.
[Exact evidence/limits](filesystem-boundary-milestone7.md#accepted-existing-tree-acl-integration).
The first integration run failed only a denied-fixture fresh read assertion; the
corrected preauthorized observer retained all ACL/byte assertions and the rerun
proved that fresh read really is denied. Production policy was not weakened.

User definition: one pass is a work session ending with completion/pause. Latest
forecast was7 remaining, conditional on proof/integration; this completed ACL
integration reduced it to6 remaining passes; initialization reduced it to5; the code-permission checkpoint reduced it to4; application payload publication reduced it to3; bounded uninstall/cleanup reduced it to2. Residual implementation review below corrected that forecast to5; accepted verify/backup reduced it to4; accepted restore reduced it to3; accepted interrupted-upgrade recovery below reduces it to **2**, not a guarantee.

| SEC-03 workstream | Current checkpoint | More passes, best estimate |
|---|---|---:|
| State/log/rollback ACLs, initialization, Setup payload and uninstall | Focused Windows checkpoints accepted | 0 |
| Command entrypoint guards, existing-code reads and stationary metadata aliases | Focused Windows checkpoint accepted | 0 |
| Administrative verify/backup held read-only paths and archive output | Focused Windows checkpoint accepted | 0 |
| Restore supervisor and pre-main worker path boundary | Focused Windows checkpoint accepted | 0 |
| Durable full-payload intent and interrupted/mixed-copy recovery | Focused Windows Setup kill/retry checkpoint accepted | 0 |
| Separate-account installer/metadata concurrency controls | Remaining | 1 |
| Genuine prior-release/full-provider/recovery closure | Remaining | 1 |
| Total | SEC-03 remains open | 2 |

No recovery CI remains active. No new feature/provider/version/release, cost,
Group3 work or PC action; keep published0.1.39. Security remains4 closed/4 open.
Windows Server results do not certify Windows10.

### SEC-03 interrupted-upgrade recovery accepted

Final source `649c1450b9e2d3e0e14afbb855ad8918d79f2001` passed
[run41947440346](https://github.com/aataat12/immich-clone/actions/runs/41947440346),
job126983031492 on Windows Server 2022. Persistent private sibling transaction
storage replaces temporary payload backups. Preparation resolves pending recovery
before overwriting the prior snapshot. Synced prewrite intent binds exact payload
destinations; held prior state sources are verified before code rollback. Service
startup is disabled during mixed copying, manual for candidate health, automatic
only after healthy commit. Terminal-first/terminal-deleted-last retirement admits
only audited fixed metadata and numbered backup leaves, retaining unsafe residue.

The actual production Inno Setup process tree was killed during native publication
of a synthetic 1 GiB payload. The helper was confirmed killed, service stopped and
disabled, and intent retained without completion. New bundled Setup recovered and
retried successfully; saved catalog/executable and independent access-code/original
hashes were preserved. Complete large payload, private intent against a separate
standard user, healthy commit/retirement, startup restoration, normal/fresh Setup,
explicit rollback/uninstall, native tests/build/vet and SEC-01 compatibility passed.
Corrupt prior catalog refuses before code rollback; prepared-only/empty-root and
fresh-state recovery controls passed. The runner's canonical Go formatting is
saved unchanged; no additional behavioral edits after this successful gate.
[Ordering, exact evidence and limits](interrupted-upgrade-milestone7.md).

This closes the SAME tracked interrupted-upgrade row. Best remaining passes
**3 to2**, unchanged remaining rows: separate-account installer/metadata races and
genuine prior-release/full-provider/recovery combined validation. SEC03 remains
open; security4 closed/4 open. Published0.1.39 unchanged. No provider/dependency/
application catalog schema/release/cost/Group3 expansion or PC action. Process-exit
execution does not certify physical power-loss or Windows10 behavior. All recovery
CI is complete. The concurrency and combined rows retain their own acceptance.

### Earlier SEC-03 interrupted payload recovery native component checkpoint

Development resumed from accepted restore checkpoint `9a9d89c` on a separate
`work/sec03-durable-recovery` branch. Native component source `5e17f2c` passed
[Windows Server 2022 run41944521442](https://github.com/aataat12/immich-clone/actions/runs/41944521442),
job126973302428: all installerfiles tests, helper build, module verification and
vet. Real child-process exits at four publication boundaries bypass deferred
rollback. A synced, closed private prewrite intent restores old/new/partial
contents only through the recorded destination objects, verifies all prior-copy
hashes before mutation, removes newly created leaves and supports repeated
recovery. Replacement objects and corrupt backups refuse before restoration.
The ordinary completion receipt remains required for completed-payload rollback.

This is a component checkpoint, not acceptance of the durable-upgrade row.
Setup still supplies temporary payload backup storage and does not call the new
`recover-interrupted-payload` action. Next: persistent private transaction storage,
pre-snapshot pending-transaction handling, commit/retirement and actual killed
Setup/retry validation with service-start gating. Process-exit execution does not
prove power-loss durability, full-provider recovery or Windows 10 behavior.
The accepted security-baseline branch remains unchanged. No release, provider,
dependency, application schema, cost or PC action. SEC03 stays open, security
4 closed/4 open; the same three remaining workstreams stay tracked, best estimate
3 additional passes. Native progress does not close the first row prematurely.

### SEC-03 initialization checkpoint accepted

Installer uses existing LocalService startup to initialize before health, rather
than invoking elevated -mode init. Explicit Windows elevated init now refuses
before diagnostics/state access, including identity-query failure.
[Run41873094251](https://github.com/aataat12/immich-clone/actions/runs/41873094251),
sourceb31f1aaf2b76ee681aa464714ca8c9f5c96d0cb1/job126725226588 passed actual
elevated direct/junction/hard-link refusal, fresh Program Files state creation
with LocalService file/directory ownership, unchanged command log, actual upload,
post-start failure/verified rollback and existing permission controls.
[Exact scope/limits](filesystem-boundary-milestone7.md#accepted-initialization-checkpoint).
Worker review/tests finished. One test-status plumbing correction retained every
negative assertion. Five broad fixture diagnostic-directory scaffolding additions
were reviewed; their full scenarios await combined acceptance. Product code bytes
match the accepted source. No workers/CI active, release or PC action.

Previous forecast6 now **5 remaining passes**. Other offline command filesystem
paths remain explicitly in SEC-03 integrated boundary review; no blanket elevated
CLI closure. Setup/code permissions2, uninstall1, integrated races/boundaries1,
combined full-provider/prior-release/recovery1. Scope stays original SEC-03,
security4 closed/4 open and published0.1.39 unchanged. Server is not Windows10.


### SEC-03 code-permission checkpoint accepted

Replaced the installer code-root path-based LocalService RX grant with native
held-tree auditing and handle-based DACL updates. Preserve trusted reader ACEs,
owners and protection; stop inheritance across protected descendants and their
whole subtree so private rollback catalog/access-code snapshots remain unchanged.
Code files request read-data plus security metadata, no data-write/delete, with
read sharing only; real running-image compatibility is verified.

[Run41882276092](https://github.com/aataat12/immich-clone/actions/runs/41882276092)
completed successfully on Windows Server2022, source
802e09733e31caa1a272400dbca4477b8b99c7e5/job126757536563. Complete logs verify
native reader/private-rollback preservation, unsafe grants/aliases, actual rename
and data-writer sharing refusals, late outgoing-alias detection before any DACL
write, unchanged outside objects, running own image, and an unprotected inherited
root. Actual custom and fresh Program Files Setup, LocalService initialization,
upload/original preservation, injected failure/native rollback and SEC01 passed.
Local Windows cross-build/vet passed; worker review/tests completed.

Windows can publish a new hard-link name to the same held object: sharing does
not freeze link count. Initial and before/after link-count checks refuse detected
aliases; the tested late pre-apply alias refuses before every ACL write. These
checks are not atomic with arbitrary later interleavings. No retroactive revocation
of existing WRITE_DAC/owner authority is claimed. Final integrated race/offline-mode
coverage remains open. For admitted ALLOW-only ACEs, post-write verification uses
an exact SID/mask/flags multiset including duplicates because Windows may reorder
inherited ALLOW entries; protected descriptors and stale-plan checks remain exact.
Failed fixture assumptions and the reproduced ordering failure are retained in
filesystem-boundary-milestone7.md; no production trust policy was weakened.

This completes only the code-permission half of Setup. Inno payload publication
is unchanged and remains next. Forecast **5 to4 remaining passes**: Setup payload
publication1, uninstall/cleanup1, integrated boundaries/races/offline modes1,
combined genuine prior-release/full-provider/recovery closure1. SEC-03 remains
open, security4 closed/4 open. No feature, provider, schema, dependency, version,
release, cost, Group3 work or PC action. Keep0.1.39; Server does not certify Win10.


### Accepted Setup application-payload publication checkpoint

[Windows Server run41894239272](https://github.com/aataat12/immich-clone/actions/runs/41894239272)
completed success, source30fa427021139265791f36a14409130adb13b4b9/job126799357326.
Native held-directory/leaf publication now installs exactly seven rootfiles and
media/ai. All targets are audited before overwrite; complete-code byte snapshots
support checked copy/Install-failure recovery before old3file/state restore/Health.
Independent distinct-prior/candidate recovery, newfile deletion, unsafe ACL/link/
writer/shape/overlap/corruptbackup/missingtarget and late-name controls passed.
Actual minimal custom/fresh Setup, fixture rollback, fixed native-file uninstall
tracking, originals/privacy and SEC01 passed. Worker review/tests completed.
[Exact evidence and limits](filesystem-boundary-milestone7.md#accepted-setup-application-payload-publication-checkpoint).

Forecast **4 to3 remaining passes**: uninstall/remaining Inno metadata1; integrated
races/offline commands/interruption handling1; combined genuine prior-release/
full-provider/recovery1. Temp fullcode snapshots do not prove durable power-loss
transactions. Inno-generated metadata/registry/shortcuts remain built-in; hostile
uninstall and fullprovider/interruption closure remain original SEC03 scope.
No new dependency/provider/schema/version/release/cost, Group3 work or PC action.
SEC03 open, security4 closed/4 open, keep0.1.39; Server does not certify Windows10.
Worker and focused CI completed. Paused at this checkpoint.

### Accepted native uninstall and cleanup checkpoint

[Windows Server run41898776310](https://github.com/aataat12/immich-clone/actions/runs/41898776310)
completed success, sourcec6d384eca9d675a5053f66e60d0354821fff1f61/job126815286190.
Hash-verified staged bootstrap and persistent directory pins replace retained
unchecked-script/path cleanup. Native selected held-target deletion preserves
state/originals/unknown files; service registration is removed only after cleanup.
Actual custom/fresh uninstall and modified-script/hardlink/real-writer refusals
passed with unchanged prior service PID/metadata/original/outside bytes on refusal.
Native audit/delete/late-name/overlap/persistent-pin and exact Inno metadata
sharing compatibility controls, installation/rollback and SEC01 passed. First
normal refusal was diagnosed and corrected without relaxing payload sharing or
actual owner/DACL policy. Worker review/tests finished.
[Exact evidence and scope](filesystem-boundary-milestone7.md#accepted-native-uninstall-and-cleanup-checkpoint).

Forecast **3 to2 remaining passes**: integrated concurrent/metadata-leaf boundaries,
offline modes and interruptions1; genuine prior-release/full-provider/recovery1.
Inno loaded-uninstaller integrity is trusted, not retroactive tamper recovery;
whole metadata-leaf concurrency and temporary fullcode/powerloss limits remain
original SEC03 scope. Partial deletion does not promise transactional rollback.
No new feature/dependency/provider/schema/version/release/cost or Group3 work.
SEC03 open, security4 closed/4 open, keep0.1.39, no PC action. Server is not Win10.
Worker and focused CI complete; paused at this checkpoint.

### Accepted command-entrypoint and stationary metadata tranche

[Run41905712864](https://github.com/aataat12/immich-clone/actions/runs/41905712864)
completed success, sourcef8f1c0a1fbd7582b6fc54d2baa7294ca42fe58c5/job126839683829.
Elevated serve/direct service and unknown modes now refuse before file access;
service requires actual SCM+LocalService. show-code reads existing held code,
without Store.Open/code generation/log writes. Actual alias/no-write/read-positive
matrices passed. Native Prepare audits stationary metadata hardlinks before
builtin writes; actual valid journal alias preserved outside bytes and prior PID.
Rollback test separately proves verified-health failure and recovery. Actual
LocalService/custom/fresh install, uninstall, privacy/originals and SEC01 passed.
Worker review/tests complete. One test-only process-exit capture fix retained
all exact assertions and unchanged product bytes on rerun.
[Evidence and residual implementation](filesystem-boundary-milestone7.md#accepted-entrypoint-and-stationary-metadata-checkpoint).

Corrected best estimate **5 additional passes**, planning range5–7. Previous2
undercounted implementation as validation. The table decomposes original SEC03:
admin verify/backup1; restore supervisor/worker1; durable interrupted payload1;
actual installer/metadata concurrency1; genuine prior-release/fullprovider1.
Admin recovery commands remain supported and UNCHANGED pending safe hardening;
blanket elevated refusal would break the published recovery workflow. Pre-main
worker, temporary fullcode intent/recovery and concurrent metadata-leaf gaps
remain open. This tranche does not close the full integrated row or SEC03.
Security4 closed/4 open; keep0.1.39, no new feature/dependency/release/cost/Group3
work or local action. Server is not Win10. Worker/CI finished; paused here.

### Accepted administrative verify/backup checkpoint

[Run41913488167](https://github.com/aataat12/immich-clone/actions/runs/41913488167)
completed success, source348dd18b790fcc47e6ccf5228a723be7f0fbd125/job126866916248.
Read-only existing-store/held catalog+original reads replace offline initialization;
ZIP catalog streams directly without source staging. Archive is exclusive/private
at creation; errors clean only its held new file, with truthful incomplete-close/
cleanup reporting. No source/log/schema/index changes. Normal browser/service
backup path remains. Native real aliases/writers/bounds/privatearchive/failure/
no-generation and independent restore controls passed. Administrator commands on
actual installed source and real RX-only account commands on verified library
copy passed with exact source/ACL/namespace/no-log checks; ZIP private permissions,
manifest hashes and independent administrator restore/verify also passed.
Two workers finished review/tests. Test-only permission teardown was isolated
into a copy without product or assertion weakening. Actual remaining Setup/
uninstall/rollback/privacy/SEC01 and automatic existing catalog baseline passed.
[Exact evidence/limits](filesystem-boundary-milestone7.md#accepted-administrative-verify-and-backup-checkpoint).

Same tracked row complete; best additional passes **5 to4**. Four remaining rows
are unchanged: restore supervisor/pre-main-worker1, durable interrupted payload1,
actual metadata/installer concurrency1, genuine prior-release/fullprovider1.
Trusted recovery compatibility here does not close the separate untrusted restore
boundary. SEC03 stays open, security4 closed/4 open. No feature/dependency/provider/
schema/version/release/cost/Group3 expansion or PC action; keep0.1.39. Server is
not Win10 certification. All CI/workers done; paused here.

### Accepted restore supervisor and pre-main worker checkpoint

[Run41922662936](https://github.com/aataat12/immich-clone/actions/runs/41922662936)
completed success, source39d7a2e6f05086d5c329326749c0efa53d82f375/job126898869323.
Windows restore now holds the selected nofollow archive, creates exclusive private
staging and passes archive/stage capabilities over an inherited pipe to the bounded
same-image child. Public legacy path-request invocation refuses before file access.
Private stage/derived directories exist before extraction, with privileged owner
for elevated recovery. Publication is held native no-replace; cleanup inventories
held descendants and reports retained private staging if safety cannot be proved.
Existing ZIP/checksum/catalog validation, budgets, hard stop/reaping and recovery
metadata behavior passed. Sibling ZIP recovery, live progress writer exclusion,
actual junction/link/hardlink/writer refusal, forged invocation, late target and
late unsafe cleanup passed. Native nonempty-parent reparse protection has an actual
FSCTL positive control. Actual administrator recovery from a separate RX user's
backup, Setup/rollback/uninstall and closed permission-policy compatibility passed.
Two workers completed focused code review/tests. Initial native sharing conflicts
were corrected; test-only process/junction snapshot defects were corrected without
weakening exit/status/refusal/preservation assertions. Automatically triggered
catalog baseline41922663060 also passed.
[Exact evidence and limits](filesystem-boundary-milestone7.md#accepted-restore-supervisor-and-pre-main-worker-checkpoint).

Same tracked row complete; best additional passes **4 to3**. The three remaining
rows are unchanged: durable full-payload interrupted recovery1; separate-account
installer/metadata concurrency1; genuine prior-release/full-provider/recovery1.
SEC03 remains open, security4 closed/4 open. No feature/dependency/provider/schema/
version/release/cost/Group3 expansion or PC action; keep0.1.39. Windows Server2022
execution does not certify Windows10. All CI/workers complete; paused here.

## Historical checkpoints (superseded where later acceptance applies)

The chronological snapshots below are retained as history. Their old current/
next-release/held-candidate wording does not override the 0.1.37/0.1.38/0.1.39
acceptance records or reopen already resolved bounded defects.

**Target Windows10 validation of0.1.30: sampled video FAIL, cause unknown.**
User confirmed version/data, dismissal/restore, photo context, authentication and
normal page navigation. HEIC AI and narrow layout were not exercised. One existing
MP4 queued then failed with generic local AI task error and0/0 coverage. Do not
claim6A target acceptance or a root-cause fix. Local candidate corrects obsolete
HEIC/video help, retains known targets after first-sample failure, and adds fixed
AI task stages plus numeric native exit codes to private-safe diagnostics.
Previous controls are hidden on the first page by design (100 faces/50 photos
per page) and appear after forward navigation; user pagination remains partial.
Need private app diagnostics and nonidentifying clip dimensions/duration to
narrow native failure; no personal media or raw support files in this repository.
No replacement installer has been published for this report.


**Released0.1.30-prototype**: [installer/release](https://github.com/aataat12/immich-clone/releases/tag/v0.1.30-prototype).
Combined acceptance [41569614980](https://github.com/aataat12/immich-clone/actions/runs/41569614980),
source042547e877f0c8957b7029ff6aa6d4ad4da9f8b0, job125699201968, passed on
Windows Server2022/Edge. Actual original0.1.29 Setup was hash-verified and installed;
failed cross-provider upgrade recovered0.1.29, successful upgrade deployed0.1.30,
installed6A HEIC/video execution passed, injected0.1.31 failure recovered0.1.30
including sampled observations/corrections, and independent restore/reinstall
regenerated HEIC/face/frame context. Full suite/vet/dependency, real NTFS full-disk,
normal browser/session/CSP and payload-canary checks passed. Full Scan deferred.
Installer104160014 bytes; SHA256 ec78e9d62b4d462f3b1448310dad1cbda30cf278509af20426bbdcd49018a007.
Public release is an unsigned prerelease. Windows10 local acceptance is next;
no clean-machine/reboot/endurance or full compatibility certification. Existing
Group2/3 mitigation remains paused, original security counts unchanged. Older
held-candidate/next-installer statements below are historical.


6A failure-recovery checkpoint passed [41567782811](https://github.com/aataat12/immich-clone/actions/runs/41567782811),
source718eab96aa5ef281524de492498757d8b74469b3, job125693367010. Actual
LocalService/Edge checks covered unavailable-decoder startup, runtime decoder loss/
HEIC retry, denied derived-crop writes, missing SFace/retry and forced service death
after observing an active native child. Completed sample records, face IDs, names/
tags and original SHA256s survived; the child was reaped and queued work completed
after restart without duplicate timestamps. Final independent restore/crop/context
regeneration passed for2 clips/3 HEICs. No full-disk or Windows10 certification
is implied. Full Scan proposal is recorded but deferred beyond6A.
Next: update normal installer acceptance from the old0.1.27 fixture to the actual
installed0.1.29 baseline, then validate the combined candidate/rollback. No new
installer/version/main publication and no user action needed for this checkpoint.


Milestone6A functional candidate passed focused Windows Server2022 LocalService/
Edge acceptance [41564920558](https://github.com/aataat12/immich-clone/actions/runs/41564920558),
source9c15c56dba52481f98d8b255e98303f6ce29af7b, job125684028344.
Actual single/grid12MP HEIC face inference, persistent video pause/restart/cancel/
resume, 12 sampled positions with timestamped face/cup observations, saved frame
viewing, desktop/narrow overflow, original SHA256 downloads and independent
restore/crop/context regeneration passed. New mock job regressions also passed on
Windows; local full Go suite/vet and focused race tests passed. Shared model
identity retains review-only still/video suggestions; policy is stored separately.
No runtime dependency was added and no native rebuild was required.
No installer/version/main publication or Windows10 certification. Remaining6A
feature/release gates are heic-video-ai.md; older planned-only statements below
are historical. Group2/3 security work remains paused.


Latest classification clarification: only original SEC-01..08 belong in Groups
1/2; all are currently Group 2 under the recorded loopback-only deployment.
No demonstrated open Group 1 item; no further Group 2 mitigation work. Additional
R-RESTORE-01/NR-01/Node platform observations are recorded in Group 3 without new
work. Original SEC-04 security protection remains validated; the unrelated normal
restore observation is not claimed fixed. Current security count3 closed/5 open;
prior combined reopening statements below are superseded by this classification.
Milestone 6A implementation continues; see security-priority-groups.md.


Current user direction (November 2, America/Chicago): active security work is
Group 1 remote Internet reachability introduced by this app/components. Other
Group 2 mitigation is paused, with findings retained; security-priority-groups.md
is authoritative. No confirmed open Group 1 exploit is demonstrated in current
records. Milestone 6A implementation proceeds: bounded full HEIC analysis frames
and actual-PTS video extraction are being executed before job/UI integration.


2026-11-03: NR-01 technical remediation is implemented and accepted in the held
application candidate: six pinned upstream OpenCV fixes plus a separately recorded
single-plane mean correction. Actual native/API/model/LocalService/restart evidence
is opencv-nr01.md. Application [41554459739](https://github.com/aataat12/immich-clone/actions/runs/41554459739), source
ae1fbc3011de33df9a1c25594b6f9864f535bdb8, job 125649834966 passed real model/browser,
missing-provider/model recovery, names/tags/original preservation and independent
restores. The old installed build is unchanged; no installer/version/main merge.

HEIC/video AI is explicitly planned as milestone 6A in both roadmap and Build Notes,
with heic-video-ai.md acceptance. Existing 320px HEIC thumbnails are not an AI
analysis-frame implementation. No HEIC/video AI feature is claimed delivered.
Next functionality focus after this servicing is bounded analysis-frame feasibility.

R-RESTORE-01: original SEC-04 is reopened for one observed valid-restore refusal
in 41553917677 (media cancellation/restart/restore test). Cause is unconfirmed.
Fixed private-safe stage/OS-code diagnostics were added without weakening limits;
20 focused Linux repetitions, 50 Windows repetitions, the full suite and actual
application CLI independent restores subsequently passed. That is revalidation,
not a root-cause fix. Investigate before release; see restore-revalidation.md.
Current fixed list: **2 closed / 6 open**, SEC-01/07 closed. Earlier closure
statements below are historical checkpoints, not the current SEC-04 disposition.

Four authorized workers completed original SEC-04 hostile backup limits and
SEC-07 review closeout, each with implementation and independent verification.
Combined Windows Server 2022/Edge acceptance [41547447887](https://github.com/aataat12/immich-clone/actions/runs/41547447887),
source 1f1cdb912b223634dec0350e58e9ae86b11893b0, job 125626387613, passed in full.
At that historical checkpoint the fixed original list had **3 closed / 5 open**: SEC-01, SEC-04, SEC-07 closed;
SEC-02, SEC-03, SEC-05, SEC-06, SEC-08 open. No item renaming or partial-probe closure.
See security-worklist.md, security-restore-limits.md and security-review-closeout.md.

Restore limits cover full ZIP/catalog work in a supervised worker, strict copied-byte
budgets, finite wall/committed-memory limits, growth detection and unpublished-failure
cleanup. Actual Windows memory-positive/excess controls, cancellation/reaping,
hostile catalogs, CLI default/explicit valid old-format restoration and seven
low-budget refusals passed. Catalog monitoring is not an atomic disk quota.
Review closeout completes threat/advisory/control/servicing coverage; it found
NR-01, an OpenCV NaryEltwise/core-safety release blocker requiring provider
remediation and actual validation under SEC-06/08. Review closure does not fix or
accept that risk, and native child filesystem isolation remains SEC-05.

Primary functionality adds independent Previous faces / Previous photos navigation.
Actual Edge checks passed with 102 persisted synthetic faces and 52 synthetic
originals: prior-page identity, independent cursors, selection and filter resets.
All five pages, desktop/narrow, CSP/session and existing People corrections passed.
See people-review-navigation.md. No user-media enlargement or product installer.
No Windows 10 certification, main merge or release/version change.

People candidate adds photo context from active/dismissed face review without
mutating data. See people-photo-context.md; Windows Server 2022/Edge acceptance
passed (41535508518), including desktop/narrow and error/close behavior.

People filtering candidate: server-side unnamed filtering and accurate search/
face continuation. Linux race/full tests/vet and Windows Server 2022/Edge acceptance passed
(41525562509). See people-review-paging.md. User-authorized workers completed
staged-Setup preparation and native-child authority probes, with exact evidence in
security-upgrade-preparation.md and security-worker-canary.md. The former passed
its focused Setup fixture; the latter proves same-token state access remains
allowed despite working resource limits. Wider security gates remain open.

People candidate: reversible false-face dismissal/restoration preserving names,
undo history, tags, analysis and originals. Linux focused race/full tests/vet
passed; Windows Server 2022/Edge acceptance passed (41519696666), including
actual dismissal/reload/restore. See face-dismissal.md. Security worker source reviews are in
security-installer-review.md and security-worker-isolation-review.md; neither
claims new runtime evidence or clears the outstanding security release gates.

Next-release candidate: 64 accepted TCP connection cap and rolling sixty-second
response write-idle bound. Linux real-socket race tests/vet passed; focused Windows
validation passed on Windows Server 2022/Edge (41501827158). See
security-baseline.md S-008. No installer/version changed.

Next-release restore candidate: validate the complete manifest/ZIP integrity and
reject noncanonical or special-file archive entries before extraction. Linux
negative/valid-restore tests and vet passed; Windows Server 2022/Edge validation
passed (41510214737, S-009). Next security review: parent-folder/reparse and ACL
boundaries, aggregate archive/catalog work and native isolation.

Next-release installer candidate replaces root DACLs instead of retaining unknown
explicit grants and preflights reparse components. Disposable Windows acceptance
passed on Windows Server 2022 (41513726020, S-010): ordinary-user access checks,
junction preservation, actual production helper/LocalService upload and restart.
Bundled new preparation now passes actual Setup/junction/current-runtime
LocalService execution (41525562509). Child ACL/owner/inheritance auditing is now implemented and tested as SEC-01.
Writable custom AppDir, concurrent races, staging-token access and actual old-package full upgrade/rollback
remain gates before packaging. No release installer/version changed.

## Current cloud-accepted prototype: 0.1.29

[Acceptance 41487369408](https://github.com/aataat12/immich-clone/actions/runs/41487369408),
source f2242c2942caf1cc923a1ea7ad708900a8a3ad5a, job 125421140599 passed Windows
Server 2022. Bundles explicit login/sign-out/current-session revocation, rolling
copy-idle and admission safeguards, legacy export overlap limits, responsive AI
status and the verified static zlib 1.3.2 worker. Model weights/schema unchanged.
Provider directory: opencv-4.12.0-zlib-1.3.2-cb35f0ccb3e3; old immutable provider
remains available for rollback, not selected by the new executable.

Actual restricted-service browser/native/backup/import/paging tests passed, plus
old 0.1.27 inference/names/tags retained without rerun, failed cross-provider
rollback back to 0.1.27/old worker, successful 0.1.29 upgrade, actual new-worker
inference on a distinct original, installed-payload code/private-marker scan,
0.1.30 failed-upgrade rollback, independent restore/reinstall, original hashes
and real NTFS full-disk recovery. Linux tests/vet passed. No dependency install.

Artifact 13161554725 retrieved; installer 104117297 bytes SHA256
646a47acc8ac77751c8343202d12c09eff0a754fd0731f976b0b7e8fc9dcea66
matches manifest. ZIP SHA256
458c02d6b316f8054d068565a5815159e20a197472d3e6161812f1e535416493
verified. Browser artifact 13162742432 ZIP verified; login-narrow, People-narrow
and Build Notes desktop screenshots inspected. Unsigned prototype, seven-day
artifacts. Source remains on work/security-baseline/PR #2; no main merge claimed.

User-reported 0.1.29 local version/data, sign-out/confirmation, wrong-code
rejection, manual unlock, refresh persistence and five-page desktop/CSP checks
passed, with no observed failures. Updated native inference and narrow layout
were not tested; see local-validation.md. No repeat of passed checks requested.
Do not enlarge the library merely to force AI inference: use an already
unprocessed photo if available, otherwise leave new-provider Windows 10 inference
pending for normal use. No full-library import or reboot requested. Broader
security/People milestone, clean Windows 10, long endurance and wider resource/
storage/isolation review remain open. Server success is not certification.

## Working boundaries

Authoritative requirements: ../Immich_Clone_Project_Requirements.md including the
Server-first amendment. No new charges. User authorized two bounded security workers alongside People
feature development on 2026-11-02; release security gates remain. Never publish private
machine details, media/inventories/paths or raw support logs. Public/synthetic
fixtures only. Ask the user before a security mitigation conflicts with the agreed
plan. Repository visibility remains unchanged; decision deferred. CI time is UTC.
Older acceptance checkpoints remain in Git history and dependency-evidence.md.

## User-reported Windows 10 checkpoints

0.1.15: all twelve People/tag review-flow checks passed; original-photo comparison,
not newly reported SHA256 proof. 0.1.17: Bulk import worked well. 0.1.19: Home
navigation, desktop/narrow layout/version and data preservation passed. 0.1.21:
requested gallery checks passed. 0.1.25: backups, Build Notes and 87/87 eligible
photos completed with names/tags/data retained; desktop passed, narrow not repeated
and new-import follow-up not tested. 0.1.27: version/upgrade, five-page browser/CSP,
desktop and focused existing media/organization/import/backup/Build Notes passed;
authentication-flow checks were unavailable because controls were missing.

0.1.29: user-reported version/upgrade and existing photos/names/tags/ready AI,
pagination/import history passed; sign-out confirmation, incorrect-code rejection,
manual re-login and authenticated refresh/navigation passed. All five pages and
AI batch status worked at desktop width without blocking JavaScript/CSP errors.
No failures observed. Updated runtime inference was not tested (no eligible
unprocessed photo); unsupported video remains outside still-photo AI. Narrow
layout was not tested because that viewport was unavailable in the test surface.
Do not repeat passed checks or enlarge the library solely for testing. Native Windows 10 inference can wait for an already
unprocessed photo from normal use. These reports do not certify broad recognition,
clean-PC prerequisites, endurance or reboot startup. Reboot remains at final gate.
No raw private reports retained. See local-validation.md for the recorded result and remaining scope.

## Preserved gallery/import/AI baseline

0.1.21 indexed gallery renders/requests <=60 assets with capture/upload/case-folded
name sorting, visibility filters, Next/Previous and validated stale-cursor recovery.
Indexes/counts commit with assets/import receipts and rebuild after open/restore/
older writers. Legacy full-list API remains for explicit older/test callers only.
[41391526713](https://github.com/aataat12/immich-clone/actions/runs/41391526713)
passed 600 actual mixed native-image originals/previews/recovery/independent restore
and 121 actual browser PNGs across three pages with no full-list reads.
[Scale 41391526782](https://github.com/aataat12/immich-clone/actions/runs/41391526782):
50k metadata-only rows, page+JSON 188592 Go allocation bytes, one rebuild 827.413ms.
Not RSS/peak memory, a speed guarantee or a 50k-media ingestion result. Prior
native worker/model/licensing checks are reused; no source rebuild of providers.

## Next milestones and remaining limits

1. Large-container/ZIP64 size and throughput, longer actual mixed-media endurance,
   and storage-disconnect recovery remain open before full real-library import.
   Web backups still stage a complete archive plus catalog/reserve in application
   storage; browser destination needs additional space. No external destination
   picker, incremental backup, automatic retention/scheduling or backup-byte resume.
2. Import still limits selection to 50000 nonempty files, 16MiB manifest, 4GiB/file,
   128 sessions and one active copy lease. Resume retransfers accepted bytes for
   SHA256 receipts; no metadata-only skip or partial-file continuation. See bulk-import.md.
3. People/tag scalability and wider family/age/profile quality remain separate
   gates. Native still-image AI only; HEIC/video AI is planned in milestone 6A; ready reanalysis remains deferred. Face dismissal and
   photo context have accepted next-release candidates; they are not shipped. See ai-feasibility.md.
4. Hidden Media local-location GUI and deliberate permanent managed-copy deletion
   with recovery remain queued: hidden-media.md. No deletion implemented.
5. Clean Windows 10 prerequisites, final reboot/startup, security/signing remain
   open. User defers reboot to the end; setup needs none. Repository visibility
   decision remains deferred and unchanged.

Managed originals remain immutable SHA256 copies; external sources stay untouched.
Persistent jobs, loopback/code-cookie auth, restricted LocalService, metadata/
orientation, previews, downloads/export and consistent restore retain documented
bounds. RAW/HDR/high-bit-depth and broad camera coverage remain open. Do not repeat
old diagnostics or completed small local checks without a relevant change. Exact
dependencies/evidence: dependency-evidence.md; choices: decisions.md; local checks:
local-validation.md; partner tasks: partner-handoff.md. Older status is in Git history.
