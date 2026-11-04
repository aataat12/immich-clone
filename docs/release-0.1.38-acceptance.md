# 0.1.38 combined acceptance

## First combined attempt — installed media gate failed; no publication

[Run41769332340](https://github.com/aataat12/immich-clone/actions/runs/41769332340), source 79c869b3365521ee9be11683c92fcc08914a303d, job126375076678: FAILED at the new installed media matrix. No candidate installer artifact or release was published.

Core tests/vet, existing dependency check, real bounded full-disk recovery, full application/browser/backup restore, actual checksum-verified0.1.37 installation/upgrade and initial rollback passed. Installed HEIC/video AI, reanalysis, Hidden Media batch/navigation, source-loss import and backup/recovery-instructions download/discard also passed before the new gate.

The new24-image installed harness returned "Installed preview not ready" after its wait. That message did not name the fixture or distinguish a failed preview from a premature/stale observation. Do not infer decoder incompatibility or call this merely a harness issue without evidence. The existing pixel/metadata/original-checksum assertions remain required. The worker is responsible for a focused reproduction and exact diagnosis; root reviews preview routing and integration in parallel.

Focused evidence remains valid within its scope: People index scale/Edge41767370225, native400-photo quality41767731927 and standalone media/provider matrix41767014589. They do not replace the failed installed gate. Published baseline remains0.1.37 until combined acceptance succeeds.

## Focused installed-harness readiness correction

The pinned Playwright1.62.1 implementation in https://raw.githubusercontent.com/microsoft/playwright/v1.62.1/packages/playwright-core/src/server/frames.ts polls a synchronous predicate. An async predicate returns a truthy Promise on its first call, and fulfilling with that Promise adopts the eventual false value instead of polling until readiness. Source review identified this precise failure mechanism; it was not treated as proof of a decoder defect.

Focused Windows run41770783033 (https://github.com/aataat12/immich-clone/actions/runs/41770783033), source fd52bbe76068b7aac6197252be725ca4ed84aa03, job126379721501: PASS first execution. It rebuilt the exact failed candidate's production serve path with the same pinned media provider and24 fixtures. A controlled use of the old async wait returned false in46ms while22 real previews were still queued. The corrected bounded Node polling loop fetched fresh catalogs, waited21,124ms for all24 Ready states, then passed every unchanged geometry, metadata, pixel and original-hash assertion.

Only the new media harness's wait is corrected:90s total bound, immediate refusal of an actual failed fixture, clear public-fixture diagnostic, and use of the verified all-ready catalog snapshot. Read-only restored mode still forbids uploads. No production decoder, format, provider, timeout/size policy, assertion or unsupported-format designation was changed. Other historical waits were not audited or changed. This focused serve proof does not replace actual installed LocalService and restored-reinstall acceptance; combined acceptance must pass before publishing.


## Accepted combined delivery: 0.1.38 published

[Run41771361544](https://github.com/aataat12/immich-clone/actions/runs/41771361544), source86807bff81930a5394c60d695185bbdb7008d81c, job126381550300: PASS first combined execution after the focused wait correction. Actual checksum-verified0.1.37 upgrade, both rollback paths, installed native HEIC/video AI/reanalysis, independent API pagination, Hidden Media batch/navigation, source-loss import and backup/recovery controls passed. Core tests/vet, existing dependency checks, full-disk recovery and application service/browser/backup restoration passed.

The corrected installed matrix waited for actual Ready states and passed every one of24 image geometry/metadata/pixel/original-hash checks under LocalService. Independent pre-deletion archive restoration and full old-state-offline restoration/reinstallation passed. After reinstall, the read-only matrix verified all24 preexisting restored identities and regenerated previews without uploading or repairing any original; all hashes/metadata/pixels matched again. Restored names/tags, video observations/corrections, reanalysis proposals and frame/crop access also passed. Both uninstalls retained expected data and completed cleanup.

Published [0.1.38-prototype](https://github.com/aataat12/immich-clone/releases/tag/v0.1.38-prototype),104194388 bytes; GitHub asset SHA25614e1f0dd6b5bc709d45235963e137e47ee5740fee13ec3c8c03d4756ddd24038. Release notes include the accepted combined run and focused target-PC checklist. This is an unsigned Server-tested prototype, not final Windows10 certification.

Milestone2's current functional scope and Milestone6's declared existing-format/resource scope are delivered. Original Group2/3 security work remains deferred and is not relabeled complete. The finite corpus/scale/format limits in the linked evidence remain explicit. RAW/HDR expansion, Full Scan, physical-storage outage/endurance and final clean-machine/reboot gates remain separate. No further work is claimed or scheduled beyond this requested delivery.
