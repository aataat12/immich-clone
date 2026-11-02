# Previous-page People review navigation — candidate

People & tags adds Previous faces and Previous photos, with separate in-memory
cursor histories. Returning to a prior page retains the current search/filter.
Switching unnamed/dismissed face filters clears the face history/selection; a
new search or Clear resets both histories. Pending navigation suppresses polling
and duplicate page clicks; a failed request restores its prior cursor/history
unless the user has changed the filter. No naming, AI or original mutation.

The existing server still returns at most 100 faces/50 assets; no API/schema/model
or dependency change. This is navigation correctness, not indexed People query
performance or broad-library ingestion acceptance.

Local JavaScript syntax and web/cmd tests passed before integrating worker restore
changes; final integration/vet and actual Windows/Edge are pending. The browser
probe creates 102 actual manual face records on a synthetic PNG and 52 accepted
small synthetic originals, exercises independent next/previous histories, cursor
reset and selection clearing, and verifies prior-page identity. No user media or
forced local import. Hold for the broader combined release.
