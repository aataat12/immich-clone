# Milestone 4: import boundary and recovery evidence

This is one completed bounded acceptance slice for the existing advertised per-file
limit. It does not raise that limit, finish milestone 4, establish real-library
endurance, certify Windows 10, or prove 4 GiB video decoding/AI/backup behavior.

## Gap and fix

Earlier accepted evidence covers 600 actual mixed native-image imports, real
partial-body socket disconnect, reselect/SHA256 replay, duplicate accounting,
independent receipt restoration and import ENOSPC. Those checks did not execute
an actual file at the advertised 4 GiB limit.

The opt-in test `TestImportExact4GiBHTTPBoundary` now runs the real authenticated
HTTP server handler over a loopback TCP listener. It streams deterministic input
from a 64 KiB generator; no multi-GiB input buffer or retained source fixture is
created. Its disposable state requires at least 12 GiB free before any large I/O.
The normal test suite skips this large-I/O test unless explicitly opted in.

The first actual boundary run accepted the exact limit and hash-checked replay,
but found that a 4 GiB + 1 byte body returned generic HTTP 500. The import handler
now classifies `http.MaxBytesError` / `library.ErrTooLarge` as HTTP 413, matching
the ordinary upload route. Accepted receipt, copy, error and limit semantics are
unchanged. Production `internal/library/import.go` was not modified.

A fast normal-CI regression injects the same HTTP size-error after a partial read,
then checks 413, unchanged earlier original/receipt/counters, no acceptance of the
partial item and no retained temporary upload. The opt-in test supplies the
separate actual multi-GiB execution proof.

## Executed evidence

Local Linux, Go 1.27.3, real loopback HTTP listener and filesystem, with listener
and catalog close/reopen after interruption. This is not a Windows service,
browser file picker, network-share disconnect or installer execution.

| Phase | Actual input / output | Verified result |
|---|---:|---|
| Earlier accepted original | 63 bytes | Retained across all failures and restart |
| Interrupted source | Reader produced 8,388,608 bytes before `io.ErrUnexpectedEOF` | Large item accepted bytes stayed zero; saved error and earlier receipt survived; partial upload removed |
| Exact-limit import | 4,294,967,296 bytes physically stored | HTTP 200; original size and SHA256 receipt matched generated stream |
| First original readback | 4,294,967,296 bytes over HTTP | SHA256 matched input |
| Unchanged receipt replay | 4,294,967,296 bytes | HTTP 200; same receipt and no double counting |
| Same-size final-byte change | 4,294,967,296 bytes | HTTP 409; original receipt/counters unchanged |
| Over-limit body | Source produced 4,294,967,297 bytes | HTTP 413; temporary upload removed and original retained |
| Second original readback | 4,294,967,296 bytes over HTTP | SHA256 still matched original input after both refusals |
| Finish/integrity | 4,294,967,359 bytes accepted in 2 new files, 0 duplicates | Session finished; `Store.Verify` passed |

After the classification fix, the complete opt-in run passed in 45.594 seconds.
Observed phase elapsed times were 5.844 s exact import, 14.535 s unchanged replay,
8.700 s changed replay and 5.245 s over-limit refusal. These are one disposable
local run's timings, not throughput promises or benchmarks. The initial failing
run took 70.843 s and established the HTTP 500 defect; it is not erased by the
subsequent pass. An earlier capacity-guard attempt correctly stopped before large
I/O because the default temporary filesystem lacked the required space.

## Reproduce in a cloud/disposable environment

Normal fast checks:

```sh
go test -race -count=1 ./internal/web -run 'TestImportBodySizeError|TestImportAuthenticationManifestAndReceipts'
```

Actual boundary on a sufficiently large temporary parent:

```sh
IMMICH_IMPORT_BOUNDARY=1 IMMICH_IMPORT_BOUNDARY_TEMP=/path/to/disposable-parent \
  go test -mod=readonly -v -count=1 -timeout=10m ./internal/web \
  -run '^TestImportExact4GiBHTTPBoundary$'
```

For Windows cloud execution, use `scripts/Test-ImportBoundary.ps1` with an existing
temporary parent, for example `-TemporaryParent $env:RUNNER_TEMP`. It restores its
environment flags afterwards. The test creates/removes only its own random child
state directory. Do not run it on a user's real library or ask for local setup.

