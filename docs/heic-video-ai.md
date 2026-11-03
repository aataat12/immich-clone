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
suite/vet and focused race checks passed. Real application/browser/restart and
independent-restore acceptance is pending the focused media-ai workflow.
No installer/version/main update or Windows10 certification is claimed.
