# Milestone 6A — HEIC and video AI

User requested this milestone on 2026-11-03. Implementation candidate in progress; not released.
Keep original milestones 1..8 unchanged; 6A adds explicit acceptance rather than
silently widening still-image AI. Group 1 mitigation proceeds alongside this milestone; Group 2 mitigation is paused per security-priority-groups.md.
No new model/runtime, mobile feature, GPU, cloud inference or local SDK required.

## HEIC still-image analysis

Reuse the existing hash-pinned HEIC media decoder, including grid reconstruction,
then feed a bounded oriented full-image analysis frame to the patched native
face/object provider. Do not analyze the small thumbnail as though it were the
original. Preserve immutable HEIC bytes and existing user names/tags/corrections;
apply current review-only suggestions. Unsupported HDR/high-bit-depth, bursts or
auxiliary-image variants must show a specific unsupported status, not success.

Acceptance: actual supported single/grid/12MP HEIC decoding followed by face/object
inference under LocalService; public/synthetic face-bearing HEIC with expected
results (an empty synthetic color grid is insufficient); orientation/geometry,
identity of original SHA256, missing-provider/retry, corrupt/oversized/timeout,
restart and independent backup/restore. Compare source-image inference to HEIC
inference with tolerances established before running the test. Windows Server
and applicable target Windows 10 results must be stated separately.

## Video sampled-frame analysis

First delivery analyzes a finite declared set of timestamped frames per clip.
It must label results as sampled coverage, never claim every frame was analyzed.
Proposed initial policy for feasibility: up to 12 evenly spaced timestamps, with
an explicit tested short-clip policy and configurable finite frame/work budget.
Validate real seek/timestamp behavior before freezing the product policy. Full
frame extraction/analysis, audio transcription, tracking and automatic identity
merging are outside this first milestone.

Acceptance: retain source-relative timestamps, sample dimensions/orientation,
provider/version and coverage policy; expose per-clip progress, cancel/pause,
restart/retry without duplicate observations. Keep frame-level results separate
from existing still-image face IDs; suggestions require review, repeated faces
must not silently create conflicting person names. Version persisted frame
observations; export and independently restore them alongside the original,
manual corrections and completed work. Frames/crops remain reproducible derived
files, not managed originals. Existing ready still-photo results are not rerun.

Prove actual H264/HEVC MP4/MOV processing and real face/object inference on frames
with known content at known timestamps, including rotated video, short/VFR clips,
no-video streams, malformed/unsupported inputs and missing decoder/model. Bound
clip probing/seek, frame count/size, scratch space, memory, execution time and
queue concurrency; maintain service responsiveness and import priority. Process
one sample at a time with prompt derived-file cleanup. Test cancel/crash/restart,
storage failure, corrections, browser filtering and independent recovery.

## Dependency and delivery gates

1. Build and execute the NR-01 OpenCV backports, API regressions, native model
   parity and LocalService recovery before selecting a new immutable provider.
2. Retain Group 1 controls and test new routes. Group 2 mitigation is deferred,
   not claimed fixed; normal data-preservation/recovery acceptance still applies.
3. Execute a small HEIC/video analysis feasibility slice with existing providers;
   record exact sample/codec/model identities and results before substantial
   schema/browser/job implementation. Add a dependency only if the existing
   decoder demonstrably cannot supply the required bounded frames.
4. Ship tested vertical slices with migration/rollback and actual installed
   provider execution. No user-library enlargement solely for testing, no new
   paid infrastructure or prerequisite installation. Build Notes must show
   Planned until acceptance, and Released In only for the actual delivery.

## Execution checkpoint

