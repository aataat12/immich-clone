# Working instructions

Read `Immich_Clone_Project_Requirements.md`, then `docs/status.md`,
`docs/decisions.md`, and `docs/dependency-evidence.md` before work.
The requirements are authoritative. No parallel agents. No new charges.
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
