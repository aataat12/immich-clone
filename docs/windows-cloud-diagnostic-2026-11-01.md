# Windows cloud diagnostic — November 1, 2026

Result: **Passed**.

[Workflow run](https://github.com/aataat12/immich-clone/actions/runs/41320566461)

Tested commit: `0a55c52a137ed844f74aa70f143669c2e27c97ba`.

## Verified

- Workflow file creation through connected GitHub tools and automatic execution on push.
- Windows Server 2022 Datacenter, OS version 10.0.20348, administrator privileges.
- Compilation and self-contained win-x64 publication of a small .NET 8 HTTP application.
- Native Windows service registration, startup, HTTP health check, service restart, and health recheck.
- Initialization of a separate PostgreSQL 14.24 database cluster using runner-provided binaries; registration and startup as a Windows service.
- SQL table creation and insert, database service restart, and successful read of the persisted value.
- Cleanup: both diagnostic services stopped and deleted successfully.
- Report upload, artifact listing, authenticated artifact retrieval, and ZIP/result.json reading from the cloud workspace.

Artifact: `windows-diagnostic-report`, ID `13089081612`. Retention: 14 days. This document retains the outcome after artifact expiry.

## Limits

This proves the cloud control, native Windows execution, and report download path. It does not validate the photo application, finished installer, dependency installation on a clean machine, .NET 10, a final PostgreSQL version, ONNX, FFmpeg, image libraries, Windows 11 desktop behavior, GPU support, full OS reboot recovery, upgrades, backups/restores, or endurance. The runner already provided PostgreSQL; its installation fallback was not exercised. The diagnostic uses disposable loopback-only trust authentication and LocalSystem services; these are not production configuration choices.

Billing/remaining quota was not accessible. Successful execution confirms this job was allowed, not future unlimited capacity. The repository is public. No local PC components were installed.

## Next engineering milestone

Choose and pin supported production dependency versions, build a minimal photo upload/storage/thumbnail application and real installer, then validate dependency provisioning and application behavior on Windows cloud runners. Keep mobile features out of scope and complete cloud validation before asking the user to install prerequisites locally.