Dependency proof passed on Windows Server 2022 under LocalService and after
restart in [41559260341](https://github.com/aataat12/immich-clone/actions/runs/41559260341),
source e95958bc9f24a1e56cdd5e47cce2248f80440d2a, job125665621103.
Single/grid 12MP HEIC face references passed geometry and cosine tolerances
(minimum cosine0.9966728075891548). H264 known-position/content, HEVC, rotated,
VFR and short clips executed with actual inference. Unsupported/no-video inputs,
cancellation and unchanged original checks passed. Analysis edge is960px;
video uses up to12 evenly spaced targets, storing actual decoded timestamps and
skipping duplicate timestamps. These are sampled observations, not tracking.

Candidate backend/UI now persist per-clip pause/cancel/resume and individual
frame observations, names/tags and reproducible crops/context. Local full Go
suite/vet and focused race checks passed. Actual application acceptance passed
in the focused run below.
No installer/version/main update or Windows10 certification is claimed.

## Application checkpoint — November 2, 2026 (America/Chicago)

[41564920558](https://github.com/aataat12/immich-clone/actions/runs/41564920558),
source9c15c56dba52481f98d8b255e98303f6ce29af7b, job125684028344, passed on
Windows Server2022 with the actual app as LocalService and Edge:

- Both public face-bearing HEICs analyzed through the full analysis decoder.
- Explicit sampled-video start, persisted pause across service restart, cancel,
  resume to12 targets; unique actual timestamps, face observations and cup tags.
- Name/custom-tag edits, saved sampled-frame context, desktop/narrow overflow,
  no page errors and original-download SHA256 checks.
- Application-consistent archive restored independently; old state moved offline;
  saved coverage and video name retained, all face crops rebuilt, saved sampled
  context regenerated with the media provider and no AI model/provider configured.
- Windows mock regressions cover duplicate timestamps, in-flight pause, cancellation,
  resume/restart, independent restore and compatible still/video suggestions.
  Same model ID permits suggestions, never automatic naming; sampling policy is
  separately versioned. Existing ready still observations are not rerun.

Earlier application attempts41561426161/41561870989 checked face results while
still queued; explicit persisted-status polling corrected the test.
41562254568 then passed actual inference/frame viewing before its test-only
relative download URL error. Both were corrected without lowering assertions.
The source push also started broad workflows; scale/media completed successfully,
and application41561426175 was canceled. Temporary cancellation workflow removed.
Only focused workflow ran for the final source.

Remaining before full6A delivery: combined existing-feature/installed upgrade
and rollback acceptance, and applicable target Windows10 validation.
Feature-specific provider-loss/retry and storage/interruption coverage passed
in the recovery checkpoint below. Existing broad provider
and malformed-codec evidence is reused; exhaustive paused Group2 security review
is not reopened. No user-library enlargement or new dependency installation is
needed for this checkpoint. Build Notes stays Planned until actual delivery.

## Deferred proposal — post-import Full Scan

User requested that this be tabled beyond6A. Future proposal: user selects videos
or chooses all videos without a completed Full Scan, chooses detection checkboxes
(people, places, particular objects) beforehand, then runs every-frame analysis
with timestamps and persistent progress. Define place semantics and supported
object detection/model capability before implementation. This proposal does not
change current6A sampled coverage and is not being implemented now.

## Recovery checkpoint — November 2, 2026 (America/Chicago)

[41567782811](https://github.com/aataat12/immich-clone/actions/runs/41567782811),
source718eab96aa5ef281524de492498757d8b74469b3, job125693367010, passed on
Windows Server2022/LocalService/Edge with unchanged selected native/media binaries.

- Partial video observations persisted through pause and service restart; one face
  was manually named and its clip tagged before failures were introduced.
- Missing decoder at startup left saved organization accessible and disabled the
  unavailable video resume control. Runtime decoder loss visibly failed a new
  HEIC job; restoring the decoder allowed retry to ready with a real detected face.
- Denying LocalService writes to the disposable derived-crop directory failed the
  next video sample without advancing/replacing the prior checkpoint. This is
  a concrete write-permission failure, not a new claim of full-disk coverage.
- Removing the actual SFace model failed the next sample with the saved progress,
  face identity, name and tag unchanged. The failure survived service restart.
- After model restoration/resume, the test observed an actual native child and
  forcibly terminated the service process without graceful shutdown. The child
  was reaped; the restarted service completed the queued clip with unique sample
  timestamps and all prior sample records/face IDs/manual corrections retained.
- Original SHA256 downloads passed for the recovery MOV/HEIC. A final independent
  archive restore with the old state offline rebuilt all crops and saved frame
  context without an AI provider, now across2 clips and3 HEIC stills.
- The original single/grid HEIC, sampled-video UI/restart and desktop/narrow suite
  and focused Windows regressions ran successfully in the same workflow.

The first recovery run41567306332 used a90-degree rotated fixture with no detected
face and failed the required partial-face positive control. Recovery now uses a
remux of the proven upright face/cup footage, retaining that assertion. Rotation
probe/inference evidence remains distinct; no claim of rotation-invariant face
recognition is made. No product protection or result assertion was weakened.

These are functional recovery tests, not resumed Group2/3 mitigation. Full Scan
stays deferred. No installer/version/main change; target Windows10 and combined
release acceptance remain unverified for6A. The installer harness still starts
from0.1.27; the next normal release validation must use the actual installed0.1.29
prototype as its upgrade baseline and verify the combined candidate and rollback.

## Combined prototype release0.1.30

Released at https://github.com/aataat12/immich-clone/releases/tag/v0.1.30-prototype
after combined acceptance41569614980, source042547e877f0c8957b7029ff6aa6d4ad4da9f8b0,
job125699201968 passed on Windows Server2022/Edge. Actual0.1.29 Setup baseline,
failed/successful cross-provider upgrade, installed HEIC/sample-video inference,
failed0.1.31 rollback preserving6A samples/names, uninstall retention and restored
reinstall/frame regeneration passed. Installer SHA256:
ec78e9d62b4d462f3b1448310dad1cbda30cf278509af20426bbdcd49018a007,104160014 bytes.
Full Scan stays deferred. Target Windows10 acceptance remains separate; this is
an unsigned cloud-accepted prototype, not final compatibility certification.
The earlier combined-installer pending statements are superseded by this checkpoint.
