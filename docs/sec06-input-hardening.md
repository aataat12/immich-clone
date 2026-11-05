# SEC-06 input hardening candidate — November 5, 2026

Status: implemented locally, NOT executed or accepted. SEC-06 remains open.
Published prototype remains 0.1.39. No dependency/provider/model/schema change.

## Candidate changes

- Shared parent Go image decoder for preview and AI: 64 MiB encoded decode
  budget, overflow-safe 40-megapixel check, decoded/config bounds and format
  agreement, cancellation checks before/after decoding. Original upload/download
  limit remains 4 GiB; larger encoded images retain originals and report failed
  derived processing. This extends the existing 64 MiB GIF/BMP/TIFF preview bound
  to parent JPEG/PNG preview decoding; test/accept this behavior before release.
- EXIF structural failure discards partial fields; malformed metadata remains a
  warning and does not prevent otherwise valid media previews. Orientation copy
  checks cancellation between rows.
- Direct upload validates display basename length/UTF-8/control characters before
  reading any file bytes. Relative import paths reject invalid UTF-8. Names remain
  display data; originals use SHA256 storage names and downloads retain existing
  mime.FormatMediaType handling. Unicode and script-like labels remain supported.
- Names/tags reject controls before whitespace trimming. Organization/import JSON
  requests require valid UTF-8, bounded complete bodies, one JSON value and known
  fields before library calls.
- Catalog input validation runs before Open repairs/index rebuilding and on
  OpenReadOnly. Checks bounded leaf/key/tree shapes, UTF-8 and authoritative asset,
  face/label/tag/analysis/reanalysis/deletion/import records. Asset records allow
  32 MiB to preserve 4096 maximum-length source aliases even with JSON escaping;
  other leaves retain the existing 1 MiB organization-record bound. Existing
  structural bbolt checking and hard-limited restore processing remain separate.

## Prepared validation

The dedicated sec06-input workflow runs pinned Go 1.27.3 on Windows Server 2022:
module verification, canonical formatting, focused tests, full internal/CLI tests,
vet and build. It publishes no product installer. New tests cover actual truncated
JPEG/PNG/GIF/BMP/TIFF decoding, persisted preview failure, AI failure, subsequent
valid work and unchanged originals; dimension/encoded bombs; read/rotation
cancellation; malformed EXIF inside valid JPEG; rejected names before reads;
invalid JSON framing/encoding; catalog identity/field/shape/size checks; valid
Unicode, imports and maximum alias counts. A real 40MP 16-bit PNG/rotation probe
logs wall time and Go TotalAlloc. These are finite measurements, not peak RSS or
hard CPU/memory containment guarantees.

## Execution blocker and remaining acceptance

No local Go toolchain is available. Local diff whitespace and UTF-8 checks passed;
Go compilation, tests, formatting and resource measurements have NOT run.
The user explicitly authorized uploading this SEC-06 source/test payload to a
separate branch of the existing public project repository and running Windows
checks on November 5. Earlier automatic-review rejections were not bypassed.

Upload procedure: refresh the candidate snapshot, verify each uploaded blob
against its local Git hash, then move only the separate branch with an expected
head check. Inspect focused/full results, correct
failures, retain exact evidence and canonical formatting. Do not claim SEC-06
closure based only on this corpus: evaluate actual parent resource behavior,
remaining malformed native media/application integration, cancellation/usability
and original/restore parity. Existing patched NR-01 evidence is reused; SEC-05
native authority and SEC-08 frozen installer/Windows 10 acceptance remain separate.
No new charges, local setup, release or Windows 10 certification is implied.
