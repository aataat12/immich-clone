# Managed-copy bulk import acceptance

Deferred by user: people/object identification and organization feasibility now
takes priority. Retain this plan for later; no additional import runs scheduled.

Primary storage remains application-owned immutable copies. Sources are never
moved, rewritten or deleted. Identical SHA256 content reuses a managed original;
relative source aliases are retained. Previews are disposable derived data.

## Small milestones

1. Queue foundation: transactional pending-job index rebuilt from authoritative
   asset states on open; interrupted work/retry/restore/downgrade recovery retain
   jobs. Compare bounded metadata-only Windows measurements and execute a real
   PNG job, then full service/installer/independent restore regressions.
2. Import-first scheduling: authenticated, bounded transfer session suspends new
   previews while copying. Expiry/service restart cannot strand paused processing.
   Cancellation retains committed originals and queued jobs; byte acceptance is
   distinct from preview completion. Test interruption/concurrency/recovery.
3. Dedicated bulk page: deliberate copy confirmation, files/folder selection,
   byte/file progress, duplicates/failures and cancel/retry. Avoid gallery polling
   and rendering during transfer. Closing the browser requires source reselect;
   reconcile durable accepted files without falsely claiming an unreceived file.
4. Scale acceptance: actual synthetic media ingestion (not metadata-row counts),
   interrupted transfers/restart/disk-full, representative mixed media, bounded
   gallery paging, endurance, independent backup/restore and original SHA256.
   Only then invite a large real-library import. Keep synthetic inputs public.

Browser upload currently limits each file to 4 GiB. That is an explicit limitation
until a tested larger-file path replaces it. Backup currently stages a complete
archive inside application state; large-library backup needs capacity/progress
review. Neither limit is cleared by queue benchmarks or available source space.

No new dependencies, paid resources or local setup are needed for this plan.
Final OS reboot validation remains deferred by the user.
