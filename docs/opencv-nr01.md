# NR-01 OpenCV security servicing

Status: candidate backports; actual Windows build/API/model/service execution
required before selection. Existing installed/app-pinned provider is unchanged.
No SEC item is closed by source readiness and no new product release is implied.

The exact retained OpenCV 4.12.0 source is kept, with zlib 1.3.2 and the previously
verified protobuf CVE-2022-1941 backport. Six reviewed upstream patches apply
cleanly to its production files: PR28272 NaryEltwise pointer allocation/strides;
PR28446 error-path UB; PR28579 U16 multiplication overflow; PR28581 countNonZero
function signatures; PR28959 integer point-dot overflow; PR29447 scalar/external
Mat expression lifetime. Full original patches, immutable hashes and origin URLs
are in experiments/native-ai/security-patches/manifest.json. Test hunks are not
applied to upstream source; a dedicated executable exercises these linked APIs.

Reason for this bounded backport: fix the known retained-source defects while
preserving models, DNN interface and dependency versions. A move to 4.14/5 would
also require review of later-introduced paths, including the separately reported
fastGemmThin issue absent from retained 4.12. Neither path is declared generally
safe. The chosen backports do not fix SEC-05 filesystem/token authority or certify
absence of other defects. Sources are official OpenCV PR patches referenced in
the manifest and security-review-closeout.md.

Build acceptance must verify every patch SHA before application, record resulting
source hashes, retain source/notices/patches, run actual API regressions and linked
zlib operations, and audit PE imports. Worker reports security_backports=nr01-v1;
its new SHA256 and immutable provider ID must be selected only after actual
face/object model parity, malformed/model failure, cancellation, resource limits,
Unicode paths and LocalService restart pass. API regressions include 64 actual
NaryEltwise sum/mean/min/max cases with 3/4/32/160 inputs, mixed scalar/row broadcast,
U16 saturation, five count depths, point-dot overflow and delayed scalar lifetime.
These are regressions, not sanitizer/fuzz proof; error-termination patch is
source-reviewed, not exercised as an exploitable worker path.

Application integration must retain existing ready results, names/tags/undo and
original hashes, validate new inference, and preserve old immutable provider for
rollback. Independent restore and final combined actual installer/Windows 10
checks remain SEC-08. Existing binary identity is not silently relabeled.

The first compiled candidate was rejected by the new numeric regressions. Source
inspection identified repeated mean scaling in the single-plane multi-input path:
for more than three inputs, prior partial results were divided again. A separate
project patch now scales each contribution once, matching the existing multi-plane
branch. It is explicitly not represented as an upstream security patch. The
original expected-value assertions remain, with four-input coverage added and
fixed synthetic case labels for diagnosis. Actual rerun remains required.
