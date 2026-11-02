# Photo context during face review — candidate

Active and dismissed face cards provide View photo. The dialog loads the existing
authenticated image-view endpoint, displays the recorded filename as text and
links the untouched original download. Close/Escape removes the image source and
link; automatic review refresh pauses while the dialog is open. A failed image
request shows a fixed fallback with the original link retained. No AI job, name,
dismissal, tag, original or catalog change is performed by opening the view.

The existing image endpoint retains its supported-format behavior; this does not
add HEIC/video AI, a new decoder or a promise of full-resolution TIFF rendering.
The image has no newly inferred face overlay or invented source location.

JavaScript syntax and existing CSP/full-image HTTP controls are checked locally.
Actual Edge desktop/narrow open/load/close/Escape, dismissed-view and failed-image
fallback checks passed. Hold for the next broader release with security gates.


Accepted Windows Server 2022/Edge run [41535508518](https://github.com/aataat12/immich-clone/actions/runs/41535508518),
source 77d1f1b87411b0ed51834bd054ef2603c616ea0f, job 125586209991. Actual desktop/narrow photo load,
filename/original link, Escape cleanup, failed-request fallback and dismissed-face
context passed. Existing unnamed/dismissal/session/CSP, Go/vet, full SEC-01 and
actual Setup/service checks passed. The previous run 41534740456 passed security
and Setup but the new UI test asserted before the asynchronous close event; the
retest waits for actual event cleanup instead of weakening the assertion. No
application data/decoder change, installer release or Windows 10 claim.
