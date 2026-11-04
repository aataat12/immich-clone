# Disposable worker authority canary

Run from the repository root on a disposable elevated Windows runner:

```powershell
./scripts/Test-WorkerAccessCanary.ps1
```

The script builds `./experiments/worker-access-canary` from the root module with
CGO disabled. No new dependency or nested module is added. It creates a uniquely
named LocalService service, synthetic fixture directories and a self-contained
probe executable. A `finally` block stops/deletes that service and removes all
fixtures. No production service/data, user media or supplied credentials are used.

The LocalService parent launches this same executable first with ordinary
`exec.CommandContext`, then through the exact `immichclone/internal/media.Run`
launcher used by FFmpeg and native AI. Neither launcher is copied into the probe.
Each child reports only fixed boolean fields. The parent atomically publishes a
boolean-only result after both runs; the script checks the exact field set/types,
checks the real modified canary bytes independently and prints fixed labels.

Expected acceptance is intentionally **not isolation**:

- Both children confirm LocalService identity without printing token inventories.
- Both read synthetic secret/catalog canaries and overwrite a synthetic original
  in a directory granting LocalService Modify, like current application state.
- Both fail read/write attempts in a separate SYSTEM/Administrators-only directory.
- The ordinary child can allocate 600 MiB commitment and spawn a short-lived child.
  Under `media.Run`, both operations must fail; queried peak job commitment must be
  nonzero and within the configured bound.

Thus a green result proves that current Job Object resource limits coexist with
same-token file authority. It does not certify a sandbox, malicious-media safety,
network isolation, clean Windows 10, or a particular vulnerable parser. The
catalog is only a synthetic byte canary, not an active bbolt database; actual
catalog locks may affect concurrent writes and are not tested here.

Each control/job invocation has a ten-second context; service work has an outer
25-second budget and script report wait is 30 seconds. Output remains under the
production stdout limit. Go executable command arguments necessarily contain the
disposable fixture location, but reports/log labels never emit those paths or
canary values. No report/artifact upload is required.
