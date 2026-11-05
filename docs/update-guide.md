# Manual prototype updates and recovery

Use the exact tested installer and matching SHA256SUMS.txt identified by its
acceptance record. The corrected **0.1.41 build** passed
[Windows validation](https://github.com/aataat12/immich-clone/actions/runs/41986219923);
[0.1.41 is published](https://github.com/aataat12/immich-clone/releases/tag/v0.1.41-prototype); target-PC validation is pending. See
release-0.1.41-acceptance.md and validation-0.1.41.md.0.1.40 target preparation failed
before payload replacement and is superseded by this build. Other internally
numbered rollback-injection packages are not accepted updates.

## Before an update

1. Keep an independently downloaded Ready backup and its matching recovery
   instructions. Backups contain originals and the access code; keep them private.
   A Ready temporary server copy alone is not an independent backup. Do not discard
   a retained backup merely to enable Create backup.
2. Obtain the installer and checksum list from the exact tested build handoff
   (or its identical published release). Compare SHA256 before executing.
   The 0.1.41 installer is 104,634,954 bytes, SHA256
   `2f61cca971e9d4bcbdd390f3809e9fcc55c7c1988ce40ce2e399dda799e2740a`.
   A checksum checks byte identity against the chosen record; it is not a signed
   publisher identity or independent proof against a compromised release account.
3. Retain the prior accepted installer and read the candidate's tested upgrade
   baseline and documented limits. No local SDK, Python, Node, GPU toolkit,
   database server, Docker or WSL setup is part of this application update.

## Install and check

Close/cancel the failed0.1.40 Setup before starting the correction. If the prior
service is unavailable, retain the existing independent Ready backup without
attempting a new live export.0.1.41 handles the owned stopped/disabled prior
service; manual permission edits or uninstall are unnecessary.

Run the accepted installer with its required administrator permission and retain
the existing application and data locations. Changing the live library location
is a separate backup/restore operation, not an upgrade. Setup manages the service;
do not manually replace executables, provider folders, catalog or access code.

The current prototype is unsigned. Do not disable antivirus or other protections
if Windows blocks it; report the block privately. No verified trusted-publisher
identity or SmartScreen outcome is promised. No signing purchase is authorized.

After installation, check the displayed version and existing library, names,
tags, AI results, hidden items, import history and retained backup. Apply only the
release's focused checklist; do not repeat passed old checks or import/reanalyze
media solely to fill an optional test gap. Routine prototype updates do not
request a reboot; final startup-after-reboot acceptance remains separate.

## If installation fails

Stop and check which version is actually running and whether the existing library
remains accessible. Tested failed-upgrade recovery restores the prior executable,
catalog and access code and retains immutable providers/originals; it does not
establish that every failure recovers automatically. Avoid additional writes or
manual binary/catalog swaps when state is uncertain. Keep the independent backup.

Export diagnostics using the existing application or Start menu control if
needed. Review and share the file only privately in the project conversation;
never commit it or put raw logs, access codes, media or machine inventory in public
issues, CI inputs or artifacts. A summary of the visible failure is sufficient
to begin triage.

For an independent restore, use [the recovery guide](recovery-guide.md) and the
downloaded archive's exact instructions/checksum. Restore into a new directory;
it does not switch the running library automatically. Do not overwrite live state
or perform a restore drill just to validate an ordinary update. Uninstall retains
application data, but uninstall is not a backup.

## Release preparation obligations

Preserve previously published tags/assets. A later candidate must be built with
its actual version, tested against the genuine prior accepted installer, subjected
to applicable rollback/preservation/restore checks and published only after all
required checks pass and final artifact checksums are verified. Never rerun the
0.1.39 publishing workflow merely to create fresh evidence: it targets the already
published release. Native/model/runtime changes retain their representative
execution gates. Security deferrals and final target acceptance remain explicit
in [milestone 7](milestone7-hardening.md).
