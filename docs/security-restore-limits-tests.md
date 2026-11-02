# SEC-04 restore limits: independent acceptance

This records the independent adversarial tests for original security item 4. It does not close installer trust, junction races, native AI worker isolation, or Windows 10 compatibility.

## Acceptance matrix

| Requirement | Executed test |
| --- | --- |
| Aggregate extracted bytes, entry count, manifest/catalog/ZIP metadata budgets | Actual `RestoreWithLimits` refusal, clean destination, untouched outside sentinel; valid independent backup succeeds |
| Compression expansion and arithmetic overflow | Valid 16 MiB high-compression ZIP refused below expanded size and restored above it; ZIP64 impossible declaration; exact-bound and overflow addition controls |
| Actual copied bytes | `io.Copy` through production writer reaches exact bound and cannot write further bytes |
| Hostile catalog work | Manifest-valid invalid bbolt data and self-referencing root branch sent through real restore; hard subprocess deadline contains processing and staging is removed |
| Noncooperative execution | Production supervisor starts helper body, kills/reaps on timeout and explicit cancellation; bounded output flooding is refused |
| Hard Windows memory | Same production Job runner allows 32 MiB helper and refuses 600 MiB helper within 512 MiB Job; test skips off Windows |
| Runtime catalog growth | Valid progress and small catalog pass, later growth triggers actual cancellation within one second |
| Cleanup after work starts | Staging appearance is observed, restore cancelled, no target/staging survives; outside originals/archive remain available |
| Preservation and old-format recovery | Archive source made unavailable; restore verifies originals, corrected name, name history/undo, dismissal, tags, ready AI results, synthetic access code |
| Actual product flags | `Test-RestoreLimits.ps1` invokes built executable: default and explicit successful restore/verify, seven low-budget refusals, metadata inspection, source ZIP SHA unchanged |

Fixtures are generated from synthetic data by the normal backup writer, retain the established manifest/schema, and use no user originals or credentials. Attacks use small fixtures and bounded deadlines, without giant allocations or disk writes. The sole deliberately excessive allocation is the Windows child memory test, which must be contained by its Job. Logs and fixture files stay in a disposable private test directory; only fixed result messages should be retained in CI.

## Commands and evidence

Linux execution in the development environment:

```
go test ./internal/library -run 'TestRestore(Limits|Budget|Supervisor|Windows)' -count=1
go test -race ./internal/library -run 'TestRestore(Limits|Budget|Supervisor|Windows)' -count=1
```

The focused non-race run passed (2.855 s). The final expanded focused race run passed (18.354 s). Linux does not prove Windows Job memory enforcement.

Actual Windows release acceptance must execute both the Go suite (including `TestRestoreWindowsHardMemoryBound`, without a skip) and:

```
./scripts/Test-RestoreLimits.ps1 -Binary <built-security-app.exe>
```

Do not close SEC-04 from cross-compilation, the helper starting, or a successful package import. Record actual Windows CI run/results in authoritative release evidence. Windows Server evidence does not establish Windows 10 compatibility.

## Exact enforcement scope

ZIP extraction uses a strict aggregate writer. Catalog processing is entirely in the bounded child, including opening, integrity checking, rebuilding derived data, listing, verification, retries, and closing. The Windows Job supplies the hard memory bound; the parent deadline and process kill contain noncooperative CPU work. Catalog growth is checked every 20 ms and before publication, so it has monitoring latency and is not an atomic disk quota. Dynamic available-space defaults also cannot reserve space against unrelated concurrent applications. These are residual limits, not claims of perfect disk isolation.

## Actual Windows acceptance — 2026-11-02

Independent reviewer fetched and inspected the actual decoded job logs for [run 41547447887, job 125626387613](https://github.com/aataat12/immich-clone/actions/runs/41547447887/job/125626387613), source `1f1cdb912b223634dec0350e58e9ae86b11893b0`. The runner was Windows Server 2022, build 10.0.20348. This is execution evidence, not cross-compilation evidence.

| Acceptance | Actual result |
| --- | --- |
| Product CLI default and explicit old-format restore, seven low-budget refusals, staging cleanup, original/metadata preservation | PASS; fixed CLI success message present |
| Aggregate, entry, manifest, catalog, metadata and overflow controls; independent recovery and high-expansion positive | PASS, 0.76 s |
| Invalid and self-cyclic bbolt catalogs, cleanup | PASS, 1.70 s; cyclic root contained in 1.53 s |
| Hard Windows memory control | PASS, not skipped: 32 MiB control 0.04 s; excessive 600 MiB allocation 0.03 s under 512 MiB Job |
| Supervisor success, noncooperative timeout, output flood | PASS, 1.06 s |
| Explicit cancellation and child reaping | PASS, 0.06 s |
| Live catalog-growth cancellation | PASS, 0.02 s |
| Cancellation after staging appeared, cleanup and unchanged input ZIP | PASS, 0.17 s |
| Copied-byte guard and nonwrapping budget arithmetic | PASS |
| Full library and web suites | PASS, 58.140 s and 24.745 s respectively |

**Independent disposition: original SEC-04 is ready to close.** Aggregate extraction and catalog-processing work have implemented controls and actual Windows adversarial/positive acceptance. The 20 ms catalog-growth monitor remains a bounded-latency check, not an atomic disk quota; unrelated applications can consume available disk space. Windows 10 compatibility is unchanged and is not established by this Windows Server run. Closure does not extend to the separately tracked filesystem race or native AI worker isolation items.
