# Immich Clone 0.1.44 prototype

- Browse with previous and next controls across gallery pages. The viewer
  supports keyboard navigation, zoom, pan and full screen. Closing it returns
  to the current item's position in the filtered gallery.
- Select up to 500 items across pages. Review the exact targets before applying
  favorites, custom tags, hiding or album membership. Operations save progress
  so that completed work can be reviewed after an interruption and paused work
  can be continued.
- Favorite items from the gallery or viewer. Create manual albums with
  descriptions, covers and custom ordering. Removing an album preserves its
  media.
- Add searchable titles, captions and notes. Browse folders and batches recorded
  during import. Folder browsing uses catalog records and leaves source folders
  alone. Earlier imports without those records may have no source folder.
- Combine people, tags, dates, media type, favorites, albums and source filters.
  Use the calendar to jump to a year, month or day with matching media.

For an update, retain a verified backup and the prior installer. Run Setup
with the existing application and data folders. Setup requires administrator
permission and registers the application under the restricted LocalService
account. The package includes its media and AI providers and requires no
separate SDK, Python, Docker or GPU. Uninstall preserves application data and
originals.

When a feature writes metadata that requires catalog format 3, the catalog
promotes to that version. Earlier releases, including 0.1.43, cannot open a
format-3 catalog. Use the pre-upgrade backup if you need to return to the
earlier version. Keep access codes, backups and exports private.

This is an unsigned prototype. Windows Server 2022 results cover disposable
fixtures. Final Windows 10 clean-machine, reboot, full-library endurance and
target recovery validation remain open. SEC-05 (native worker isolation) and
SEC-08 (target acceptance) remain open.

The installed license notices accompany Inno Setup, Go, bbolt, x/sys, x/image,
the bundled media and AI providers, and the AI models.
