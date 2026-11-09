# 0.1.44 prototype acceptance record

## Status

The steps 9 and 10 acceptance harnesses are prepared. The exact-source hosted
run, installer identity, and publication are pending. Steps 1–8 had separate
development acceptance on disposable Windows Server 2022 runners.
See [the prior evidence](browsing-0.1.44.md).

## Release gates

| Gate | Coverage | Result |
|---|---|---|
| Combined catalog | 100,000 synthetic metadata rows and 60,000 recorded import entries. Combined album, favorites, caption, people, tags, media, folder, batch and leap-day filters. Compare exact pages, counts, album order, viewer neighbors, return anchors and same-batch provenance. Record timings. | Pending hosted execution |
| Complete browser experience | Edge and LocalService, 65 synthetic originals, cross-page selection and viewer navigation, favorites, manual album order and cover, caption conflicts in two sessions, combined source and calendar filters, lost batch response, restart and explicit continuation. Take HTTP backup, move the source state offline and verify restored metadata, results and all 65 original hashes. | Pending hosted execution |
| Exact Setup | Fresh install, actual bundled providers, the integrated browser journey on the installed service, independent restore and reenrollment, genuine checksum-pinned 0.1.43 upgrades from catalogs 1 and 2, before/after-write rollback and older-version refusal. Verify the Setup checksum after tests. | Pending hosted execution |
| Publication | Match the accepted commit, successful run and artifact ZIP digest. Check the four release files by size and SHA256, upload them to a draft, verify the remote digests, then publish the prerelease at the tested commit. | Pending an accepted artifact |

The release gate runs on the existing `work/release-0.1.44-foundation` branch.
The installer job depends on both the catalog and browser jobs in the same
run. Only a passing installer job uploads `prototype-0.1.44-server-tested`.
The setup compiler, pinned providers and tools are acquired on the runner.

## Exact artifact handoff

After a green acceptance run, record the commit, run, successful job IDs,
artifact ID and ZIP SHA256, then independently download the ZIP and record the
size and SHA256 of each of these four files.

- `dist/ImmichClone-0.1.44-prototype-x64-setup.exe`
- `dist/SHA256SUMS.txt`
- `docs/release-notes-0.1.44.md`
- `docs/validation-0.1.44.md`

Supply those independently recorded values to
`scripts/Publish-Release44.ps1 -Manifest <file>`. The JSON fields are
`source`, `run`, `artifact`, `archive_sha256` and `assets`. The keys under
`assets` are the four archive paths above, each with `sha256` and `size`.
The default invocation verifies the artifact. Adding `-Publish` uses the same
verification before creating or completing a matching draft and checking the
public release. The publishing session requires GitHub access to read Actions
artifacts and write releases in this repository. Record the public tag, URLs
and installer checksum in this acceptance record after verification.

## Remaining validation limits

The combined scale test uses synthetic metadata rather than full-media
ingestion. Timings apply to the runner and its disposable fixture. The
interruption scenario covers a lost batch response and service restart.
Windows 10 clean-machine compatibility, full OS reboot, physical power loss,
full-library endurance, final target recovery and code signing remain open.
See [validation scope](validation-0.1.44.md).
