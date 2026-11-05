# 0.1.41 corrective installer acceptance

0.1.40 target update validation found a preparation compatibility defect: normal
Windows ProgramData parent creation/attribute grants were rejected as replacement
rights. The refusal occurred before payload replacement and left the retained
prior service stopped and disabled. Private support evidence remains local and
is not a CI input or public attachment. 0.1.40 is superseded by this correction.

0.1.41 narrows the parent refusal to replacement/deletion/permission takeover,
checks owner and ACE forms, keeps no-follow identity and private-child policies,
audits namespace parents before service/ACL writes, and anchors fresh private
state before later actions. Existing application root trust is checked after
native open even if another actor created the name first. No parent permissions
are changed to make Setup pass. No dependencies, schema or feature scope changes.

Validation is pending until exact focused and full Windows runs below pass.
The focused gate adds creation-right and replacement-right native controls,
a real standard-account sibling-create/rename versus private-child refusal,
and fresh actual Program Files/ProgramData installation to all existing races,
interrupted publication/retry and unsafe-location controls. The full gate uses
genuine published 0.1.39 in actual default parents, stops/disables that owned
service, then proves rollback and corrected upgrade with full pinned providers,
original/organization preservation and independent restore/reinstall.

References for access semantics: [file rights](https://learn.microsoft.com/en-us/windows/win32/fileio/file-access-rights-constants),
[file security](https://learn.microsoft.com/en-us/windows/win32/fileio/file-security-and-access-rights),
and [nonempty directory reparse refusal](https://learn.microsoft.com/en-sg/windows-hardware/drivers/ifs/fsctl-set-reparse-point).
Windows Server evidence cannot certify final Windows 10 or physical power-loss
acceptance. Target-PC validation remains pending after release publication.
