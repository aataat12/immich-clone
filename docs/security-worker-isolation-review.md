# Native worker isolation review — 2026-11-02

Source checkpoint: `3973eade5d52e02328857c3499de964bc202c62c`.
This is a source/evidence review, not a newly executed Windows containment test.
Production code was not changed. Public/synthetic inputs only; no user media,
credentials or machine reports were inspected. User authorized parallel workers
for this review while functionality development proceeds; release gates remain.

## Current boundary

| Surface | Source-established behavior | Limit of that protection |
|---|---|---|
| Launch identity | `internal/media/job_windows.go:89` calls `windows.CreateProcess`, with no restricted-token creation, alternate token, AppContainer attribute or separate worker identity. `packaging/Manage-Service.ps1` registers the parent as LocalService. | Child launch does not introduce a security identity boundary below its parent. The installer’s use of LocalService limits the application relative to an administrator; it does not isolate child parsers from application state. |
| Resources/lifetime | Same launcher creates a Job Object before launch and supplies `PROC_THREAD_ATTRIBUTE_JOB_LIST`. Limits: 512 MiB committed memory per process/job, one active process, kill on last job-handle close. `internal/media/types.go` caps stdout at 1 MiB. Cancellation terminates the job and waits for confirmed process termination. | No configured CPU-rate limit or whole-service memory cap. These controls do not deny file, network or same-account process access. The caller supplies the deadline: the media wrapper has one 30-second budget for probe/decode; AI gives each of two sequential tasks 30 seconds. |
| Handles/environment/arguments | Explicit inherited-handle list contains only the output pipe and NUL. Input is NUL, stderr is discarded to NUL. Environment contains provider-only PATH and SystemRoot; working directory is the provider directory. Executable must be absolute and arguments use ComposeCommandLine, not a shell. | This reduces accidental handle/environment leakage and command ambiguity. A process can still attempt to open new objects permitted to its token. PATH/cwd are not a filesystem jail. |
| Media native input | FFmpeg/ffprobe receive a managed-original path. Provider hashes are checked at provider construction. Format/dimension/thread/probe/output guards and the file/pipe protocol whitelist constrain intended operations. | Protocol configuration is not an OS network restriction. Filesystem access is not limited to the path passed on the command line. Startup hash verification is not a lock on the executable for its entire lifetime. |
| AI native input | `internal/recognition/provider.go` checks the exact worker and three fixed model SHA256 values before each analysis, writes an ICAI BGR frame in staging, then runs faces/objects with fixed task/model/frame arguments. App frame bound is 960px, minimum 32px; no arbitrary model upload. C++ accepts at most 1600px and bounds each serialized model to 45 MiB. JSON response dimensions, finite values, confidence/counts and embedding norm are validated before results persist. | Narrow inputs reduce exposure; they do not contain native code after a parser/inference defect. Path-based verification and later native opens are separate operations. No source finding here demonstrates an exploitable parser defect. |
| Application state ACL | `packaging/Storage-Security.ps1:28` intentionally grants LocalService Modify on data/log roots. The child uses the service identity. | Conditional on successful child compromise and ordinary inherited state ACLs, catalog/access code/originals are within that identity’s filesystem authority. Immutable-original application logic and SHA256 checks are not an OS write denial against such a child. |
| Parent image decoding | `internal/library/organization.go:152` decodes original JPEG/PNG/GIF/BMP/TIFF in the Go service before BGR conversion. `internal/library/store.go:446` also runs native-Go preview decoding in the parent. Dimension/encoded-size guards are present (AI 64 MiB/40MP; preview encoded guard specifically BMP/GIF/TIFF). | A stronger native-child boundary would not automatically sandbox the Go decoders or impose the child Job Object’s memory/deadline budget on their allocations/CPU work. Do not describe all media parsing as out of process. |

## Finding and severity

**WI-01 — confirmed missing child/state containment; exploitability not established.**
Priority: medium design-hardening work, with **high potential confidentiality and
integrity impact if a native child is compromised**. The source supplies resource
containment and input constraints but no lower-privilege child token/file boundary.
No malicious media exploit, catalog modification or secret theft was executed by
this review. Do not report this as an established high-severity parser CVE or as
safe execution of arbitrary hostile native code.

The current root ACL fix protects against unrelated ordinary accounts. It cannot
separate the parent and a child sharing LocalService. It also does not provide
per-service separation from other processes using that same built-in identity.
There is no explicit launcher network-denial policy. Do not infer egress isolation
from absent codec networking or a minimal environment. Host administrator/OS
compromise remains outside the project's application containment boundary.

