# People review filtering before pagination — candidate

Unnamed only now filters on the server before the 100-face page is selected. A
page of named faces can no longer hide later unnamed faces. Changing the checkbox
clears the face cursor and selection. Search and dismissed/active filters are
applied before deciding whether another matching page exists; exactly 100 matches
followed by nonmatches no longer advertise a phantom Next faces page.

Named-reference suggestions and the name picker still use all active named
references, even while the face response is filtered to unnamed results. No model,
original, catalog format, correction or AI-job change. Existing API callers keep
their default unfiltered behavior. Response page limits remain 100 faces/50 photos.

Linux Go 1.27.3 focused race tests passed using 207 metadata-only face records on
one actual synthetic original. They prove named prefixes do not hide later unnamed
records, continuation is complete/unique, dismissed state stays separate and exact-
limit/empty searches have accurate continuation. HTTP filtered-response test, full
library/web/cmd tests and vet passed. Actual Windows/Edge acceptance is pending.
These rows do not establish native inference, large-library ingestion, bounded
whole-query memory or broad recognition accuracy. Organization still loads/scans
catalog and reference records; indexed People performance remains open.

Hold for the next broader release. Security worker experiments and unresolved
installer/containment gates remain independent; no local import or installer yet.
