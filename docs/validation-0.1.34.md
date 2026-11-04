# 0.1.34-prototype: target-PC validation

Use only after the tested installer is published. These checks supplement cloud Windows Server evidence; they do not establish clean-machine, reboot or long-term Windows 10 acceptance.

1. In 0.1.33, create and download a backup. Upgrade with the same application and data locations. Confirm **0.1.34-prototype** appears and existing photos/videos, names, tags, AI results and import history remain accessible.
2. Choose an existing item with review data. Hide it, open **Hidden Media**, then **Managed file location**. Check that Copy path identifies its managed original under the application data folder. Close the dialog and choose **Restore to library**. Confirm its original, names, tags and results remain available after refresh.
3. For permanent deletion, use only a disposable managed item whose external source you still have. Hide it and choose **Permanently delete managed copy**. A wrong confirmation must leave deletion disabled. Type **DELETE**, choose Cancel and confirm the original remains downloadable. Open the dialog again, type **DELETE** and confirm. After refresh, the item should be absent from the library and Hidden Media, and its external source should still exist. Completed deletion has no Undo; earlier backups remain separate recovery sources. Do not manually remove files inside managed storage.
4. Sign out, reject an incorrect access code, then unlock with the existing code. Refresh and visit Photos & videos, People & tags, Bulk import, Backups and Build Notes. Confirm milestone 3 is listed in 0.1.34; milestone 4 remains in progress and Full Scan remains deferred.
5. If practical, narrow the browser window and check Hidden Media/dialogs for overflow. Report this as NOT TESTED if unavailable.

Return PASS / FAIL / NOT TESTED for each numbered step. If a step fails, give the action and displayed message; download diagnostics for private review. No forced reanalysis, large import, full-library deletion, or real disk-filling exercise is required on your PC.
