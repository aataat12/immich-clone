# Windows 10 compatibility checkpoint

This is an early unsigned test build, not the finished application. Server 2022
results do not establish Windows 10 behavior. No dependency installation is needed:
the native executable and built-in Windows service facilities are packaged.

1. Obtain the installer from the successful run's `prototype-server-tested`
   artifact. Compare its SHA256 to that same artifact's `SHA256SUMS.txt`. Windows
   may show an unknown-publisher/SmartScreen warning because no signing certificate
   has been purchased. Do not disable antivirus or other protection to run it;
   report a block if it occurs.
2. Install with administrator permission. Select a **new, empty application data
   local NTFS folder outside cloud-sync folders**. This is managed storage, not the personal
   source library. Leave the real source library untouched for this checkpoint.
3. Open `scripts/Test-LocalPrototype.ps1`, copy its contents and paste them into an
   administrator Windows PowerShell window. It uses local HTTP only, starts from an empty library, uploads two tiny
   synthetic files, tests PNG decoding, restarts the service, checks download
   integrity and restores a backup into a temporary independent directory. Two
   synthetic originals remain in the test library. It neither enumerates hardware
   nor transmits specifications, reports, paths, credentials or personal media.
4. Open the Start menu entries **View access code** and **Open library**. Sign in,
   confirm the PNG preview/full-image viewer works, and keep the access code private.
5. At final acceptance (deferred by user), perform one normal PC reboot and check that the library starts
   again and both synthetic originals remain downloadable. No automatic reboot or
   OS update is requested or scheduled by the application.

Return only the PASS lines, a brief description of browser/reboot behavior, or the
failure message. Do not post transcripts, hardware reports, paths or access codes
to GitHub. No personal machine information is needed in public project records.

Uninstall preserves the application data folder and originals. JSON catalog export
and ZIP backup are available in the UI. Later builds add JPEG EXIF orientation,
BMP/GIF/TIFF and optional CPU HEIC/video previews; consult status and prototype
notes for tested formats and limits. AI, RAW/HDR previews, physical disconnect,
bulk import and extended endurance remain unaccepted. Managed blobs use content
hashes; relative source aliases are preserved by folder import and JSON export.
Do not bulk-import the real library yet.

Local functional checkpoint was reported passed on 2026-11-01, including full-image
view. Do not ask the user to repeat it without a relevant change or failure.
Post-reboot validation is explicitly deferred until the end of development.


## People and tags pilot: small private checkpoint

After the 0.1.15 Server acceptance gate passes, upgrade the existing installation
and confirm `0.1.15-prototype` at the top right. No new data location, dependency
installation, mass import or reboot is requested.

1. Open **People & tags**. Use two or three ordinary JPEG/PNG photos of the same
   person, plus a photo of someone else. Use different photographs, not copies.
2. Select **Analyze photo locally** on the first photo. Check the detected face
   crop, enter a name and save it. Analyze the others. Confirm a correct suggestion
   or supply the correct name; check that nobody gets named automatically.
3. Try **Undo name**. For a missed face, choose **Mark missed face**, drag a box
   and supply a name. Try selecting faces and applying an existing or new name.
4. Add a custom tag, find it with search, and dismiss/restore an incorrect AI tag.
   Refresh the page and check that your corrections remain.
5. Download an original and confirm it is still the original file. Report only
   whether analysis works, obvious missed/wrong matches, and correction behavior.
   Keep private photos, names, paths, access codes and raw logs out of GitHub.

This is a manually requested CPU pilot, not full-library recognition acceptance.
AI is limited to native JPEG/PNG/BMP, first GIF frame and first TIFF page, up to
64MiB/40MP, with a 960px working frame. Small/profile/aged faces may be missed;
80 common object categories are supported. HEIC/video AI, automatic clustering,
face dismissal and ready-result reanalysis remain deferred. A transformed public
fixture in CI proves the suggestion controls, not independent-photo accuracy.


## Later small bulk-import checkpoint (0.1.17)

No additional local action is requested while the 0.1.15 People/tag checkpoint is
pending. After that checkpoint, a small available-offline test folder can exercise
Bulk import: explicitly copy, check file/byte/duplicate counts, pause or close the
page, reselect the same folder and resume, then confirm original downloads. Cancel
keeps all accepted copies. The page explains that resume retransfers accepted
bytes to verify their receipts. This does not authorize importing the full source
library; paging, larger endurance and large-backup capacity remain open gates.


User-reported checkpoint update: all twelve 0.1.15 People/tag functional checks
passed, with original-photo comparison rather than newly reported SHA256 results.
0.1.17 Bulk import also reported working well. No repeat of these completed small
checks is requested. These reports do not establish broad recognition accuracy,
large-library/endurance or a clean-machine/reboot acceptance. Reboot stays deferred.


Focused 0.1.17-to-0.1.19 upgrade validation reported all requested navigation,
desktop/narrow/no-overflow, actual-version and preservation checks passed, including
photos, face names, tags and import history. No observed problems. No raw private
report retained and no further 0.1.19 test requested; reboot stays deferred.


0.1.21 gallery validation reported all requested checks passed with no observed
problems. No repeat of that completed checkpoint is requested. Backup changes
remain a separate small-library checkpoint after cloud acceptance; final reboot
and full-library import stay deferred.


## Small backup-page checkpoint (0.1.23)

After upgrade, confirm the actual build, then open Backups using the existing
small test library. Check required space, create a backup, wait for ready and
download it. Refresh the page and repeat the retained download. Once a downloaded
copy is kept, explicitly discard the temporary application copy and confirm the
gallery/names/tags remain available. Return only PASS/failure descriptions; keep
the ZIP private because it contains originals and the access code. No need to
enlarge the library to catch cancellation; cloud tests exercise it deterministically.
No dependency installation, AI retest, full-library import or reboot requested.
