## Milestone 7 authorization — November 4, 2026 (America/Chicago)

The user explicitly approved resuming the five original open items SEC-02,
SEC-03, SEC-05, SEC-06 and SEC-08 for milestone 7. This supersedes their Group 2
work deferral, not their classification or closure criteria. SEC-01/04/07 remain
closed; Group 3 investigations and paid signing remain deferred. No parallel
workers or new charges are authorized. Final target acceptance remains distinct.

## Authorized execution amendment — November 1, 2026

The user explicitly authorized developing and validating on the available cloud
Windows Server 2022 runners, then validating Windows 10 locally once a working,
packaged build is ready. This supersedes requirements below that make cloud
Windows 10 access a prerequisite to implementation or local testing. No paid
Windows 10 cloud machine is required. All other preservation, evidence, installer,
privacy and cost requirements remain in force. Server results do not certify
Windows 10. No local dependency setup until the relevant Server validation passes.
The original project-source text is retained below; this amendment records the
later user instruction rather than silently changing its historical meaning.

---

Build a reliable, Windows-native, self-hosted photo/video manager with an Immich-style browser experience. Favor correctness, recoverability, straightforward installation, and maintainability over rapid feature accumulation.

## Nonnegotiable requirements

- Target the user’s normal Windows 10 PC. Determine the exact Windows edition, build, CPU architecture, and relevant hardware before finalizing compatibility requirements.
- No Linux, Docker, WSL, or virtual machine requirement on the user’s PC.
- Mobile applications, mobile backup, and compatibility with Immich’s mobile clients are out of scope.
- Develop and test in the cloud using the online workspace and https://github.com/aataat12/immich-clone.
- Do not ask the user to install application dependencies locally until the applicable cloud validation has passed.
- Deliver a Windows installer with all required redistributable dependencies bundled or supplied in a clearly identified prerequisite package. Avoid requiring development tools on the user’s PC.
- Comparable reliability to Linux Immich is an acceptance goal, not a claim to make before supporting evidence exists.

## Established facts and limits

A diagnostic successfully built a self-contained .NET 8 application on GitHub’s Windows Server 2022 runner, registered and restarted it as a Windows service, initialized a PostgreSQL 14.24 database from preinstalled binaries, verified persistence across a database service restart, and uploaded/downloaded the report.

This established the cloud workflow and artifact retrieval path only. It did not validate Windows 10, a finished installer, clean-machine prerequisite installation, ONNX, media libraries, GPU acceleration, full OS reboots, or extended reliability.

The repository is currently public. Keep credentials, personal media, and private machine details out of commits, logs, and artifacts.

## Prove compatibility before committing to components

Treat every proposed dependency as a candidate until verified. This includes .NET, PostgreSQL, ONNX Runtime, AI models, FFmpeg, image libraries, metadata tools, installer tooling, and native runtimes.

For each critical dependency, document:

- Exact version, architecture, official Windows support, and required Windows build.
- Native DLLs, runtimes, CPU instruction requirements, and any GPU/driver requirements.
- License, redistribution rights, model-weight terms, and acquisition source.
- Whether installation and execution work unattended under the intended Windows service account.
- Update/security support horizon and a practical replacement or migration path.
- A small executable test covering the actual operation we need.

“Available on Windows,” a successful package restore, or a successful compile is insufficient. Execute representative operations using the selected binaries and configuration.

For ONNX/AI, test loading the actual intended models and running representative inference. Verify preprocessing, outputs, native dependencies, memory use, service execution, and model licensing. Begin with CPU execution; make GPU acceleration optional unless explicitly approved.

For media tools, test the promised image/video formats with representative files, including HEIC/HEIF, RAW, HDR, and unusual metadata where those formats are in scope. State unsupported formats plainly.

Do not build substantial downstream features around an unproven critical dependency. Time-box feasibility experiments. If an approach fails, record the evidence, recommend a viable alternative, and resolve the blocker before deepening the dependency.

## Windows 10 verification

A Windows Server runner is useful supplemental evidence, not a substitute for Windows 10 validation.

Establish a legitimate, accessible cloud Windows 10 test environment early. Verify availability, licensing, cost, permissions, and automation capabilities before promising it.

Use official current sources to verify support and lifecycle information. Distinguish “runs successfully” from “vendor-supported.” If Windows 10 or a required component has support/security limitations, explain the concrete consequence and obtain a decision before making an incompatible architectural commitment.

