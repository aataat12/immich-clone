# Independent backup recovery drill — prototype

This procedure restores a downloaded backup to a **new folder** for verification. It does not replace the installed service's live library. Use the executable from the same prototype version that created the archive, or a version with explicitly proven restore compatibility. Current baseline is 0.1.36. No dependency installation is required.

## Before starting

1. On Backups, create a backup and wait until Ready. Download it completely and retain the displayed archive SHA256 separately. Keep the ZIP private: it contains originals, catalog and the library access code. A ready archive on the same disk is not an independent copy; retain a verified copy on separate storage.
2. Select a local recovery destination outside cloud-sync folders. Its parent must already exist; the destination folder itself must **not** exist. Do not select the live application-data folder. Have space for the full extracted library plus the retained ZIP and reserve. The existing default capacity checks can refuse a restore; they do not reserve disk space against other applications.
3. Find the installed executable (normally `C:\Program Files\ImmichClone\immichclone.exe`). Custom installations may differ. Choose a maintenance window: the current prototype instructions require stopping the service for restoration. Browsing/import/AI will be unavailable while it is stopped.

No recovery drill is requested from the user at this checkpoint. The following is a manual procedure for when a drill is wanted; replace all example paths first. Run in Windows PowerShell as administrator when stopping the installed service requires it. Do not put your archive, access code, filenames or output in public repository issues.

## Check the downloaded archive

```powershell
cls
$app = 'C:\Program Files\ImmichClone\immichclone.exe'
$archive = 'E:\PrivateBackups\library.zip'
$target = 'D:\RecoveryDrill\restored-library'
$expectedSHA256 = 'PASTE_THE_64_CHARACTER_BACKUP_SHA256_HERE'

if (-not (Test-Path -LiteralPath $app -PathType Leaf)) { throw 'Executable not found' }
if (-not (Test-Path -LiteralPath $archive -PathType Leaf)) { throw 'Archive not found' }
if ($expectedSHA256 -notmatch '^[a-fA-F0-9]{64}$') { throw 'Enter the saved archive SHA256 first' }
if (Test-Path -LiteralPath $target) { throw 'Recovery target must not exist' }
$parent = Split-Path -Parent $target
if (-not (Test-Path -LiteralPath $parent -PathType Container)) { throw 'Create only the recovery parent folder first' }
$before = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash
if ($before -ne $expectedSHA256) { throw 'Archive hash mismatch: stop here' }
& $app -mode version
if ($LASTEXITCODE -ne 0) { throw 'Cannot read application version' }
```

A matching downloaded-file hash checks the transfer against the saved value. The restore operation additionally validates archive structure, payload checksums and catalog relationships. It never overwrites an existing target.

## Restore and verify without switching the live library

Continue in the same PowerShell window after checking the version. The service is restarted in `finally` only if it was running beforehand; restarting it continues to use the **original** library.

```powershell
$service = Get-Service -Name 'ImmichClone' -ErrorAction Stop
if ($service.Status -notin @('Running', 'Stopped')) { throw 'Wait for the service transition to finish first' }
$wasRunning = $service.Status -eq 'Running'
try {
    if ($wasRunning) {
        Stop-Service -Name 'ImmichClone' -ErrorAction Stop
        (Get-Service -Name 'ImmichClone').WaitForStatus('Stopped', [TimeSpan]::FromSeconds(30))
    }
    & $app -mode restore -archive $archive -data $target
    if ($LASTEXITCODE -ne 0) { throw 'Restore failed: retain the archive and inspect the private diagnostics' }
    & $app -mode verify -data $target
    if ($LASTEXITCODE -ne 0) { throw 'Restored originals failed verification' }
    if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash -ne $before) { throw 'Archive changed during the drill' }
    Write-Output 'PASS: restored originals verified; installed library was not switched'
} finally {
    if ($wasRunning) { Start-Service -Name 'ImmichClone' -ErrorAction Stop }
}
```

`verify` proves the catalog/original checksum relationships it checks. It is not a browser, AI, face-assignment or full functional acceptance test. Restored previews are derived data and regenerate when the restored application is run. The saved access code belongs to that restored library; keep it private.

## Refusal, interruption and recovery

- A nonzero exit is a failure even if some private staging files appeared. Do not treat a partial directory as a restored library. Do not copy individual catalog or original files into the live library to work around a refusal.
- For a normal cancellation, use Ctrl+C and wait for the command to return. The Windows extraction-cancellation test proved supervisor cleanup and a fresh independent retry; it did not prove arbitrary controller crashes, power loss or physical storage removal. Do not close the PowerShell window to simulate those failures on your own library.
- If cleanup is reported incomplete, keep the ZIP and existing library, and request help before deleting unfamiliar folders. If the target exists after interruption, do not rerun into it or blindly remove it. Check whether publication completed and verify its contents first.
- A budget/space/timeout refusal is not permission to disable checks or blindly raise limits. Preserve the original error privately and establish the required capacity/workload first. The known intermittent progress-read:32 issue remains unexplained and deferred; repeated success is not a fix.
- Switching the installed service to a recovered library is a separate maintenance operation. This guide does not change registry state, service configuration, folder ACLs or installer data location. Keep the current library and backup intact until that activation procedure is explicitly validated for the intended environment.

## Existing evidence and remaining limits

Actual Windows Server 2022 evidence: [installed backup and independent restored reinstall](hidden-media-milestone.md#0136-combined-installed-batch-acceptance-and-publication), [65,536-original ZIP64 entry-count restore](backup-zip64-entry-count-acceptance.md), [archive above 4 GiB](backup-large-volume-acceptance.md), and [mid-extraction cancellation/cleanup/retry](restore-interruption-acceptance.md). These tests exercise the underlying operations; this exact manual guide has been reviewed against the CLI and packaging sources, not run as a new end-to-end user-PC drill.

Windows 10 clean-machine/reboot acceptance, natural full-library endurance, physical-storage outage and controller-crash recovery remain unproven. Incremental/scheduled backups and every-frame Full Scan remain deferred. Do not delete the existing library or your independent archive on the strength of these bounded prototype results.
