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
fallback checks are pending. Hold for the next broader release with security gates.
