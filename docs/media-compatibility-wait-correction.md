# Milestone 6 installed-matrix wait correction

The first combined0.1.38 run [41769332340](https://github.com/aataat12/immich-clone/actions/runs/41769332340), source `79c869b3365521ee9be11683c92fcc08914a303d`, job `126375076678`, stopped at `Installed preview not ready` in the newly added24-image installed matrix. Publication was blocked. The earlier direct Store matrix had passed; the failure alone did not establish an unsupported codec or decoder defect.

## Cause

The new harness used an async fetch predicate inside Playwright waitForFunction and discarded the resulting handle. In the exact pinned Playwright1.62.1 [frames.ts source](https://github.com/microsoft/playwright/blob/v1.62.1/packages/playwright-core/src/server/frames.ts), the injected polling loop calls the predicate synchronously and tests the returned value's truthiness. An async predicate returns a truthy Promise immediately; resolving it to false fulfils the wait rather than arranging another poll. The harness therefore continued before actual preview completion. This explains why the first installed matrix failed within3seconds despite the service processing one queued preview per tick.

## Narrow correction

Only this new matrix's wait was changed. A bounded90-second Node polling loop requests fresh actual catalogs, rejects an actual failed public fixture immediately with its fixed fixture name/status/MIME/error, and finishes only when all24 expected IDs are present and Ready. The same verified Ready snapshot supplies subsequent assertions. Temporary HTTP429 responses stay within the existing overall bound. No arbitrary sleep declares success and no failed preview is retried or repaired.

Geometry, metadata, independent expected pixels, original-hash checks and all24 inputs remain unchanged. The `--restored` mode still skips every upload and requires all IDs to exist already; it uses the corrected readiness poll before checking regenerated previews. No production code, decoder, format, dependency, threshold, resource limit or security mitigation was changed. Other existing harness waits were not audited or modified as part of this bounded task.

## Actual control and corrected proof

[Windows run41770783033](https://github.com/aataat12/immich-clone/actions/runs/41770783033), source `fd52bbe76068b7aac6197252be725ca4ed84aa03`, job `126379721501`: PASS first focused execution on Windows Server2022 using the exact failed-release production source, existing pinned media provider and actual headless Edge/Playwright1.62.1.

- The focused-only old-wait control required actual queued previews. It then demonstrated the old async wait returned **false in46ms with22 actual previews queued**; a fresh catalog still showed queued work after that return.
- The corrected explicit catalog loop waited **21,124ms** for all24 actual Ready previews.
- All24 production serve/upload previews then passed unchanged source dimensions, camera EXIF/date/orientation fields, independent pixel samples and full original-download SHA256 checks. The same three public reduced camera JPEGs, eight orientation fixtures, progressive/CMYK JPEG, PNG modes, BMP, animated GIF first frame, four compressed/multipage TIFF first pages and12MP JPEG were used.
- No actual failed preview/unsupported format was observed. The application was not changed to force the test to pass.

The focused replay is standalone production serve execution with the pinned media provider, not a new installed LocalService/installer acceptance claim. Combined acceptance must still pass the actual installed create/check and independent restored `--restored` modes before release. Final Windows10 clean-machine/reboot/endurance acceptance remains separate.

Files: corrected `tests/media-compatibility-installed.cjs`; disposable `tests/media-compatibility-serve.cjs`; `.github/workflows/media-compatibility-serve.yml`. The control runs only with IMMICH_COMPAT_ASYNC_CONTROL=1, set by the focused replay wrapper; the installed release harness uses only the corrected poll. The wrapper refuses an occupied app port and cleans only its own generated state. No user library, image, private diagnostic or embedding was involved.
