# Hidden Media: accepted feature, implementation pending

User requested a Hidden Media page with leave-hidden, restore and permanently
remove controls, plus each file's directory for manual investigation. AI remains
the current implementation priority; this specification does not claim new UI.

## Behavior and preservation boundary

- Open Hidden Media from library navigation. Leaving an item hidden requires no
  action; restore returns it to the visible library with metadata/jobs intact.
- Show the managed original's full directory and filename, with a copy-location
  control. Show recorded relative source aliases separately. Browser folder
  selection does not expose the original source's absolute directory; never
  invent it. Absolute paths are computed locally, authenticated and never put
  in diagnostic events, public records or CI inputs from a personal library.
- Permanent delete is available only for hidden assets and requires explicit
  confirmation identifying the item. Clearly say it removes the application's
  managed original, previews, AI results and catalog relationships. Never remove
  external source files, grant source permissions or accept an arbitrary path
  from a deletion request. Duplicate source aliases may refer to one managed
  content blob; confirmation must make the stored-copy scope clear.
- Existing downloaded backups remain independent copies; deleting live media
  does not erase those archives. A restore of an earlier backup can bring the
  asset back. Do not claim forensic secure erasure or removal from external sync.

## Acceptance before delivery

Authenticated same-origin/CSRF-protected writes, validated asset identity and
hidden-state recheck under serialization. Visibility refreshes cannot authorize
another item. Deletion must coexist safely with jobs, duplicate import and backup.
Use durable deletion intent/recovery so service failure or file-access failure
cannot report success with unfinished deletion or leave runnable jobs referencing
removed originals. Retain actionable retry status without logging personal paths.

Synthetic checks: leaving hidden/restoration, denied unauthenticated/cross-origin
writes, refusal to delete a visible asset, confirmation scope, retained external
source bytes, actual managed-file removal, preview/AI cleanup, permission failure,
crash/restart recovery, concurrent job/backup behavior and independent restoration.
Browser checks must inspect directory display without publishing a personal path.
No deletion of user media is performed by this feature specification or CI tests.
