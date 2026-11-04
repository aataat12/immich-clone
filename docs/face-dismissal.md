# Reversible face dismissal — next-release candidate

People review offers Not a face and a separate Dismissed faces view with Restore
face. Dismissal preserves the stored detection/crop, name and name-undo history.
It removes the face from active reference suggestions, the active name list and
name-based photo search; restore returns it without rerunning AI. Custom tags,
ready analysis, manual geometry and original bytes remain independent. Both
detected and manually marked faces can be corrected this way.

An additive dismissed boolean lives in the existing face-label record; no new
bucket or original/backup format. Consistent catalog backup and embedding-free
JSON export retain the flag. Older executables do not implement dismissal and
may drop this additive field if they edit the label; no cross-version correction
editing guarantee is claimed. Upgrade rollback remains gated on installer tests.

Linux Go 1.27.3 focused race tests passed: named-reference positive control,
dismissed reference exclusion, preserved names/undo across reopen and independent
backup restore, tags/ready AI/original verification. HTTP tests reject missing
authentication, foreign origins and unknown input fields; valid dismiss/restore
and separate review work. Full library/web/cmd tests and vet passed. Windows/Edge
acceptance is pending. No installer/version change or local import requested.

Remaining milestone work: safe ready-analysis reruns, larger review/search
performance and broader recognition quality. Security worker source reviews remain
release gates; a new feature does not clear or defer them silently.


Windows Server 2022/Edge acceptance passed:
[41519696666](https://github.com/aataat12/immich-clone/actions/runs/41519696666),
source 7c49a78e4dd4c714c87148c5a2a2885c24776f2f, job 125532231288.
Library restart/independent restore and HTTP protection tests passed. Actual Edge
used a synthetic PNG/manual detection, clicked Not a face, reloaded, opened the
dismissed view and restored the original name. Existing five-page desktop/narrow
CSP/session checks and ordinary-user ACL/LocalService upload/restart passed.
No native inference/recognition-quality or Windows 10 proof is implied.

Initial browser run 41519176696 failed setup before dismissal: a copied PNG had
extra pixel data and Go correctly rejected decoding it (manual-face returned 400).
Reproduced locally with image/png; the preexisting validated PNG decoded cleanly.
Only the browser fixture changed for retest; application validation stayed intact.
Linux signature baseline: 810 text inputs, zero matches/exclusions, with its
existing arbitrary-password/ignored/artifact/history coverage limitations.
No installer/version or main merge. Security acceptance, full installer upgrade/
rollback and target checks remain separate gates for the broader release.
