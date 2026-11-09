# 0.1.44 validation scope

The release acceptance record identifies the tested source, run, jobs, installer
SHA256 and published assets.

## Disposable Windows Server 2022 gates

- Exercise selection, favorites and albums, caption edits, source and calendar
  filters, and viewer navigation together. Cover competing edits in separate
  sessions, an interrupted batch response, service restart and explicit
  continuation. Compare the restored query, metadata and originals after
  restoring from backup while the old state is unavailable.
- Exercise a bounded large synthetic catalog and report the workload and
  measured operations alongside the results. The acceptance record gives the
  achieved scale and environment.
- Verify fresh installation with spaces in the paths, the packaged binary,
  LocalService, automatic startup configuration, independent restore,
  reinstallation and uninstall preservation.
- Upgrade the checksum-verified published 0.1.43 installer with starting
  catalogs 1 and 2. Inject failures before and after format-3 metadata writes.
  Verify the restored executable and catalog, access code, original hashes
  and face provenance. Confirm genuine 0.1.43 read-only and service modes refuse
  format 3 without changing source bytes.
- Exercise the pinned installed media and native AI providers, backup/restore,
  package privacy and the retained installed regression suite. Publish the
  same Setup file accepted by these gates.

## Focused Windows 10 checks after prototype publication

Retain a verified pre-update backup and the preceding installer. Update with
the existing folders and confirm the reported build is 0.1.44. Check that
ordinary browsing, media, existing names and tags still work.

On a small set of existing items, exercise a combined filter and calendar
choice, open an item, move through the filtered results and return to the grid.
When there is a normal reason to organize those items, try a small selection,
favorite, caption and album edit. If recorded source folders or import batches
are available, check their matching results. Use disposable fixtures for any
interruption or independent restore drill.

The package is unsigned. Windows 10 clean-machine acceptance, a full OS reboot,
physical power-loss behavior, full-library endurance and final target recovery
remain open. This focused check does not require expanding the library, adding
application dependencies, rebooting or replacing the existing catalog.
