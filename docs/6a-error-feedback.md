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

Windows Server2022/LocalService/Edge execution passed in
[41583936189](https://github.com/aataat12/immich-clone/actions/runs/41583936189),
source222ec96387f6a7bad9cc1aae33b6040d65c25eda, job125744900977. Synthetic
first-failure/retry, actual native exit17, visible/disabled first-page Previous
controls, public HEIC/video inference, provider/storage failure, service crash
and independent restored crop/context regeneration passed. The first run
41583371739 timed out because its name-text selector could match a different face
with a suggested name; the test now selects the exact saved face ID and requires
the saved name before opening its sampled-frame context. No inference assertion
was removed. Local focused tests, JavaScript syntax and Windows cross-build passed.

This is not an installer release or an assertion
that every possible native inference failure has been explained or fixed.
