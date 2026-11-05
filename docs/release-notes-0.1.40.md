# Immich Clone 0.1.40 prototype

This prototype delivers the accepted existing-function UI redesign and installer/
recovery hardening since 0.1.39. It is intended for a focused update validation.
It does not finish milestone 7 or certify final Windows 10 acceptance.

- Clear desktop/narrow navigation for Library, Hidden Media, People & tags,
  Bulk import, Backups and Build Notes; search handoff and existing task panels.
- Safer installer filesystem handling, private backup/restore operations and
  separate-account metadata/namespace protections. Unsafe writable parent
  locations are refused; existing security checks must not be bypassed.
- Persistent interrupted-upgrade recovery, stopped-service protection during
  mixed copies and checked rollback/retry. Independently restored libraries can
  be enrolled into the service while preserving private storage and original bytes.

The release gate uses the genuine checksum-verified published 0.1.39 installer,
full existing pinned media/AI providers, failed-upgrade rollback, original and
organization preservation, actual browser/native processing, and independent
restore/reinstall with the old state unavailable. Prior accepted source/evidence:
[SEC03 combined checkpoint](https://github.com/aataat12/immich-clone/blob/7b2c22dfd9baefc90b81e1f9bfe66f857bcfbd61/docs/sec03-combined-acceptance.md).

Download this release's installer, SHA256SUMS.txt and validation-0.1.40.md. Verify
the installer checksum, retain an independent Ready backup and its recovery
instructions, then upgrade with the same application/data folders. No development
tools or routine reboot are required. The installer remains unsigned; do not
disable Windows security protections if it is blocked.

Validation instructions for the validation AI are in validation-0.1.40.md.
Curation/Cinema, durable global review counts/decisions, library-wide server
filters, Full Scan and RAW/HDR expansion remain deferred. No provider/model,
catalog schema, import limit or restore budget is changed in this release step.

Original security progress is 5 closed / 3 open: SEC-05 native worker isolation,
SEC-06 remaining input review and SEC-08 final target/release acceptance remain.
Windows Server testing does not certify Windows 10 clean-machine prerequisites,
reboot/startup, multi-day endurance or physical power-loss recovery. Keep private
media, access codes, backups, diagnostic logs and machine reports out of public
issues and artifacts.
