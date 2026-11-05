# SEC-06 input hardening — accepted candidate, November 5, 2026

SEC-06 is closed in the validated unpublished candidate for the current
loopback/authenticated local-import/catalog deployment. The original item was
implemented and exercised, including parent Go decoding; this is not a claim of
absence of parser defects or an OS sandbox. Published prototype remains 0.1.39.
SEC-03, SEC-05 and SEC-08 remain open. No dependency/provider/model/schema change.

## Implemented controls

- Preview and AI share overflow-safe 40-megapixel preflight and decoded/config
  size/format agreement checks. Existing encoded limits are preserved: JPEG/PNG
  previews and originals 4 GiB, GIF/BMP/TIFF previews and parent AI 64 MiB. A proposed
  uniform 64 MiB JPEG/PNG cutoff was removed before acceptance; a valid encoded
  JPEG above 64 MiB still previews, while its existing AI size refusal remains.
- TIFF uses the same pinned upstream parser through bounded context-checked random
  reads. Repeated compressed strips now consult cancellation rather than replaying
  an internal input cache. Existing production preview/AI deadlines are 45/75 s.
- TIFF padded tiles have a separate pre-decode budget of 40MP plus 1MP for ordinary
  edge padding. Image dimensions alone permit a skinny 1x40M image with 17x40M
  tiles to advertise 5.44 GB of decompression storage. The guarded regression proves
  refusal before pixel-payload reads, without allocating/decompressing that amount.
  The [pinned upstream implementation](https://raw.githubusercontent.com/golang/image/v0.47.0/tiff/reader.go)
  was reviewed for scalar, strip, tile and decompression behavior; this finding is
  resource amplification, not a demonstrated memory-corruption exploit.
- EXIF structural failure resets partially parsed fields. A corrupt EXIF pointer
  inside valid JPEG produces a metadata warning and a normal unrotated preview.
  Rotation checks cancellation per row and uses typed pixel access to avoid
  per-pixel heap allocation while retaining all eight transforms and color parity.
- Direct upload validates display basename length/UTF-8/controls before reading
  bytes; relative import paths reject invalid UTF-8. Names remain display data,
  storage paths remain SHA256 identifiers, and downloads use existing safe MIME
  disposition formatting. Unicode and script-like names/tags remain supported.
- Names/tags reject controls before trimming. Organization/import JSON requires
  valid UTF-8, bounded complete bodies, one value and known fields. Existing
  textContent/browser escaping, nosniff, attachment-only arbitrary originals,
  safe inline-media allowlist and script-hash CSP were reviewed and retained.
- Catalog validation precedes Open repairs/index rebuild and also runs for
  OpenReadOnly. It checks bounded record/key/tree shapes and authoritative assets,
  faces/labels/tags, analysis/reanalysis/video samples, deletion and import records.
  IDs/key bindings, field/status/size/encoding bounds and source paths are checked.
  Asset records allow 32 MiB for the existing 4096 maximum-length aliases including
  worst-case JSON escaping; other leaves retain the 1 MiB organization bound.
  Malformed Open refuses without changing catalog or original bytes. Structural
  bbolt checking remains inside the existing hard-limited untrusted restore worker.

## Executed evidence

All runs below used disposable Windows Server 2022, pinned Go 1.27.3, GOAMD64=v1
and CGO disabled. No end-user toolchain or local dependency setup was requested.

| Candidate / run | Result |
|---|---|
| Initial 3c03df2, [42073757825](https://github.com/aataat12/immich-clone/actions/runs/42073757825), job127408339355 | Focused/full internal/CLI tests, module verification, vet and build passed. Initial 40MP PNG16+rotation allocated 1120197168 bytes in 2.305 s. |
| 2c7b511, [42074997508](https://github.com/aataat12/immich-clone/actions/runs/42074997508), job127412730456 | Typed rotation, TIFF strip/tile/count/expansion/cancellation and large encoded JPEG parity passed; full tests and actual media matrix passed. |
| 5605eeb7e91e12129260ff2439312af579dbcdda, [42075603671](https://github.com/aataat12/immich-clone/actions/runs/42075603671), job127414841336 | Padding-amplification guard passed focused/full tests, vet/build and the complete configured media/restore matrix. |
| d9f4146a1e0138e80d0e21a50f7b83cad2691439, [42076364254](https://github.com/aataat12/immich-clone/actions/runs/42076364254), job127417509993 | Final catalog display/reanalysis bounds and all focused/full internal/CLI tests, module verification, vet/build passed. Native steps intentionally reused the preceding passing run: no native/media/rotation code changed. |

Final finite resource probe: actual 40MP PNG16 decode+rotation took 1.078 s and
allocated 480197136 bytes, versus 1120197168 initially. These are Go TotalAlloc
measurements, not process peak RSS, a speed guarantee or a hard memory boundary.
A 1 MiB compressed TIFF stream advertising one pixel allocated 49208 bytes;
1024 repeated strips stopped after eight payload reads in deterministic
cancellation testing. Oversized/overflowing dimensions, tile/strip counts,
truncated JPEG/PNG/GIF/BMP/TIFF, read/rotation cancellation, original hashes,
persisted failed previews and subsequent valid preview/AI work all passed.

Configured production-media checks passed 24 independently generated image
fixtures, three reduced public camera EXIF samples, all eight JPEG transforms,
DateTimeOriginal precedence, progressive/CMYK JPEG, PNG color/alpha/palette/gray,
BMP, animated GIF first frame, four compressed/multipage TIFF first pages and
12MP JPEG, with aliases/original hashes and independently restored regenerated
previews. Actual pinned FFmpeg/ffprobe rejected malformed HEIC/MP4/MOV containers,
then resumed valid video thumbnail/probe/frame work and recovered after
cancellation. Audio-first and nonzero-start timestamp fixtures passed. Existing
native process/memory/output/death controls passed in the full Windows suite;
existing patched NR-01 inference/application evidence remains opencv-nr01.md.

Initial actual installer/trust regression42073757491 also passed; catalog probes
42073757656/42074997951/42075603808 passed. The final catalog baseline was checked
separately before closeout. The runner's canonical Go formatting was copied back;
all 14 formatted-file hashes match its diff receipt. No behavioral edits follow
final validation. User-authorized worker review/tests and independent TIFF guard
review are complete. Uploaded blobs are verified against local Git object hashes.
Source/test/doc uploads exclude private media, support logs and personal machine/
network details; normal existing GitHub account attribution remains Git metadata.

## Limits retained in the original worklist

Parent decoders remain in the Go service with finite byte/pixel/block limits and
cooperative cancellation; these controls do not claim an OS-enforced parent
memory/CPU quota. Native children retain their existing finite JobObject/output/
time controls, but filesystem/network authority remains original SEC-05, not
cleared by hostile-media tests. Unknown future parser defects require servicing.
SEC-03 concurrency and SEC-08 frozen combined installer/full-provider/rollback/
restore/browser and final Windows 10 gates remain unchanged. Server results do
not certify Windows 10. No release, main merge, paid resource or Group 3 work.
