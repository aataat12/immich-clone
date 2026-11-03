Windows x64 prototype update: review-first additive reanalysis.

- Completed supported photos and sampled MP4/MOV videos offer **Prepare reanalysis proposal**, then **Apply** or **Discard**. Existing completed results remain available during preparation.
- Apply adds non-overlapping face detections and refreshes AI object suggestions. Existing face IDs, names, tags and dismissals stay intact. Existing detections are not replaced; overlapping new detections are skipped. There is no automatic naming or rerun.
- Pause/resume, per-frame video checkpoints and proposals persist across restart and application-consistent backups. Full Scan remains deferred; videos still sample up to 12 positions.

Create and download a backup before upgrading with the existing application/data locations. No separate dependencies are required. Reanalysis is optional; existing ready results do not need to be rerun.

Focused actual LocalService/Edge acceptance: https://github.com/aataat12/immich-clone/actions/runs/41651264618 . Combined installer acceptance passed: https://github.com/aataat12/immich-clone/actions/runs/41652777887 (source `73b8e9b910633b8525097d18e2eae37a2ecc1d1c`, job `125975035550`, first attempt). Actual 0.1.32 upgrade, both injected rollback paths, installed reanalysis, retained proposal restore/apply, uninstall preservation and independent restored reinstall passed.

Unsigned early prototype. Windows Server 2022 checks do not certify Windows 10; target-PC, clean-machine/reboot and extended reliability acceptance remain separate. Known intermittent restore progress-read:32 remains unexplained; a passing release test does not establish a fix. Group 2/3 mitigation and every-frame Full Scan remain deferred.

Published installer: 104169200 bytes; SHA256 `24fa4ab3713e59b29530ebd7c29d25b06da021bd0040539c2422d1d6db00dfd8`.
