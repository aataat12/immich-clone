Windows x64 prototype maintenance release for6A feedback and navigation.

- Corrected the People help text to describe supported HEIC and sampled-video AI.
- Preserve a successfully probed video sampling plan after a first-sample error; distinguish unknown coverage from known0/12 progress.
- Report face/object task failures separately and preserve numeric native exit codes in fixed-schema diagnostics. Child stderr and media paths are not added to diagnostic records.
- Show Previous faces/photos on the first page, disabled until forward navigation.
- Retain existing native/media providers, resource limits, ready results, names/tags and manual retry behavior. Every-frame Full Scan remains deferred.

This improves feedback and diagnosis; it is not a guarantee that every native inference failure has been resolved. No automatic reanalysis is performed.

Upgrade the existing installation using the same application and data locations. Keep a verified backup before upgrading. No separate FFmpeg/Python/SDK installation is required.

Unsigned early prototype. Cloud Windows Server2022 validation does not certify Windows10. Target Windows10 acceptance, clean-machine/reboot and extended reliability checks remain separate.