If suitable cloud Windows 10 testing is unavailable, report that blocker promptly. Do not quietly replace it with Server/Windows 11 tests or ask for local installation as though the cloud validation requirement had been met.

## Scope and architecture

Maintain a short agreed feature list and measurable milestone acceptance criteria in the repository. Identify which Immich-style features are included, deferred, or excluded before treating “clone” as full parity.

Deliver tested vertical slices: installation, service startup, browser interaction, durable storage, recovery, and removal. Establish basic upload, original-file preservation, metadata, thumbnails, browsing, and retrieval before layering on advanced features.

Use mature components where they reduce risk. Keep AI optional and replaceable so a model/runtime failure cannot prevent basic storage, browsing, or retrieval.

Keep originals authoritative. Generated thumbnails, transcodes, and AI indexes should be reproducible. Use versioned schemas and tested migrations. Preserve useful metadata and provide exports that avoid trapping the user’s media in proprietary storage.

Avoid unnecessary infrastructure, framework changes, or rewrites. Record significant architectural decisions and the evidence supporting them.

## Reliability and release acceptance

Never modify or overwrite originals silently. Use checksums and safe write procedures. Handle duplicates deliberately.

Persist background work and recover safely from interrupted uploads, failed jobs, service crashes, and restarts. Surface failures rather than reporting success prematurely.

Test full disks, permission failures, disconnected storage, malformed media, concurrent operations, and interrupted processing as those paths are implemented.

Provide application-consistent backup and restore procedures for database, originals, configuration, and required secrets. Perform an actual restore test.

Test the installer on a clean supported Windows 10 environment with no development SDKs or assumed preinstalled dependencies. Exercise fresh install, prerequisite provisioning, service permissions, reboot startup, upgrade, failed-upgrade recovery, and uninstall. Preserve user media by default.

Use production-appropriate service accounts and authentication. Diagnostic shortcuts such as LocalSystem or trust-based database authentication do not automatically become production defaults.

Pin release dependencies, verify downloaded artifacts, and retain reproducible build instructions. Record release checksums. Explain code-signing requirements and any unsigned-installer limitations before delivery.

Label prototypes, test builds, and validated releases accurately. Never equate a green CI result with complete product validation.

## Efficient execution and continuity

Keep compact, authoritative repository records for current status, accepted requirements, dependency evidence, architectural decisions, test results, and the next milestone. Update them when meaningful decisions or results change.

At the beginning of a work session, read the relevant current records and repository state. Reuse verified findings rather than repeating research. Investigate again only when versions, requirements, source changes, or unresolved failures justify it.

Work in small, reviewable changes. Run focused checks first; run broader checks at milestone/release boundaries. Do not repeatedly rerun passing tests without a reason.

Use bounded diagnostics before long builds. Cancel obsolete jobs. Minimize unnecessary logs, large artifacts, duplicate documentation, and repeated full-file output.

Do not launch parallel agents unless the user requests them or a later project instruction explicitly authorizes them. Give each worker a distinct, bounded task and reuse its findings.

Proceed autonomously with authorized, reversible work. Ask only for missing information or decisions that materially affect compatibility, cost, security, scope, or an irreversible action. Obtain approval before purchases, paid infrastructure commitments, or increasing spending limits.

## Communication and honesty

Keep explanations concise. Report meaningful progress during active work and always provide an outcome when the turn ends.

State what was verified, the environment used, what remains unverified, and the next concrete step. Link to relevant commits, runs, and findings.

Report blockers promptly and identify the smallest action needed to unblock progress. Never imply access or permissions merely because a tool exists.

Do not promise unattended continuation or later notification unless an actual mechanism has been established. Background workers report through this conversation unless a separate communication mechanism is explicitly available.

Do not drift into unrelated features, local setup requests, or cosmetic polishing while core compatibility and reliability blockers remain unresolved.

## User priority amendment — November 2, 2026 (America/Chicago)

Proceed with milestone 6A HEIC/video AI implementation alongside Group 1 security
mitigation: Internet-exploitable attack surfaces created on the user's machine
by the app or its installed supporting components. Pause other Group 2 security
mitigation for later; retain findings and implemented controls. This supersedes
prior ordering requiring the entire security backlog before feature development.
Normal original preservation, recovery, representative dependency execution and
truthful release/Windows 10 acceptance remain. See docs/security-priority-groups.md.

Clarification: Groups 1 and 2 classify only original SEC-01..SEC-08. Newly
identified issues are Group 3, recorded without investigation or mitigation time.
Only Group 1 receives security work time; 6A functional implementation continues.
