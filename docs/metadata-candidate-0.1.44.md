# Internal 0.1.44 metadata and recovery candidate

This package is a disposable-runner test fixture. It is not a released installer
or a claim of Windows 10, reboot or endurance acceptance.

The candidate tests viewer navigation and catalog-3 storage for favorites,
title/caption/notes, albums, ordered membership and covers. Original files and
recognition provenance remain authoritative and separate. Catalogs 1/2 remain
unchanged until a real new organization write promotes them atomically to 3.
Older releases require a compatible earlier backup to operate after promotion.

Organization bounds: UTF-8 title 256 bytes, album description 4096 bytes, caption
16 KiB, notes 64 KiB, 1000 albums, 5000 members per album, 100000 memberships
total and 64 MiB total serialized organization values. Existing backup/restore
resource bounds are retained. These limits are enforced before committing writes.

The existing installer failure injection has bounded observation delays only
when both IMMICH_INSTALL_TEST_FAILURE and IMMICH_CURATION_INSTALL_TEST equal 1.
They permit the disposable test to commit ordinary authenticated metadata and
inspect exact restored bytes before the old service rebuilds indexes. Neither
flag bypasses schema, authentication or filesystem validation. Ordinary Setup
execution has no delay or behavior change.

Acceptance and exact source/run identities will be recorded after execution.
