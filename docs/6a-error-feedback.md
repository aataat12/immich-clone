# 6A candidate error feedback and navigation

This candidate changes feedback and diagnostic coverage; no native binary/model
or resource limit changes. No automatic retry or forced reanalysis.

- Keep a successfully probed sampling plan when the first controlled inference
  fails. Zero completed samples then displays0/12 rather than losing known targets.
  A clip whose plan could not be probed displays an explicit unavailable plan.
- Distinguish face/object child task failures in the UI and fixed-schema diagnostic
  stages. Preserve only a numeric native exit code; do not publish child stderr,
  model paths, media identifiers or image content. Successful task records follow
  response validation, so rejected responses are not logged as task success.
- Correct the HEIC/video help to describe supported8-bit HEIC and explicit sampled
  video processing. Every-frame Full Scan remains deferred.
- Show Previous faces/photos on the first page, disabled until forward history
  exists. Preserve independent cursors, filters and selection behavior.

Verification: synthetic first-inference failure/retry regression; privacy-safe
numeric child-exit classification; actual Windows child exits17; Edge first-page
control visibility; existing public-fixture LocalService inference, provider/
storage failure, native-child crash and independent restore suite.

Windows execution is pending. This is not an installer release or an assertion
that every possible native inference failure has been explained or fixed.
