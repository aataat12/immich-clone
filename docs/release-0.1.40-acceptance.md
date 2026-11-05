# 0.1.40 release acceptance

**Historical Server gate PASS and published prototype; target preparation FAIL.**

The target update subsequently refused normal ProgramData creation permissions
before payload replacement and left the retained prior service stopped/disabled.
This release is superseded by the 0.1.41 corrective build. See
[0.1.41 acceptance](release-0.1.41-acceptance.md). Private support logs stay local.

The user requested 0.1.40 after acceptance of the combined SEC-03/UI checkpoint.
The release retains the accepted runtime and closed SEC-01/02/03/04/07 controls.
This packaging step changes the actual version and assertions to 0.1.40, uses an
unpublished 0.1.41 failure fixture, updates partial milestone 7 Build Notes, and
adds release notes, validation instructions and branch-restricted publication.
No new provider, model, runtime, schema or feature implementation was introduced.

## Exact source and validation

Source `5fcab695e086403c8e9bb1fdb339a7dc697b9eb6` passed Windows Server 2022
[run 41979194020](https://github.com/aataat12/immich-clone/actions/runs/41979194020),
job `127085130561`. The published tag points to that exact tested commit.

All required steps passed: genuine checksum-verified published 0.1.39 upgrade,
initial and late injected rollback, full pinned native media/AI execution,
package privacy canaries, actual desktop/narrow Clear UI under LocalService/Edge,
backup/range downloads and independent recovery. Restored reinstall with the old
state unavailable retained authentication, original hashes, HEIC previews, names,
tags, 6A observations/corrections, proposals and frame/crop context; the restored
24-image browser compatibility matrix and clean uninstall passed. Go tests/vet,
dependency verification/vulnerability scan and bounded NTFS ENOSPC passed.
Earlier same-runtime native/actual Setup/separate-account concurrency evidence
remains in [SEC-03 acceptance](sec03-combined-acceptance.md).

The malformed-link preparation run was cancelled before installer/release
publication. The corrected source above supplied all final assets. Publication
was restricted to work/release-0.1.40 after checks and artifact retention, and
verified the installer checksum before creating the release. It refuses an
already published release. Previous published tags and assets were retained.

## Verified published files

[0.1.40 prototype](https://github.com/aataat12/immich-clone/releases/tag/v0.1.40-prototype)
is published as a prerelease, not a draft. Installer asset712328558 is104650356
bytes, SHA256 `ec49d308d237cc0a639d5e9b548c9cda04de0d404efd23168ef40d6a7688f292`.
SHA256SUMS.txt, release-notes-0.1.40.md and validation-0.1.40.md are included as
permanent release assets.

Test artifact13393786248 ZIP SHA256
`2ec60724104ce5c7a064d0add7af94c2d6b0d1dff37f828a709fb3acec0a61f7`
was downloaded and hash-checked. Every extracted installer/checksum/note file
matched its published release asset SHA256; the installer also matched the
packaged SHA256SUMS.txt. Nothing was executed on the target PC by the build chat.
Synthetic browser artifact13393388387 ZIP SHA256
`41c199f8c362e6d9e46d94afe548daa6c6a029ffe48c0de179a97ae90e78809d`.

## Target handoff and limits

Target-PC 0.1.40 results are pending. [The validation assignment](validation-0.1.40.md)
covers a focused same-folder update and existing-data/navigation regression.
Prior exercised 0.1.39 user checks remain accepted. No routine reboot, destructive
recovery drill, extra import or forced processing is requested.

Milestone 7 remains in progress: security 5 closed / 3 open, SEC-05/06/08 remain.
Server evidence does not certify final Windows 10 clean-machine prerequisites,
reboot/startup, endurance or physical power-loss. No Group 3 work, new charge or
signing purchase was introduced. The installer remains unsigned.
