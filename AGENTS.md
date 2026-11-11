# Working instructions

Read `Immich_Clone_Project_Requirements.md`, then `docs/status.md`,
`docs/decisions.md`, and `docs/dependency-evidence.md` before work.
The requirements are authoritative. Parallel agents require explicit user authorization. No new charges.
Record actual environment and evidence; Server results never certify Windows 10.
Keep private PC reports out of this public repository and Actions artifacts.
Never publish the user's exact OS build, hardware models, memory, GPU/driver
details, disk inventory/free space, personal filesystem paths, enrollment state,
or raw machine reports. Do not echo those values into CI. Public docs may state
generic product compatibility requirements and disposable-runner evidence only.
Experiments are not production components. Keep downstream work gated on evidence.

Never commit user-returned support logs. Keep private diagnostic files out of
GitHub, CI inputs and public issue attachments; only sanitized, generic findings
may enter project records. For tool-based source uploads, verify each blob SHA
against the local Git object hash before moving a branch ref.

Security changes: if a mitigation conflicts with the agreed project plan, explain
the conflict and ask the user before implementing that conflicting change.
Continue independent work that does not depend on the decision.

## Release production

Release production is owned by the persistent `Immich Clone — Release Production`
discussion. Feature, UI, and research discussions must hand off a candidate there
with its source commit, branch, validation evidence, and known limitations.

Before release work, read `docs/release-runbook.md` and
`docs/release-state.json`. The supported route is cloud-based: GitHub Actions
builds and validates the Windows installer; GitHub REST inspects runs, jobs,
artifacts, tags, and releases. GitHub CLI installation is not required.

If connector writes are unavailable, use the bundled Git executable and the
existing Windows Credential Manager authentication in the appropriate user
context. A restricted-shell credential error alone is not proof that GitHub
access is unavailable. Never print, persist, or commit credentials.

Do not publish until the exact tested source commit and installer artifact have
been verified. Update `docs/release-state.json` with sanitized release evidence.
