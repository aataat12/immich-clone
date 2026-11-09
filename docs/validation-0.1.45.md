# Immich Clone 0.1.45 validation scope

The release workflow retains Setup only after combined catalog and browser
regressions, Windows Server 2022 service and installer checks, and verification
that the tested installer matches the build checksum.

The browser fixtures cover the People directory, comparison, event and trip
grouping, filmstrip boundaries, edited membership and retry after a lost save
response. Installed checks include upgrading a catalog-three library from the
published 0.1.44, injected rollback and independent recovery of saved albums.

Clean-machine Windows 10, full OS reboot, physical power loss, full-library
endurance, target PC recovery and code signing require separate validation.
