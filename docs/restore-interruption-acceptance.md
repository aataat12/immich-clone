# Milestone 5: actual mid-extraction restore cancellation and retry

[Windows run 41743990678](https://github.com/aataat12/immich-clone/actions/runs/41743990678), source `e9666959d011f43c8734b1d2445fd05be5a2448a`, job `126292284233`: PASS, first execution. Windows Server 2022, Go 1.27.3 windows/amd64, CGO disabled, GOAMD64=v1. No production change, format change, dependency change, release or restore-limit increase was needed. Local compilation/opt-in skip and vet passed before Windows execution.

## Actual operations

- Production PutWithPath physically stored one 512 MiB synthetic original with a source alias. Production StagedBackup created its archive with catalog and a synthetic required secret. The payload uses the existing deterministic AES-CTR fixture generator, with a public constant key, streamed without a full-payload memory buffer.
- The archive was moved outside the source state, and that state was renamed offline before either restore attempt. No repair reads from the old state occurred during restoration.
- A watcher inspected only the fixture's restore staging directory. Cancellation required an actual original file with at least 16 MiB and less than 512 MiB extracted, rather than just staging appearance.
- Observed **17,825,792 / 536,870,912 original bytes** before cancelling the restore context. Production supervision returned context.Canceled, and cleanup left neither a published target nor private staging. This attempt took **386.702 ms**. The full archive SHA256 matched its pre-attempt value.
- A new RestoreWithLimits call used the same archive and target with unchanged default limits. It completed independently in **4.119 seconds**. This starts restoration again; it is not byte-resume.
- Exact restored original size and full SHA256 passed against independently regenerated fixture bytes and the original import ID. The offline original was also fully hashed. Restored catalog name/size/source alias and required secret matched. The archive hash still matched; only the final target remained, with no extra staging.
- The complete test, including fixture construction and cleanup, passed in **9.86 seconds**.

## Reproduction and bounds

`internal/library/restore_interruption_test.go`: TestActualRestoreCancellationDuringExtractionAndRetry. Requires `IMMICH_RESTORE_INTERRUPTION=1`, optional disposable `IMMICH_RESTORE_TEMP`, and at least 3 GiB free. Ordinary tests compile and skip it. `.github/workflows/restore-interruption.yml` runs the Windows proof, with 3-minute operation context, 4-minute Go timeout and 8-minute job timeout. Staging observation polls at 1 ms and requires actual partial extraction; it fails instead of claiming interruption if extraction is missed. Production restore budgets remain unchanged. Peak memory and total filesystem allocation were not measured.

## Evidence boundaries

This advances the earlier stage-appearance cancellation proof to actual mid-original extraction and an independently verified retry. Context cancellation causes the production supervisor to terminate and reap the active worker before cleanup. It is not forced termination of the restore controller/installer, power loss, physical storage removal, installed LocalService/browser acceptance, natural-media endurance, Windows 10 or completion of all milestone 5 work. The previous >4 GiB backup/archive proof remains separate; this test did not repeat that workload.

No security Group 2/3 investigation or mitigation was undertaken. The unexplained restore progress-read:32 issue remains deferred and unfixed; this passing run does not explain it. Every-frame Full Scan remains deferred. Application baseline stays 0.1.36; no user-PC action is needed.
