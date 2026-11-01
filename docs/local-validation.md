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
5. When convenient, perform one normal PC reboot and check that the library starts
   again and both synthetic originals remain downloadable. No automatic reboot or
   OS update is requested or scheduled by the application.

Return only the PASS lines, a brief description of browser/reboot behavior, or the
failure message. Do not post transcripts, hardware reports, paths or access codes
to GitHub. No personal machine information is needed in public project records.

Uninstall preserves the application data folder and originals. JSON catalog export
and ZIP backup are available in the UI. This build lacks AI, generated video
thumbnails, EXIF orientation processing, HEIC/RAW/HDR preview validation, physical
disconnect testing and extended endurance. Browser folder copies flatten source
directory structure. Do not bulk-import the real library yet.