## Remaining gates

- Windows cloud execution of the exact same HTTP boundary and fast regression: passed in the run below.
- Actual packaged LocalService/browser large-file selection and interruption on
  the supported Windows target; source/storage disconnect scenarios beyond a
  deliberately failing reader; longer mixed HEIC/video/large-library endurance.
- Combined installer/regression validation before a release includes the HTTP
  classification fix. No new installer, version, provider, dependency or public
  workflow has been created by this worker slice.
- Files larger than 4 GiB remain unsupported. Larger-file support would need an
  explicitly designed/tested path; ZIP64/backups remain separate milestone 5 work.


## Windows cloud execution

[Run 41683468062](https://github.com/aataat12/immich-clone/actions/runs/41683468062), source `e7d7b7ff8474875dd73ad672a549af14c3e94f0f`, independent job `126082908579`: PASS on Windows Server 2022, Go 1.27.3 windows/amd64, CGO disabled, GOAMD64=v1. The actual boundary test passed in 62.44 seconds. Exact import physically stored 4,294,967,296 bytes (13.992 s); unchanged replay streamed the same bytes (18.085 s); changed replay streamed the full size and returned 409 (14.088 s); +1-byte body produced 4,294,967,297 bytes and returned 413 (5.241 s). Both full HTTP SHA256 readbacks and final integrity/receipt/counter checks passed. The fast 413 regression passed as well. These timings are disposable-runner evidence, not performance promises.

This was loopback HTTP plus filesystem execution, not packaged LocalService/browser selection, Windows 10 or a large-video decoding/backup test. The separate deletion job in the same workflow failed its permission-fixture read precondition; the overall workflow must not be called successful. Worker slice implementation and its Linux/Windows HTTP proof are complete, but milestone 4 and installed release acceptance remain open. Work paused at the user's request; no new installer was published.

## Installer integration

The oversized-request HTTP 413 fix is included in 0.1.34-prototype after combined installer run [41704978730](https://github.com/aataat12/immich-clone/actions/runs/41704978730) passed actual 0.1.33 upgrade, rollback, installed regressions and independent restored reinstall. The exact 4 GiB Windows HTTP proof above was reused, not repeated. Packaged/browser large-file selection, broader storage disconnects and mixed-library endurance remain open milestone 4 work.

## Actual selected-source loss and recovery: completed bounded slice

[Run 41715462566](https://github.com/aataat12/immich-clone/actions/runs/41715462566), source `60576704c67371c6dec1b6ca047178c47358cbd2`, job `126194160663`: PASS, first attempt, Windows Server 2022, Go 1.27.3, actual LocalService and Edge/Playwright 1.62.1. The test selected two real filesystem paths, then physically removed the second disposable source before copying. It used no route interception, synthetic failing reader or mocked file contents. The first copy remained durably accepted; the second had no accepted receipt or bytes, and the processing lease was released.

After reopening the page, the missing 8 MiB source was physically restored. The earlier accepted source was changed while preserving its byte length and timestamp. The browser manifest matched, but actual replay hashing refused the changed bytes, retained the original receipt/hash/counters, and displayed the saved changed-source explanation. Restoring that first source and reselecting both files then completed the same import: two new originals, zero duplicates, exact accepted bytes, no residual receipt errors, unchanged source bytes, matching original downloads and no stranded processing pause. The narrow page had no overflow or JavaScript errors, and offline CLI verification passed every catalogued original's SHA256.

The unreleased UI now explains selected-source unavailability as well as connection interruption, surfaces the saved per-file failure reason, keeps HTTP 413 understandable, clears stale messages when a run starts and rejects an incomplete success response rather than leaving the transfer promise pending. Local focused import/HTTP and bundled-script regressions passed. The new source-loss browser test is included in the next packaged-installer acceptance, but that new installer run has not happened yet.

This proves an actual selected file becoming unavailable; no source volume, network share or destination disk was physically detached. It does not prove mixed HEIC/video endurance, packaged 4 GiB browser selection, Windows 10, large-video decoding or large backup recovery. Those existing milestone 4/5 gates remain open. The already accepted 4 GiB and Hidden Media navigation runs were reused; their unrelated workflow jobs were skipped. No new dependency, limit increase or installer was published. Installed baseline remains 0.1.34; the import messages and tested Hidden Media navigation correction are queued for the next release. Security Group 2/3 work and every-frame Full Scan remain paused/deferred.

## Mixed PNG/HEIC/MP4/MOV batch crash and restore: completed bounded slice

[Run 41718160138](https://github.com/aataat12/immich-clone/actions/runs/41718160138), source `1a76aa003b7526dd6ba584cb43f7c4ae63be13bc`, job `126203544097`: PASS, first attempt, Windows Server 2022, Go 1.27.3 windows/amd64 with CGO disabled and GOAMD64=v1, actual LocalService, Edge/Playwright 1.62.1 and the unchanged accepted `ffmpeg-8.1.3-7eb77d88e2e9` provider. No new native dependency or application behavior change was needed for this test.

The real folder selection contained 240 files: 192 distinct byte hashes (48 PNG, 48 HEIC, 48 HEVC MP4 and 48 rotated MOV) plus 48 deliberate byte-identical duplicates. PNG colors were generated in the browser; the already accepted synthetic HEIC/video fixtures were varied using valid ISO-BMFF free boxes, not a varied natural-camera corpus. One MP4 contained a 1 MiB free box to make partial staging observable. This is a modest byte-volume synthetic batch, not a large-video or full-library endurance claim.

The actual browser connection was upload-throttled. After 82 originals were durably accepted and at least 64 KiB of the next upload existed in its real staging file, the harness forcibly terminated the actual service process. Browser interruption handling completed. Restart retained exactly those 82 receipts, kept the partial file unaccepted, removed interrupted staging and released copy-first state. Reselecting the same folder and resuming re-hashed accepted sources and completed all 240 receipts with exactly 192 new originals, 48 duplicates and the manifest byte total. All source files and downloaded managed originals matched SHA256; every recorded relative alias matched its source and all 192 previews reached ready under LocalService. The narrow import page had no overflow or JavaScript errors.

The service produced a real archive after completion. Offline CLI verification passed all originals, restore published to a separate folder, and the original state was made unavailable by renaming it offline. `TestMixedImportArchiveRestored` then verified the restored history, all 240 receipts, exact counts/accepted-byte totals, all 192 original hashes, all four MIME counts and source-alias presence. It regenerated every derived preview using the accepted provider in the CI account; that final check passed in 8.604 seconds. This is separate from LocalService decoding of the original batch. Local focused import regressions and test compilation passed before cloud execution.

Remaining milestone 4 work retains its earlier scope: packaged/browser actual large-file selection/interruption, real destination or share/volume disconnects, larger and longer mixed-library workloads, and target Windows acceptance. The 4 GiB HTTP, actual source-file loss and Hidden Media navigation proofs were reused without rerunning their unrelated jobs. This run does not establish Windows 10 compatibility, sampled AI behavior for large videos, or full-scale ZIP64 backup capacity. No new installer was published. Next is the combined installer candidate carrying the queued navigation and import-message changes, using actual 0.1.34 as its upgrade baseline; that previous executable already understands pending deletions and should refuse reimport until cleanup finishes. The separate unaware 0.1.33 writer/conflict proof remains recorded in the 0.1.34 acceptance rather than silently changing that earlier evidence. Security Group 2/3 work and Full Scan remain paused/deferred.


## 0.1.35 packaged recovery integration

The navigation/import-message update is published after [combined installer run 41722938850](https://github.com/aataat12/immich-clone/actions/runs/41722938850), source `734046903d88a0036b64243a4c6f2ee3a9173a75`, job `126220153707`, passed on its first attempt. The actual 0.1.34 upgrade, both rollback paths and independent restored reinstall passed. The selected-file disappearance, same-metadata changed replay and exact resume/count/hash test now passed against the actual installed executable under LocalService as well as its earlier focused harness. Installed Hidden Media return navigation passed too.

The separate 240-file mixed crash/restore and exact 4 GiB HTTP results above were reused; they were not rerun or relabelled as packaged 4 GiB selection evidence. Milestone 4 remains in progress with packaged large-file selection/interruption, real destination/share/volume disconnects, larger/longer workloads and target Windows checks still open. No file limit or native dependency changed. Published installer: 0.1.35-prototype, 104212262 bytes, SHA256 `2b6ba59fa6f31eb3f2edd2e73d8ad003143a2306d0a2317dbea23e7d9e65e604`. User validation is limited to the small UI/history checklist in [validation-0.1.35.md](validation-0.1.35.md).


## Worker destination recovery and 0.1.36 checkpoint

A dedicated worker completed the actual originals-directory outage/reconnect task: [run 41737710757](https://github.com/aataat12/immich-clone/actions/runs/41737710757), source `9d365f2a35269c0cbf8054ad659de6b309f725a5`, job `126271092301`, PASS first attempt on Windows Server 2022 LocalService/Edge. Review and scope are in [import-destination-acceptance.md](import-destination-acceptance.md). Pending publication and accepted-receipt replay were rejected while the real originals directory was absent; existing catalog/receipt/offline bytes survived. Reconnect before restart and browser resume produced three receipts, two originals and one duplicate with correct hashes/counters. No production change was needed. Test/evidence files were integrated via `d5e5a92eca354508cb77a8ed25a5d995dc0a4d55` after the release run had started; they were not executed as part of that combined run.

Published 0.1.36 passed the separate [combined installer run 41737634745](https://github.com/aataat12/immich-clone/actions/runs/41737634745), source `8d9107766d8d5967e63ad5da5bada5d6dfac42bd`, job `126270821113`: actual 0.1.35 upgrade, both rollback paths, installed batch Hidden Media/Show-Hide controls, source-file recovery, media/AI/reanalysis and independent restored reinstall. Installer size 104213538 bytes, SHA256 `33d044091b0c03861a4357ccc7ccbf75ae42340d87be72659455255383a5c79a`.

Milestone 4 remains open for actual packaged large-file selection/interruption, physical destination/share/volume disconnection and larger/longer workloads plus target Windows validation. The completed worker proof is a real directory outage with reconnect preceding restart, not those broader failure cases. No dependency or file-size limit changed. Group 2/3 security work and Full Scan remain paused/deferred.


## Installed 4 GiB browser gap closed; ZIP64 entry-count work advanced

[Run 41741097565](https://github.com/aataat12/immich-clone/actions/runs/41741097565), harness source `b030900d8726aa29d2bc5917d1ad3867fdb5cb5d`, job `126282504308`: PASS first attempt using the checksum-verified published 0.1.36 installer, actual LocalService and Edge on Windows Server 2022. Real disk-backed browser file-input selection rejected 4 GiB + 1 byte before any import session, accepted exact 4 GiB, survived forced termination after actual partial staging, and resumed with the earlier receipt retained. Full streamed original readback, source SHA256/size, counters, staging cleanup and installed offline verification passed. See [installed-import-boundary-acceptance.md](installed-import-boundary-acceptance.md) for exact steps and limits. This is new packaged/browser proof; the earlier Go HTTP boundary evidence retains its own scope.

The assigned worker also completed a bounded milestone 5 ZIP64 entry-count proof: 65,536 tiny originals, 65,540 ZIP entries and independent default-budget restoration with the original state offline. See [backup-zip64-entry-count-acceptance.md](backup-zip64-entry-count-acceptance.md). It is not a >4 GiB archive/payload-byte proof or a 65k UI-import run.

Milestone 4 still needs physical storage/share-volume outage behavior, larger/longer natural-media workloads and target Windows acceptance. Milestone 5 still has large byte-volume/throughput/interruption and practical full-library workflow gates. No production behavior, format, dependency or limit changed, so no new release was created for these checks. Installed baseline remains 0.1.36. No local action is required for this checkpoint.


## Large backup byte-volume checkpoint

Milestone 5 now also has an actual >4 GiB archive proof: [run 41743027697](https://github.com/aataat12/immich-clone/actions/runs/41743027697), source `483ae0b0854d1d39ad084023dcb5a1ed53845507`, job `126289008861`, PASS on Windows Server 2022. Production import wrote 4 GiB + 1 MiB across two originals. Mid-copy cooperative cancellation cleaned staging; retry created a 4,296,346,455-byte archive. Full archive verification, independent default-budget restore with the source state offline, and full source/restored hash checks against regenerated bytes passed. See [backup-large-volume-acceptance.md](backup-large-volume-acceptance.md), including the prior pre-job workflow validation failure. No production code, version or limits changed. Physical storage outages, natural-media endurance, installed large-backup workflow and target Windows acceptance remain separate gates; this does not close all of milestone 5.
