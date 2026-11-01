# Recommended partner workflow

Use independent Work sessions with each collaborator's own account/GitHub
connection. Coordinate reviewed code through this repository, scoped branches
and pull requests. Do not assume a shared conversation shares tools, local files,
credentials, state, or an automatic task/usage pool. The project owner grants
repository access separately; this document sends no invitation or assignment.

Read, in order: `AGENTS.md`, `Immich_Clone_Project_Requirements.md`,
`docs/status.md`, `docs/decisions.md`, `docs/dependency-evidence.md`.
The requirements and latest recorded evidence supersede conversational history.
Avoid copying the private project conversation: it contains private machine data.
No personal inventory, source paths, logs, access codes or media belong in GitHub.

Recommended coordination: this project's primary session retains integration and
release ownership. Partner work uses `partner/<task>` branches and PRs; no direct
main pushes or releases. Agree one bounded task before changing shared files.
No new charges, OS switch, local dependency setup requests, source deletion, or
unsupported compatibility claims. Reuse verified results and synthetic fixtures.
No spawned parallel agents under the current repository instructions.

## Suggested first task: independent restore failure-path review

Branch: `partner/restore-review`, from the latest main after 0.1.9 acceptance.
Review `internal/library/backup.go`, existing backup/fault tests, and restoration
requirements. Find uncovered correctness/recovery risks using disposable synthetic
inputs. Do not repeat already-covered tests just to restate passing evidence.

Deliver a PR containing concise findings and, where a genuine gap is demonstrated,
a focused regression test in a new `internal/library/partner_restore_test.go` file.
Each finding needs expected/actual behavior, synthetic reproduction, severity and
exact tested commit/environment. A clean review with no reproducible defect is a
valid outcome. Avoid speculative vulnerability claims.

Leave implementation, dependencies, schema, originals layout, installer, media
provider and release workflows unchanged for this first task. Run focused tests
appropriate to the review; no repeated full installer runs without a reason. Ask
the primary session to review the PR before taking an overlapping implementation
task. This handoff does not establish automatic cross-account communication or
background coordination; bring the PR link back to the project conversation.

Current delivered build: 0.1.11; Server acceptance run 41352572001.
Local MOV disappearance remains unconfirmed; current status records the evidence
needed. Do not conflate the new visibility controls with a fix for that report. Local evidence
for 0.1.3/0.1.5/0.1.7 is user-reported; new media provider local checks and final
reboot remain separate. Never infer Windows 10 compatibility from Server results.
