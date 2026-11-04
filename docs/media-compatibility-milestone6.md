# Milestone 6 — supported-media compatibility and resource acceptance

Functional acceptance for the existing supported-format scope, using the pinned production decoders and actual image/video operations. No new format, provider, model, dependency, storage schema or resource limit is introduced. RAW/HDR expansion, advanced TIFF metadata/color management, video transcoding, mobile and every-frame Full Scan remain deferred/excluded. Final Windows 10 clean-machine/reboot/endurance acceptance is separate.

## Current scope and acceptance matrix

| Surface | Accepted operation and limits | Evidence |
|---|---|---|
| JPEG camera metadata | Bounded APP1/TIFF subset: orientation 1–8, camera make/model, DateTimeOriginal priority over fallback date; wall-clock retained without invented timezone. No maker-note/GPS/advanced metadata extraction promise. | New 24-image matrix, with three reduced real-camera EXIF samples and independent Pillow expectations |
| JPEG pixels | Baseline, progressive, CMYK and synthetic 4032×3024 (12MP) preview, correct oriented source geometry and expected quadrant pixels | New matrix |
| PNG | RGB, alpha, palette and grayscale preview; alpha compared through normalized RGBA pixels | New matrix |
| BMP | 24-bit preview | New matrix; broader variants not claimed |
| GIF | First animation frame only for thumbnail/analysis; original download remains intact | New independently encoded animated fixture with different later frame |
| TIFF | First page only, raw/LZW/Adobe-deflate/PackBits compression; later page has different pixels, so selecting the wrong page fails | New matrix; all TIFF variants/orientation/metadata and multipage navigation are not claimed |
| Original preservation/recovery | Exact imported ID/SHA256 and source alias, actual production preview, independent default-budget ZIP restore with source renamed offline, regeneration and repeated checks | New matrix on all 24 images |
| Video metadata edge cases | Actual MP4 with audio stream at index0/video at index1; MOV with source start5 seconds; correct selected stream, dimensions, source-relative sample0, actual decoded red pixels and original preservation | New pinned-production provider test |
| Existing HEIC/video coverage | Single/grid12MP 8-bit HEIC; H264/HEVC MP4/MOV, rotation, VFR/short clips, actual sampled timestamps/content and model inference | Reused exact accepted 6A proof; details below |
| Resource bounds | Existing single native process,512MiB Windows Job memory, finite output and deadlines, thread limits; images checked against40MP and existing encoded-byte policies | Reused actual control/limit operations and current tests; no new security mitigation or broader isolation claim |

This is a finite declared supported-format matrix, not a claim to decode every camera, codec, TIFF photometric/compression combination, alpha auxiliary/burst HEIC, HDR profile or RAW variant. Unknown/unsupported previews remain explicit failures while originals are retained/downloadable.

## New executed evidence

