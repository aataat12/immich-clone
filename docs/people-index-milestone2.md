# Milestone 2: People review scale and functional acceptance

This closes the remaining review/search performance work for the current functional scope. Earlier false-face dismissal, confirmed suggestions, independent paging, photo context and additive reanalysis remain delivered features. The [companion native-quality evaluation](native-face-quality-milestone2.md) passed its declared production-pipeline screen in run41767731927. Combined installed acceptance still gates delivery.

Original Group 2/3 security work remains deferred under the user's existing priority amendment. Functional milestone delivery does not close those original security items or establish comprehensive security/Windows 10 certification. See security-priority-groups.md. No new security item, provider/model, threshold, dependency, catalog format or resource-limit change is introduced by this index.

## Behavior and correctness

People pages previously reloaded the entire asset catalog and every face embedding on each request/poll. A Store-local immutable derived index now tracks the committed bbolt transaction ID. An unchanged committed snapshot reuses the index; every successful write transaction invalidates it automatically. A failed transaction leaves authoritative state unchanged. Reopening, older-writer rollback and independent restore create a fresh Store/index rather than retaining stale cached data. The index is not serialized or included in backups.

The index retains lightweight asset metadata, ordered face records and normalized searchable filename/person/custom-tag/visible-AI-tag terms. Only active named reference embeddings stay resident; unnamed embeddings are read for requested page suggestions. Face cursor lookup uses binary search. Asset cursor lookup keeps the existing bbolt seek; substring matching checks cached terms before decoding selected records. Search is still a substring scan of cached terms, not a claim of sublinear arbitrary substring search or approximate vector indexing. Names/references still apply across all visible active named faces.

Responses remain capped at 100 faces/50 photos. Unnamed/dismissed/search filters apply before continuation, exact-last-page searches have no phantom Next, separate previous/next histories remain intact, and hidden/dismissed references do not leak into active suggestions. Suggestion calculation was extracted without changing the existing compatible-provider cosine threshold .363 or .05 runner-up margin. Naming continues to require user confirmation. Cached response names/boxes/timestamps are returned independently so callers cannot mutate derived state through a returned view.

First request after a committed write still rebuilds the index in O(catalog size) time. The derived resident cache grows with library metadata and named reference embeddings; no universal memory/latency bound is claimed. Native matching remains linear in named references for requested unnamed faces. These limits are explicit rather than presenting finite scale evidence as unlimited scalability.

## Executed evidence

[Windows run41767370225](https://github.com/aataat12/immich-clone/actions/runs/41767370225), source `e6f85932bbfc8b32cd28eada06764136b28c5217`, job `126368815642`: PASS first execution, Windows Server2022, Go1.27.3 windows/amd64, CGO disabled and GOAMD64=v1.

- Scale fixture physically wrote **10,000 distinct tiny originals** and directly seeded valid catalog metadata with **30,000 faces**, including 10,000 finite unit-vector native-compatible records and 100 named references. This measures review/query performance; it does not claim 10k UI import throughput or native inference on those synthetic vectors.
- Cold query **246.8991ms**. Eight warm queries: maximum **1.1012ms**, mean allocated bytes/query **333,961**. Pre-existing test budgets were cold<=20s, warm<=1s, allocated bytes/query<=8MiB. These are observed disposable-runner results, not a hardware-independent performance guarantee; total resident/peak memory was not measured.
- Independent face/photo cursor pages, exact two-page 100-match case-insensitive tag search with no duplicate/phantom continuation, empty search, and every physical original's independent source bytes/ID passed.
- Named correction, rename, undo, Unicode case-insensitive custom-tag search, AI-tag dismissal, dismissed-face/reference separation, hide/restore, returned-view independence and independent backup restoration with old state offline passed. Existing organization/face/tag/reanalysis/HTTP regressions and vet passed.
- Actual Edge app created 55 distinct PNG originals and 102 manual faces through the normal HTTP APIs. Separate face/photo next/previous identity, name/tag search invalidation, unnamed/dismissed changes and restore, reload, stable background, desktop/390px layout and every downloaded original hash passed with no page JavaScript errors. Native inference is covered by the companion quality/installed tests, not this manual-face navigation fixture.

Local correction/restore tests and vet passed; local corresponding scale observed cold249.55ms, warm maximum3.01ms and334177 allocated bytes/query before Windows execution. Focused checks were not repeatedly rerun without changes.

Harnesses: internal/library/organization_index_test.go, tests/people-index.cjs, .github/workflows/people-index.yml. All data is disposable synthetic/public. Final target Windows10 clean-machine/reboot/endurance acceptance remains separate. Every-frame Full Scan stays deferred.
