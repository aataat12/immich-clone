# Reversible face dismissal — next-release candidate

People review offers Not a face and a separate Dismissed faces view with Restore
face. Dismissal preserves the stored detection/crop, name and name-undo history.
It removes the face from active reference suggestions, the active name list and
name-based photo search; restore returns it without rerunning AI. Custom tags,
ready analysis, manual geometry and original bytes remain independent. Both
detected and manually marked faces can be corrected this way.

An additive dismissed boolean lives in the existing face-label record; no new
bucket or original/backup format. Consistent catalog backup and embedding-free
JSON export retain the flag. Older executables do not implement dismissal and
may drop this additive field if they edit the label; no cross-version correction
editing guarantee is claimed. Upgrade rollback remains gated on installer tests.

Linux Go 1.27.3 focused race tests passed: named-reference positive control,
dismissed reference exclusion, preserved names/undo across reopen and independent
backup restore, tags/ready AI/original verification. HTTP tests reject missing
authentication, foreign origins and unknown input fields; valid dismiss/restore
and separate review work. Full library/web/cmd tests and vet passed. Windows/Edge
acceptance is pending. No installer/version change or local import requested.

Remaining milestone work: safe ready-analysis reruns, larger review/search
performance and broader recognition quality. Security worker source reviews remain
release gates; a new feature does not clear or defer them silently.
