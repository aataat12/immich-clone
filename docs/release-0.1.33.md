Windows x64 prototype update: review-first additive reanalysis.

- Completed supported photos and sampled MP4/MOV videos offer **Prepare reanalysis proposal**, then **Apply** or **Discard**. Existing completed results remain available during preparation.
- Apply adds non-overlapping face detections and refreshes AI object suggestions. Existing face IDs, names, tags and dismissals stay intact. Existing detections are not replaced; overlapping new detections are skipped. There is no automatic naming or rerun.
- Pause/resume, per-frame video checkpoints and proposals persist across restart and application-consistent backups. Full Scan remains deferred; videos still sample up to 12 positions.

Create and download a backup before upgrading with the existing application/data locations. No separate dependencies are required. Reanalysis is optional; existing ready results do not need to be rerun.

Focused actual LocalService/Edge acceptance: https://github.com/aataat12/immich-clone/actions/runs/41651264618 . Combined installer validation is required before this workflow publishes the prerelease.

Unsigned early prototype. Windows Server 2022 checks do not certify Windows 10; target-PC, clean-machine/reboot and extended reliability acceptance remain separate. Known intermittent restore progress-read:32 remains unexplained; a passing release test does not establish a fix. Group 2/3 mitigation and every-frame Full Scan remain deferred.