**WI-02 — parent decoder exposure remains outside child resource controls.**
Priority: medium review/availability hardening. Existing size/dimension checks and
context-aware reads are useful, but no corpus result in this review establishes
bounded worst-case parent decode CPU/allocations for every malformed supported
image. This is a remaining negative-test obligation, not a demonstrated exploit.

## Existing execution evidence to reuse

These are recorded accepted runs, summarized from `docs/dependency-evidence.md`
and `docs/security-baseline.md`; this review did not rerun or re-download them.

- [41346172423](https://github.com/aataat12/immich-clone/actions/runs/41346172423):
  LocalService controls allowed 600 MiB allocation and process creation outside
  the job, then refused them inside; 64 MiB remained usable. Crash, timeout,
  output flood and parent-death cleanup were exercised. No orphan survived.
- [41347993775](https://github.com/aataat12/immich-clone/actions/runs/41347993775):
  the media probe used the shared `internal/media` launcher under LocalService.
- [41483644241](https://github.com/aataat12/immich-clone/actions/runs/41483644241):
  exact replacement AI worker executed native inference/parity, malformed frames,
  cancellation/model-failure recovery and restart under LocalService; reported
  peak commitment 271380480 bytes within the 512 MiB job cap.
- [41487369408](https://github.com/aataat12/immich-clone/actions/runs/41487369408):
  actual integrated installer/provider inference, upgrade/rollback, preserved old
  AI/corrections, independent restore and original hash checks passed on Server.
- [41513726020](https://github.com/aataat12/immich-clone/actions/runs/41513726020):
  unrelated ordinary-user read/write denial and LocalService upload/restart passed
  after root ACL replacement. This did **not** test denial to a native child using
  LocalService or demonstrate child-specific isolation.

All are Windows Server 2022 evidence. None certifies the target Windows 10 PC,
an OS sandbox or absence of vulnerabilities. Keep those limits in release notes.

## Smallest useful next experiment

1. Add a disposable Windows canary helper launched by the **exact production
   launcher under LocalService**, without native dependencies or real media. Give
   it intended input/model/cwd access plus synthetic sibling secret/catalog and
   original-write canaries. Record only success/denial booleans and token facts
   needed for acceptance, never canary values or paths. An unrestricted same-token
   control must first prove that the fixture would expose the canaries.
2. Require the control to demonstrate the current boundary honestly: job memory,
   process-count and cancellation constraints can pass while canary reads/writes
   remain allowed. This establishes actual file/token behavior before choosing a
   mitigation rather than inferring it from ACL text alone.
3. Time-box one isolated **restricted-token/per-job ACL** prototype. Its acceptance
   target is read-only required executable/models/input, permitted private scratch
   only, denied state secret/catalog/original writes and denied unintended input
   reads. Removing privileges alone is insufficient if LocalService's state ACE
   still authorizes access; demonstrate denial using actual access attempts.
   Whether this can run reliably from the existing service account is an open
   experiment. Do not integrate an unexecuted token/API design.
4. If that cannot meet file and network requirements, compare an AppContainer or
   separate low-privilege worker identity/broker. These would change launch/ACL/
   packaging and possibly IPC. Document the smallest concrete design and ask the
   user **before** implementing a choice that conflicts with the agreed plan.
   Do not silently add another service, enrollment, local SDK, Linux or VM.
5. For a successful candidate, rerun existing memory/spawn/crash/flood/death tests,
   exact FFmpeg decode and AI model parity, Unicode/space paths, concurrency,
   interruption/retry, original hashes, backup/restore and installed upgrade/
   rollback. Probe loopback and outbound access with disposable controlled targets
   before claiming any network isolation. Then execute applicable Windows 10 checks.

Separately add a small malformed Go-image corpus with measured parent allocations,
wall time, cancellation and post-failure service usability. Use actual decoded
operations, not only successful DecodeConfig. Moving a decoder into another
process is an option only after measurements identify a need and the new boundary
is proven; it is not prescribed as an untested rewrite.

If stronger isolation is deferred for a prototype, retain explicit resource-only
wording, pin/update the supported codecs/models, preserve independent backups,
and record WI-01/WI-02 as open. A documentation-only deferral does not clear a
production hostile-media security acceptance gate.
