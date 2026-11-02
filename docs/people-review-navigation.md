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

Local JavaScript syntax, integrated Go library/web/cmd tests and vet passed.
Actual Windows Server 2022/Edge acceptance [41547447887](https://github.com/aataat12/immich-clone/actions/runs/41547447887), source
1f1cdb912b223634dec0350e58e9ae86b11893b0, job 125626387613, passed in full. The browser
probe created 102 actual manual face records on a synthetic PNG and 52 accepted
small synthetic originals. It verified independent next/previous histories, prior
page identity, cursor/filter reset and selection clearing. Existing five-page
desktop/narrow, CSP, session, photo context and correction controls also passed.
No user media, forced local import, native provider or catalog schema changed.
Hold for the broader combined release; this is not Windows 10 certification.