[Windows run41767014589](https://github.com/aataat12/immich-clone/actions/runs/41767014589), source `f283b961c83ab382906287742654eaea45a5b54c`, job `126367619112`: PASS, first execution on Windows Server2022, Go1.27.3 windows/amd64, CGO disabled, GOAMD64=v1.

- Independent producer generated24 expected image records; three are reduced camera JPEG samples retaining real camera EXIF. It supplied expected displayed dimensions, supported metadata fields and four displayed-orientation pixel samples per image. Production import/preview/recovery did not generate its own expected answers.
- Production Store.PutWithPath imported every actual file and confirmed expected SHA256 identities without duplicates. ProcessOne generated previews; exact orientation/dimensions, metadata/date priority, pixel tolerances, source aliases and immutable original hashes passed.
- Production backup was created outside the source state; the source was closed and renamed offline. Default restoration created a separate library. Regenerated previews and original/catalog/alias/metadata checks passed for all24 items. Test duration2.38seconds.
- The exact pinned provider read actual audio-first MP4 metadata and selected video stream1, source start0; actual MOV source start5,000,000us was normalized to a decoded source-relative sample0. Thumbnail and full analysis frame pixels were red as independently specified, with160×120 source geometry and unchanged original bytes. Durations136ms and134ms; combined0.27seconds.
- Existing metadata routing/40MP guards and finite analysis-output allowance tests passed in the same Windows run. Local matrix execution and vet passed before cloud execution. Peak memory for the new fixture matrix was not measured; timing is a bounded CI observation, not a universal throughput guarantee.

Harnesses: `experiments/media-compatibility/prepare.py`, `prepare-video.ps1`, `internal/library/media_compatibility_matrix_test.go`, `internal/media/compatibility_matrix_windows_test.go`, `scripts/Test-MediaCompatibility.ps1`, `.github/workflows/media-compatibility.yml`. All external inputs/encoder/provider identities are pinned and checked. Fixture encoders are CI-only and are not distributed with the application. Temporary fixture/provider files are removed after the focused run.

## Reused execution evidence

The unchanged media worker/provider has actual Windows LocalService/restart checks for H264/HEVC, rotated MOV, 8-bit single/grid HEIC reconstruction with independent pixels, disabled network protocols, finite output and timeout termination, preassigned512MiB Job memory, one-process limit, crash and parent-death recovery. See `experiments/media-worker/README.md`, `docs/dependency-evidence.md`, and [accepted6A run41559260341](https://github.com/aataat12/immich-clone/actions/runs/41559260341), source `e95958bc9f24a1e56cdd5e47cce2248f80440d2a`. That6A run additionally proved face-bearing single/grid12MP HEIC and H264/HEVC/rotated/VFR/short timestamped inference with unchanged originals. The delivered installer path most recently passed in [0.1.37 combined run41752068473](https://github.com/aataat12/immich-clone/actions/runs/41752068473). Previously passing dependency operations were not rerun just to collect more green results.

Native resource controls are limits on time/memory/process count/output, not a filesystem/token sandbox or a CPU utilization guarantee. Pure Go raster decoding remains in the main service with existing dimension/encoded-byte checks; this milestone does not claim a hard per-image process memory sandbox. Security Group2/3 investigations and mitigation are still paused.

## Camera sample provenance and limitations

Source: [ianare/exif-samples](https://github.com/ianare/exif-samples/tree/5332a9cf0220e9c6c93f88a187daa90808051e10), fixed commit `5332a9cf0220e9c6c93f88a187daa90808051e10`. Its `jpg/README` identifies these as reduced Wikimedia images preserving camera EXIF; these are not a full-resolution camera-image corpus. Its repository README states contributed images use Attribution-ShareAlike4.0; original image authors/licenses are available via the mapped Wikimedia description pages. Evaluation downloads originals only; no camera images are committed, packaged or uploaded as CI artifacts. No new commercial image redistribution rights are claimed.

| Fixture | Original Wikimedia title credited by repository | SHA256 |
|---|---|---|
| Canon_40D.jpg | Iguana_iguana_male_head.jpg | `6bfdabd4fc33d112283c147acccc574e770bbe6fbdbc3d4da968ba7b606ecc2f` |
| Nikon_D70.jpg | Anolis_carolinensis_brown.jpg | `8e2a627b96ca71c20129161f46bda3d338407da99bd11b1055adb27af27d7ef5` |
| Pentax_K10D.jpg | Mrs._Herbert_Stevens_May_2008.jpg | `146601c9d406410abdaa832508ee4ccddbc7ad54530e81d57962c1b7728e2e6d` |

Other new images are unrestricted synthetic quadrant fixtures. Pillow12.0.0 and pillow-heif1.8.1 are the already hash-pinned test-fixture tools. The new test encoder is the same previously executed hash-pinned FFmpeg8.1.3 fixture encoder, not the application decoder. Metadata expected values come from Pillow and its independent EXIF reader, not from the production reader.

## Combined delivery handoff

`tests/media-compatibility-installed.cjs` performs the new24-image pixel/metadata/original matrix against an explicitly opted-in disposable installed LocalService library. Root can generate the same fixtures after the existing pinned Python fixture tooling is available, set IMMICH_COMPAT_FIXTURES, IMMICH_TEST_STATE and IMMICH_COMPAT_INSTALLED=1, then run it with the existing pinned Edge/Playwright harness. This installed pass is not claimed by the focused library/provider run above; record the combined release result before marking delivered. The harness enlarges only disposable CI state, never a user library. Its `--restored` mode skips all uploads, requires all24 expected IDs to exist already in the restored catalog, waits for regenerated previews, and performs the same geometry/metadata/pixel/original-hash checks. It checks that the page issued no mutation requests after authentication; no reimport or repair operation is allowed in that mode. Root can run this read-only mode after independent restored reinstall to retain the full matrix through the combined recovery gate.

No new installer or milestone completion version is assigned by this worker. Milestone6 functional scope has complete focused Windows acceptance; the root owns installed combined integration and final delivery status. Final target Windows10 clean-machine/reboot testing remains an explicit separate gate, not silently replaced by Server evidence.


## Combined installed delivery — 0.1.38

[Run41771361544](https://github.com/aataat12/immich-clone/actions/runs/41771361544), source86807bff81930a5394c60d695185bbdb7008d81c, job126381550300: PASS. Actual LocalService accepted the24-image matrix with exact expected geometry, metadata, pixels and original hashes. The full library was backed up and independently restored/reinstalled with the old state offline. The restored mode found all24 original IDs already present and verified regenerated previews and original downloads without any repair upload. This closes the combined functional delivery gate for the declared supported-format matrix; published release is0.1.38. See [release acceptance](release-0.1.38-acceptance.md) and [wait correction](media-compatibility-wait-correction.md) for the first failed harness run and deterministic correction.

No new decoder/provider/format or policy limit was needed. Existing resource controls retain their exact finite scope. This delivery does not certify Windows10, advanced RAW/HDR/TIFF variants, real-world recognition accuracy, process isolation for Go decoders, security Group2/3, physical-storage outages or full-library endurance.
