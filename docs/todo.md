# Project to-do list

## Release 0.1.43 planning

Development checkpoint: the three items are accepted scope for 0.1.43. Candidate
date/gallery search passed the Windows feature gate; automatic assignment remains disabled until
its independent quality/compatibility gates pass. No released behavior is claimed.

All three improvements below were identified in the November 5, 2026
**Release 0.1.42 discussion** and are confirmed backlog items. They are not yet
released; implementation and acceptance are tracked in release-0.1.43-development.md.
See the [documentation research and prioritized candidates](release-0.1.43-feature-research.md)
for the wider feature review. Newly researched candidates are proposals, not an
instruction to implement all of them in 0.1.43.

## Clearer photo dates

- Show **Date taken** when a usable capture date is available and **Date imported**
  otherwise. Do not present an import date as the date a photo was taken.
- Move timezone detail into the photo information panel. Distinguish an absent
  timezone from a timezone the app has not read; do not invent a timezone.
- Keep the original embedded metadata intact. Broader metadata extraction and
  user date corrections are separately estimated research candidates.

## Search within the photo gallery

- Searching in **Photos & videos** filters that gallery without navigating to
  People & tags. Keep normal gallery cards and viewing behavior.
- Match assigned person names, filenames, and custom/non-dismissed object tags.
  Unconfirmed face suggestions must not count as assigned identities.
- Keep the query, matching result count, and clear-filter control visible.
  Preserve the filter through pagination and returning from a photo.
- A photo with multiple assigned people must be discoverable under each name.
  Keep naming/review and tag editing available separately in People & tags.
- Search the whole eligible catalog, not just the currently loaded page, and
  preserve existing hidden-item rules.

## Automatic identification of known people

Requested November 5, 2026, during the 0.1.42 discussion. Assigned to 0.1.43. Candidate mechanisms and a narrow native policy screen passed;
production assignment remains disabled pending broader quality and compatibility evidence.

- Use manually confirmed faces as references to automatically name strong,
  unambiguous matches in newly imported and analyzed photos.
- Keep uncertain matches in the review queue instead of requiring confirmation
  for every recognizable face indefinitely.
- Distinguish automatic assignments from manually confirmed names, allow easy
  correction, and preserve explicit user corrections.
- Validate an automatic-assignment policy separately from the current suggestion
  threshold; existing review-only quality evidence is insufficient for automatic
  naming. Prevent incorrect automatic assignments from reinforcing themselves.
- Support multiple people per photo: each identified face retains its person
  association, and the photo can be found under each person's name.

This records future feature scope; 0.1.42 remains review-only.
