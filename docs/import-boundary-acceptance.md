# Milestone 4: exact 4 GiB HTTP import boundary

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
