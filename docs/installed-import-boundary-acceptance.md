# Milestone 4: installed browser 4 GiB import boundary

Completed bounded acceptance slice; no application, dependency or file-size limit changed.

[Run 41741097565](https://github.com/aataat12/immich-clone/actions/runs/41741097565), harness source `b030900d8726aa29d2bc5917d1ad3867fdb5cb5d`, job `126282504308`: PASS, first attempt, Windows Server 2022, actual installed LocalService and Edge/Playwright 1.62.1. The test installed the actual published 0.1.36-prototype package from accepted run 41737634745 after verifying installer SHA256 `33d044091b0c03861a4357ccc7ccbf75ae42340d87be72659455255383a5c79a`. The service account and executable version were asserted; this was not a fresh relabelled application build.

## Real operations

- A guarded disposable drive had at least 12 GiB available before large I/O. A 64 KiB generator physically wrote a 4,294,967,297-byte source file; no multi-GiB JavaScript buffer or sparse-file shortcut was used. The actual browser file-input rejected that over-limit file and created no import session.
- The fixture was truncated by one byte to exactly 4,294,967,296 bytes and reselected alongside a tiny earlier file. Browser File sizes and enabled Copy control were asserted. File selection used real filesystem paths, not injected file buffers; the native Windows Open dialog itself was not automated.
- After the earlier file was durably accepted, a throttled upload of the large file wrote at least 128 KiB into the application's actual staging file. The real installed service was forcibly terminated. Browser interruption handling completed without reporting an accepted large-file receipt.
- Restart retained the earlier receipt and its byte total, left the large file unaccepted, removed interrupted staging and released copy-first state. Browser reselect/resume completed the same import: two new originals, zero duplicates, exactly 4 GiB plus the earlier source's byte count, and no saved receipt errors or stranded processing lease.
- The managed large original's real file size and receipt hash matched the generated source. An authenticated full HTTP download was streamed through a SHA256 digest without retaining its body in a multi-GiB browser/API buffer. A separate full source-file read matched the same hash and exact byte count; the earlier source bytes were unchanged. Offline verification using the installed executable passed all catalogued originals.

The source/install/state files were created only beneath the harness's own randomly named CI directory and were cleaned after the installed test. The harness refuses an existing installation and checks ownership before fallback service cleanup. No real user files or access code were committed or uploaded.

## Reproduction and scope

The dedicated `.github/workflows/installed-import-boundary.yml` retrieves the accepted installer, provisions pinned Playwright and executes `scripts/Test-InstalledImportBoundary.ps1` with the two phases of `tests/installed-import-boundary.cjs`. Run only on disposable cloud CI with the free-space guard satisfied. No local test or dependency installation is requested from the user.

This closes the Windows Server packaged/browser file-input selection, real transfer interruption and resume gap for the advertised 4 GiB storage boundary. The earlier exact-limit HTTP replay/changed-content/+1-byte body refusal proof is retained separately and was not rerun here. The new +1-byte check was client selection rejection, not another multi-GiB over-limit HTTP transfer.

The payload is deterministic unsupported binary content, not a 4 GiB video decoding/AI test. This does not establish Windows 10, physical volume/share detach, cloud-only file placeholders, longer natural mixed-media endurance or >4 GiB ZIP64 backup/restore. The file limit remains 4 GiB. No new installer was needed; user baseline remains 0.1.36. Security Group 2/3 work, the known restore progress-read:32 issue and every-frame Full Scan were not investigated or changed.
