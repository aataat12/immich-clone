Windows x64 prototype update: Hidden Media navigation and clearer import recovery.

- Hidden Media now has a prominent **Photos & videos Home** button. Clicking **Hidden Media** again also returns to Library. The heading, selector and pressed state stay synchronized.
- Bulk import explains when a source becomes unavailable or a connection is interrupted. Saved per-file errors explain changed-source conflicts; stale messages clear when retry starts. Earlier accepted originals and receipt/hash reconciliation remain authoritative.
- No format, provider, AI model or file-size limit changed. Milestone 4 remains in progress.

Focused cloud checks passed: actual Edge navigation run 41711667929; actual selected-source loss/changed-replay/resume under LocalService run 41715462566; 240-file PNG/HEIC/MP4/MOV batch with forced termination, resume, 192 original hashes/previews, 48 duplicate receipts and independent restored regeneration run 41718160138. The earlier 4 GiB HTTP boundary proof is reused. These are bounded synthetic checks, not full-library endurance evidence.

Publication is gated on actual 0.1.34 installer upgrade, failed-upgrade recovery, installed navigation/source-loss checks, media/AI/reanalysis regression, pending-deletion retention and retry, uninstall and independent restored reinstall. The pipeline appends its successful acceptance run when publishing.

Create and download a backup before upgrading with the existing application/data locations. No separate dependencies are required. Target-PC steps: [validation-0.1.35.md](https://github.com/aataat12/immich-clone/blob/work/security-baseline/docs/validation-0.1.35.md).

Unsigned early prototype. Windows Server 2022 results do not certify Windows 10. Clean-machine/reboot and extended reliability validation remain separate. Known intermittent restore progress-read:32 remains unexplained; a passing run does not establish a fix. Security Group 2/3 work and every-frame Full Scan remain paused/deferred.

Combined installer acceptance: [run 41722938850](https://github.com/aataat12/immich-clone/actions/runs/41722938850), source `734046903d88a0036b64243a4c6f2ee3a9173a75`, job `126220153707`: PASS, first attempt. Actual 0.1.34 upgrade and initial rollback, installed Home/toggle/selector and real source-loss checks, pending-intent retention through failed 0.1.36 upgrade, actual prior-aware-writer refusal, cleanup Retry, media/AI/reanalysis, uninstall and independent restored reinstall passed.

Published installer: 104212262 bytes; SHA256 `2b6ba59fa6f31eb3f2edd2e73d8ad003143a2306d0a2317dbea23e7d9e65e604`.
